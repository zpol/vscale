package topology

import (
	"time"

	"github.com/zpol/vscale/internal/models"
	"gorm.io/gorm"
)

type Node struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"` // hub | device
	Label    string   `json:"label"`
	Hostname string   `json:"hostname,omitempty"`
	Address  string   `json:"address"`
	Status   string   `json:"status"`
	OS       string   `json:"os,omitempty"`
	Online   bool     `json:"online"`
	CIDRs    []string `json:"cidrs,omitempty"`
	LastSeen *time.Time `json:"lastSeen,omitempty"`
}

type Edge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"` // tunnel | route
	Label  string `json:"label,omitempty"`
}

type Graph struct {
	VscaleID    uint     `json:"vscaleId"`
	OverlayCIDR string   `json:"overlayCidr"`
	Nodes       []Node   `json:"nodes"`
	Edges       []Edge   `json:"edges"`
}

func onlineSince(t *time.Time, window time.Duration) bool {
	if t == nil {
		return false
	}
	return time.Since(*t) < window
}

func Build(db *gorm.DB, vscaleID uint) (*Graph, error) {
	var vs models.Vscale
	if err := db.First(&vs, vscaleID).Error; err != nil {
		return nil, err
	}
	var hubs []models.Hub
	var devices []models.Device
	var routes []models.Route
	if err := db.Where("vscale_id = ?", vscaleID).Find(&hubs).Error; err != nil {
		return nil, err
	}
	if err := db.Where("vscale_id = ?", vscaleID).Find(&devices).Error; err != nil {
		return nil, err
	}
	if err := db.Where("vscale_id = ? AND status = ?", vscaleID, models.StatusApproved).Find(&routes).Error; err != nil {
		return nil, err
	}

	window := 2 * time.Minute
	g := &Graph{VscaleID: vscaleID, OverlayCIDR: vs.OverlayCIDR}

	hubIDs := map[uint]string{}
	for _, h := range hubs {
		id := "hub-" + itoa(h.ID)
		hubIDs[h.ID] = id
		cidrs := []string{h.Address}
		for _, r := range routes {
			if r.ViaType == models.RouteViaHub && r.ViaID == h.ID {
				cidrs = append(cidrs, r.CIDR)
			}
		}
		online := onlineSince(h.LastHeartbeat, window)
		g.Nodes = append(g.Nodes, Node{
			ID: id, Type: "hub", Label: h.Name, Hostname: h.Name, Address: h.Address,
			Status: h.Status, Online: online, CIDRs: cidrs, LastSeen: h.LastHeartbeat,
		})
	}

	for _, d := range devices {
		id := "device-" + itoa(d.ID)
		cidrs := []string{d.Address}
		for _, r := range routes {
			if r.ViaType == models.RouteViaDevice && r.ViaID == d.ID {
				cidrs = append(cidrs, r.CIDR)
			}
		}
		online := onlineSince(d.LastHeartbeat, window) && d.Status == models.StatusActive
		label := d.Hostname
		if label == "" {
			label = d.Name
		}
		g.Nodes = append(g.Nodes, Node{
			ID: id, Type: "device", Label: label, Hostname: d.Hostname, Address: d.Address,
			Status: d.Status, OS: d.OS, Online: online, CIDRs: cidrs, LastSeen: d.LastHeartbeat,
		})
		// spoke edges to every hub (hub-and-spoke)
		for _, h := range hubs {
			g.Edges = append(g.Edges, Edge{
				ID: "tunnel-" + id + "-" + hubIDs[h.ID],
				Source: id, Target: hubIDs[h.ID], Kind: "tunnel", Label: "wg",
			})
		}
	}

	for _, r := range routes {
		via := ""
		if r.ViaType == models.RouteViaHub {
			via = hubIDs[r.ViaID]
		} else {
			via = "device-" + itoa(r.ViaID)
		}
		if via == "" {
			continue
		}
		g.Edges = append(g.Edges, Edge{
			ID: "route-" + itoa(r.ID), Source: via, Target: via,
			Kind: "route", Label: r.CIDR,
		})
	}
	return g, nil
}

func itoa(u uint) string {
	return fmtUint(u)
}

func fmtUint(u uint) string {
	if u == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for u > 0 {
		i--
		b[i] = byte('0' + u%10)
		u /= 10
	}
	return string(b[i:])
}
