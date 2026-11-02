package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zpol/vscale/internal/allocator"
	"github.com/zpol/vscale/internal/auth"
	"github.com/zpol/vscale/internal/models"
	"github.com/zpol/vscale/internal/wgkeys"
)

type enrollReq struct {
	Token    string `json:"token" binding:"required"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
}

type enrollResp struct {
	DeviceID   uint   `json:"deviceId"`
	VscaleID   uint   `json:"vscaleId"`
	AgentToken string `json:"agentToken"`
	Address    string `json:"address"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
	ServerURL  string `json:"serverUrl"`
}

func (s *Server) agentEnroll(c *gin.Context) {
	var req enrollReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var jt models.JoinToken
	if err := s.DB.Where("token = ?", req.Token).First(&jt).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid join token"})
		return
	}
	if jt.Revoked {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token revoked"})
		return
	}
	if jt.ExpiresAt != nil && time.Now().After(*jt.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
		return
	}
	if jt.UsesLeft <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token exhausted"})
		return
	}
	var vs models.Vscale
	if err := s.DB.First(&vs, jt.VscaleID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "vscale missing"})
		return
	}
	if vs.Status != models.StatusActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "vscale not active"})
		return
	}

	kp, err := wgkeys.Generate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	agentTok, err := auth.RandomToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	name := req.Name
	if name == "" {
		name = req.Hostname
	}
	if name == "" {
		name = "device-" + agentTok[:8]
	}
	d := models.Device{
		VscaleID:   vs.ID,
		Name:       name,
		Hostname:   req.Hostname,
		OS:         req.OS,
		PublicKey:  kp.PublicKey,
		PrivateKey: kp.PrivateKey,
		Address:    "pending",
		DNS:        vs.DNS,
		Status:     models.StatusActive,
		AgentToken: agentTok,
	}
	if err := s.DB.Create(&d).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	res, err := s.Alloc.AllocateHostIP(vs.ID, vs.OverlayCIDR, models.AllocKindDevice, d.ID)
	if err != nil {
		s.DB.Delete(&d)
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	d.Address = res.CIDR
	s.DB.Save(&d)

	jt.UsesLeft--
	s.DB.Save(&jt)

	c.JSON(http.StatusCreated, enrollResp{
		DeviceID:   d.ID,
		VscaleID:   vs.ID,
		AgentToken: agentTok,
		Address:    d.Address,
		PublicKey:  kp.PublicKey,
		PrivateKey: kp.PrivateKey,
		ServerURL:  s.Cfg.PublicURL,
	})
}

type peerConfig struct {
	PublicKey  string   `json:"publicKey"`
	Endpoint   string   `json:"endpoint,omitempty"`
	AllowedIPs []string `json:"allowedIps"`
	Keepalive  int      `json:"persistentKeepalive,omitempty"`
}

type agentConfigResp struct {
	Role       string       `json:"role"` // device | hub
	Interface  string       `json:"interface"`
	Address    string       `json:"address"`
	PrivateKey string       `json:"privateKey"`
	ListenPort int          `json:"listenPort,omitempty"`
	DNS        string       `json:"dns,omitempty"`
	MTU        int          `json:"mtu"`
	Peers      []peerConfig `json:"peers"`
	Routes     []string     `json:"routes"`
	VscaleID   uint         `json:"vscaleId"`
	OverlayCIDR string      `json:"overlayCidr"`
}

func (s *Server) agentConfig(c *gin.Context) {
	agentType, _ := c.Get("agentType")
	vscaleID, _ := c.Get("vscaleId")
	vsID := vscaleID.(uint)

	var vs models.Vscale
	if err := s.DB.First(&vs, vsID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "vscale not found"})
		return
	}

	var routes []models.Route
	s.DB.Where("vscale_id = ? AND status = ? AND advertised = ?", vsID, models.StatusApproved, true).Find(&routes)

	if agentType == "device" {
		deviceID, _ := c.Get("deviceId")
		var d models.Device
		if err := s.DB.First(&d, deviceID.(uint)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
			return
		}
		var hubs []models.Hub
		s.DB.Where("vscale_id = ? AND status = ?", vsID, models.StatusActive).Find(&hubs)

		allowed := []string{vs.OverlayCIDR}
		for _, r := range routes {
			allowed = append(allowed, r.CIDR)
		}

		peers := make([]peerConfig, 0, len(hubs))
		for _, h := range hubs {
			ep := h.Endpoint
			if ep != "" && !strings.Contains(ep, ":") {
				ep = ep + ":" + itoa(h.ListenPort)
			}
			peers = append(peers, peerConfig{
				PublicKey:  h.PublicKey,
				Endpoint:   ep,
				AllowedIPs: allowed,
				Keepalive:  25,
			})
		}
		// Do not push DNS into wg-quick for clients: it rewrites resolv.conf and
		// commonly breaks local/Tailscale DNS even with split AllowedIPs.
		c.JSON(http.StatusOK, agentConfigResp{
			Role: "device", Interface: "vscale0", Address: d.Address,
			PrivateKey: d.PrivateKey, DNS: "", MTU: 1420,
			Peers: peers, Routes: allowed, VscaleID: vsID, OverlayCIDR: vs.OverlayCIDR,
		})
		return
	}

	// hub
	hubID, _ := c.Get("hubId")
	var h models.Hub
	if err := s.DB.First(&h, hubID.(uint)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "hub not found"})
		return
	}
	var devices []models.Device
	s.DB.Where("vscale_id = ? AND status = ?", vsID, models.StatusActive).Find(&devices)

	peers := make([]peerConfig, 0, len(devices))
	for _, d := range devices {
		allowed := []string{d.Address}
		for _, r := range routes {
			if r.ViaType == models.RouteViaDevice && r.ViaID == d.ID {
				allowed = append(allowed, r.CIDR)
			}
		}
		peers = append(peers, peerConfig{
			PublicKey:  d.PublicKey,
			AllowedIPs: allowed,
		})
	}
	routeCIDRs := make([]string, 0)
	for _, r := range routes {
		if r.ViaType == models.RouteViaHub && r.ViaID == h.ID {
			routeCIDRs = append(routeCIDRs, r.CIDR)
		}
	}
	c.JSON(http.StatusOK, agentConfigResp{
		Role: "hub", Interface: "vscale0", Address: h.Address,
		PrivateKey: h.PrivateKey, ListenPort: h.ListenPort, MTU: h.MTU,
		Peers: peers, Routes: routeCIDRs, VscaleID: vsID, OverlayCIDR: vs.OverlayCIDR,
	})
}

type heartbeatReq struct {
	TransferRx    int64    `json:"transferRx"`
	TransferTx    int64    `json:"transferTx"`
	LocalCIDRs    []string `json:"localCidrs"`
	Hostname      string   `json:"hostname"`
	OS            string   `json:"os"`
	PeerStats     []peerStat `json:"peerStats"`
}

type peerStat struct {
	PublicKey     string `json:"publicKey"`
	TransferRx    int64  `json:"transferRx"`
	TransferTx    int64  `json:"transferTx"`
	LastHandshake int64  `json:"lastHandshake"` // unix
}

func (s *Server) agentHeartbeat(c *gin.Context) {
	var req heartbeatReq
	_ = c.ShouldBindJSON(&req)
	now := time.Now()
	agentType, _ := c.Get("agentType")

	warnings := []allocator.Conflict{}
	if len(req.LocalCIDRs) > 0 {
		vsID, _ := c.Get("vscaleId")
		id := vsID.(uint)
		var vs models.Vscale
		if s.DB.First(&vs, id).Error == nil {
			warnings = s.Alloc.CheckOverlap(vs.OverlayCIDR, &id, req.LocalCIDRs)
		}
	}

	if agentType == "device" {
		deviceID, _ := c.Get("deviceId")
		var d models.Device
		if err := s.DB.First(&d, deviceID.(uint)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		d.LastHeartbeat = &now
		d.TransferRx = req.TransferRx
		d.TransferTx = req.TransferTx
		if req.Hostname != "" {
			d.Hostname = req.Hostname
		}
		if req.OS != "" {
			d.OS = req.OS
		}
		if len(req.LocalCIDRs) > 0 {
			d.LocalCIDRs = strings.Join(req.LocalCIDRs, ",")
		}
		for _, ps := range req.PeerStats {
			if ps.LastHandshake > 0 {
				t := time.Unix(ps.LastHandshake, 0)
				d.LastHandshake = &t
			}
		}
		s.DB.Save(&d)
		c.JSON(http.StatusOK, gin.H{"ok": true, "warnings": warnings})
		return
	}

	hubID, _ := c.Get("hubId")
	var h models.Hub
	if err := s.DB.First(&h, hubID.(uint)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	h.LastHeartbeat = &now
	h.TransferRx = req.TransferRx
	h.TransferTx = req.TransferTx
	s.DB.Save(&h)

	// update device handshake from hub view
	for _, ps := range req.PeerStats {
		var d models.Device
		if s.DB.Where("public_key = ?", ps.PublicKey).First(&d).Error == nil {
			d.TransferRx = ps.TransferRx
			d.TransferTx = ps.TransferTx
			if ps.LastHandshake > 0 {
				t := time.Unix(ps.LastHandshake, 0)
				d.LastHandshake = &t
			}
			s.DB.Save(&d)
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "warnings": warnings})
}

// hubPeers is an alias-friendly endpoint for hubs (same as config peers).
func (s *Server) hubPeers(c *gin.Context) {
	s.agentConfig(c)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
