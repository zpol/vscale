package allocator

import (
	"fmt"
	"net"
	"sync"

	"github.com/zpol/vscale/internal/models"
	"gorm.io/gorm"
)

const (
	// DefaultGlobalPool is RFC1918 space reserved for vscale overlays.
	// Intentionally NOT 100.64.0.0/10 — that range is used by Tailscale and
	// installing a 100.64/16 AllowedIPs route blackholes Tailscale + often "internet".
	DefaultGlobalPool = "10.66.0.0/16"
	// DefaultVscalePrefixLen is the size reserved per vscale inside the global pool.
	DefaultVscalePrefixLen = 24
)

var privateWarnings = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
}

type Conflict struct {
	CIDR      string `json:"cidr"`
	Conflicts string `json:"conflicts"`
	WithKind  string `json:"withKind,omitempty"`
	WithCIDR  string `json:"withCidr,omitempty"`
	Severity  string `json:"severity"` // error | warning
}

type Result struct {
	CIDR      string     `json:"cidr"`
	Conflicts []Conflict `json:"conflicts,omitempty"`
}

type Allocator struct {
	db               *gorm.DB
	globalPool       *net.IPNet
	vscalePrefixLen  int
	mu               sync.Mutex
}

func New(db *gorm.DB, globalPoolCIDR string, vscalePrefixLen int) (*Allocator, error) {
	if globalPoolCIDR == "" {
		globalPoolCIDR = DefaultGlobalPool
	}
	if vscalePrefixLen == 0 {
		vscalePrefixLen = DefaultVscalePrefixLen
	}
	_, pool, err := net.ParseCIDR(globalPoolCIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid global pool: %w", err)
	}
	ones, bits := pool.Mask.Size()
	if vscalePrefixLen < ones || vscalePrefixLen > bits {
		return nil, fmt.Errorf("vscale prefix length %d out of range for pool %s", vscalePrefixLen, globalPoolCIDR)
	}
	return &Allocator{db: db, globalPool: pool, vscalePrefixLen: vscalePrefixLen}, nil
}

func parseCIDR(s string) (*net.IPNet, error) {
	ip, n, err := net.ParseCIDR(s)
	if err != nil {
		// allow bare IP as /32
		ip2 := net.ParseIP(s)
		if ip2 == nil {
			return nil, err
		}
		if ip2.To4() != nil {
			return &net.IPNet{IP: ip2.To4(), Mask: net.CIDRMask(32, 32)}, nil
		}
		return &net.IPNet{IP: ip2, Mask: net.CIDRMask(128, 128)}, nil
	}
	_ = ip
	return n, nil
}

func overlaps(a, b *net.IPNet) bool {
	return a.Contains(b.IP) || b.Contains(a.IP)
}

func (a *Allocator) listAllocations(vscaleID *uint) ([]models.IpAllocation, error) {
	var allocs []models.IpAllocation
	q := a.db.Model(&models.IpAllocation{})
	if vscaleID != nil {
		// global overlays (vscaleID nil) + this vscale
		q = q.Where("vscale_id IS NULL OR vscale_id = ?", *vscaleID)
	}
	if err := q.Find(&allocs).Error; err != nil {
		return nil, err
	}
	return allocs, nil
}

func (a *Allocator) CheckOverlap(cidr string, vscaleID *uint, localCIDRs []string) []Conflict {
	n, err := parseCIDR(cidr)
	if err != nil {
		return []Conflict{{CIDR: cidr, Conflicts: "invalid CIDR", Severity: "error"}}
	}
	var conflicts []Conflict

	allocs, err := a.listAllocations(vscaleID)
	if err != nil {
		return []Conflict{{CIDR: cidr, Conflicts: err.Error(), Severity: "error"}}
	}
	for _, al := range allocs {
		other, err := parseCIDR(al.CIDR)
		if err != nil {
			continue
		}
		if overlaps(n, other) {
			conflicts = append(conflicts, Conflict{
				CIDR:      cidr,
				Conflicts: fmt.Sprintf("overlaps allocated %s (%s)", al.CIDR, al.Kind),
				WithKind:  al.Kind,
				WithCIDR:  al.CIDR,
				Severity:  "error",
			})
		}
	}

	for _, lc := range localCIDRs {
		other, err := parseCIDR(lc)
		if err != nil {
			continue
		}
		if overlaps(n, other) {
			conflicts = append(conflicts, Conflict{
				CIDR:      cidr,
				Conflicts: fmt.Sprintf("overlaps local interface/route %s", lc),
				WithKind:  "local",
				WithCIDR:  lc,
				Severity:  "error",
			})
		}
	}

	for _, pw := range privateWarnings {
		other, _ := parseCIDR(pw)
		if overlaps(n, other) && !a.globalPool.Contains(n.IP) {
			conflicts = append(conflicts, Conflict{
				CIDR:      cidr,
				Conflicts: fmt.Sprintf("uses private range %s (intentional LAN route?)", pw),
				WithKind:  "rfc1918",
				WithCIDR:  pw,
				Severity:  "warning",
			})
		}
	}
	return conflicts
}

func hasErrorConflicts(cs []Conflict) bool {
	for _, c := range cs {
		if c.Severity == "error" {
			return true
		}
	}
	return false
}

// AllocateVscaleOverlay finds the next free /vscalePrefixLen inside the global pool.
func (a *Allocator) AllocateVscaleOverlay(vscaleID uint, preferred string) (*Result, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if preferred != "" {
		n, err := parseCIDR(preferred)
		if err != nil {
			return nil, fmt.Errorf("invalid preferred CIDR: %w", err)
		}
		ones, _ := n.Mask.Size()
		if ones != a.vscalePrefixLen {
			return nil, fmt.Errorf("preferred CIDR must be /%d", a.vscalePrefixLen)
		}
		if !a.globalPool.Contains(n.IP) {
			return nil, fmt.Errorf("preferred CIDR outside global pool %s", a.globalPool.String())
		}
		cs := a.CheckOverlap(preferred, nil, nil)
		if hasErrorConflicts(cs) {
			return &Result{CIDR: preferred, Conflicts: cs}, fmt.Errorf("preferred overlay conflicts")
		}
		vsID := vscaleID
		if err := a.persist(&vsID, models.AllocKindOverlay, preferred, "vscale", vscaleID, "vscale overlay"); err != nil {
			return nil, err
		}
		return &Result{CIDR: preferred, Conflicts: cs}, nil
	}

	candidates := a.iterateSubnets()
	for _, c := range candidates {
		cs := a.CheckOverlap(c, nil, nil)
		if hasErrorConflicts(cs) {
			continue
		}
		vsID := vscaleID
		if err := a.persist(&vsID, models.AllocKindOverlay, c, "vscale", vscaleID, "vscale overlay"); err != nil {
			return nil, err
		}
		return &Result{CIDR: c, Conflicts: cs}, nil
	}
	return nil, fmt.Errorf("no free overlay blocks in %s", a.globalPool.String())
}

func (a *Allocator) iterateSubnets() []string {
	ones, bits := a.globalPool.Mask.Size()
	blockSize := 1 << (bits - a.vscalePrefixLen)
	poolSize := 1 << (bits - ones)
	count := poolSize / blockSize
	base := ipToUint(a.globalPool.IP.To4())
	out := make([]string, 0, count)
	for i := 0; i < count; i++ {
		ip := uintToIP(base + uint32(i*blockSize))
		out = append(out, fmt.Sprintf("%s/%d", ip.String(), a.vscalePrefixLen))
	}
	return out
}

func ipToUint(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func uintToIP(n uint32) net.IP {
	return net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}

// AllocateHostIP allocates next free /32 inside the vscale overlay for hub or device.
func (a *Allocator) AllocateHostIP(vscaleID uint, overlayCIDR, kind string, refID uint) (*Result, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	overlay, err := parseCIDR(overlayCIDR)
	if err != nil {
		return nil, err
	}
	ones, bits := overlay.Mask.Size()
	total := 1 << (bits - ones)
	base := ipToUint(overlay.IP.To4())

	var used []models.IpAllocation
	if err := a.db.Where("vscale_id = ? AND kind IN ?", vscaleID, []string{models.AllocKindDevice, models.AllocKindHub}).Find(&used).Error; err != nil {
		return nil, err
	}
	usedSet := map[string]struct{}{}
	for _, u := range used {
		usedSet[u.CIDR] = struct{}{}
	}

	// skip network and broadcast for sanity when prefix < 31
	start, end := 1, total-1
	if total <= 2 {
		start, end = 0, total
	}
	for i := start; i < end; i++ {
		ip := uintToIP(base + uint32(i))
		cidr := ip.String() + "/32"
		if _, ok := usedSet[cidr]; ok {
			continue
		}
		vsID := vscaleID
		if err := a.persist(&vsID, kind, cidr, kind, refID, kind+" address"); err != nil {
			return nil, err
		}
		return &Result{CIDR: cidr}, nil
	}
	return nil, fmt.Errorf("no free host IPs in %s", overlayCIDR)
}

// ReserveRoute checks conflicts then persists a route allocation.
func (a *Allocator) ReserveRoute(vscaleID uint, cidr string, routeID uint, localCIDRs []string, force bool) (*Result, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	vsID := vscaleID
	cs := a.CheckOverlap(cidr, &vsID, localCIDRs)
	res := &Result{CIDR: cidr, Conflicts: cs}
	if hasErrorConflicts(cs) && !force {
		return res, fmt.Errorf("route conflicts with existing allocations")
	}
	if err := a.persist(&vsID, models.AllocKindRoute, cidr, "route", routeID, "advertised route"); err != nil {
		return nil, err
	}
	return res, nil
}

func (a *Allocator) ReleaseByRef(refType string, refID uint) error {
	return a.db.Where("ref_type = ? AND ref_id = ?", refType, refID).Delete(&models.IpAllocation{}).Error
}

func (a *Allocator) ReleaseCIDR(cidr string) error {
	return a.db.Where("cidr = ?", cidr).Delete(&models.IpAllocation{}).Error
}

func (a *Allocator) persist(vscaleID *uint, kind, cidr, refType string, refID uint, note string) error {
	al := models.IpAllocation{
		VscaleID: vscaleID,
		Kind:     kind,
		CIDR:     cidr,
		RefType:  refType,
		RefID:    refID,
		Note:     note,
	}
	return a.db.Create(&al).Error
}

// HostIP returns the IP without mask from a CIDR.
func HostIP(cidr string) string {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return cidr
	}
	return ip.String()
}
