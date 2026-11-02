package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zpol/vscale/internal/auth"
	"github.com/zpol/vscale/internal/models"
)

func (s *Server) listDevices(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleAccess(c, vs.ID) {
		return
	}
	var devices []models.Device
	if err := s.DB.Where("vscale_id = ?", vs.ID).Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type deviceView struct {
		models.Device
		Online bool `json:"online"`
	}
	out := make([]deviceView, 0, len(devices))
	for _, d := range devices {
		online := d.LastHeartbeat != nil && time.Since(*d.LastHeartbeat) < 2*time.Minute && d.Status == models.StatusActive
		out = append(out, deviceView{Device: d, Online: online})
	}
	c.JSON(http.StatusOK, out)
}

type updateDeviceReq struct {
	Name   string `json:"name"`
	Tags   string `json:"tags"`
	Status string `json:"status"`
}

func (s *Server) updateDevice(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleManage(c, vs.ID) {
		return
	}
	did, ok := s.parseID(c, "deviceId")
	if !ok {
		return
	}
	var d models.Device
	if err := s.DB.Where("vscale_id = ? AND id = ?", vs.ID, did).First(&d).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	var req updateDeviceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name != "" {
		d.Name = req.Name
	}
	if req.Tags != "" {
		d.Tags = req.Tags
	}
	if req.Status != "" {
		d.Status = req.Status
	}
	s.DB.Save(&d)
	c.JSON(http.StatusOK, d)
}

func (s *Server) revokeDevice(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleManage(c, vs.ID) {
		return
	}
	did, ok := s.parseID(c, "deviceId")
	if !ok {
		return
	}
	var d models.Device
	if err := s.DB.Where("vscale_id = ? AND id = ?", vs.ID, did).First(&d).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	d.Status = models.StatusRevoked
	d.AgentToken = ""
	s.DB.Save(&d)
	_ = s.Alloc.ReleaseByRef(models.AllocKindDevice, d.ID)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type joinTokenReq struct {
	Label     string `json:"label"`
	Reusable  bool   `json:"reusable"`
	ExpiresIn int    `json:"expiresInHours"`
	Uses      int    `json:"uses"`
}

func (s *Server) listJoinTokens(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleManage(c, vs.ID) {
		return
	}
	var tokens []models.JoinToken
	s.DB.Where("vscale_id = ?", vs.ID).Find(&tokens)
	c.JSON(http.StatusOK, tokens)
}

func (s *Server) createJoinToken(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleManage(c, vs.ID) {
		return
	}
	var req joinTokenReq
	_ = c.ShouldBindJSON(&req)
	tok, err := auth.RandomToken(24)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uses := req.Uses
	if uses == 0 {
		if req.Reusable {
			uses = 1000
		} else {
			uses = 1
		}
	}
	jt := models.JoinToken{
		VscaleID:  vs.ID,
		CreatedBy: auth.UserID(c),
		Token:     tok,
		Label:     req.Label,
		Reusable:  req.Reusable,
		UsesLeft:  uses,
	}
	if req.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresIn) * time.Hour)
		jt.ExpiresAt = &t
	}
	if err := s.DB.Create(&jt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, jt)
}

func (s *Server) revokeJoinToken(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleManage(c, vs.ID) {
		return
	}
	tid, ok := s.parseID(c, "tokenId")
	if !ok {
		return
	}
	s.DB.Model(&models.JoinToken{}).Where("vscale_id = ? AND id = ?", vs.ID, tid).Update("revoked", true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type createRouteReq struct {
	CIDR        string   `json:"cidr" binding:"required"`
	ViaType     string   `json:"viaType" binding:"required"`
	ViaID       uint     `json:"viaId" binding:"required"`
	Description string   `json:"description"`
	LocalCIDRs  []string `json:"localCidrs"`
	Force       bool     `json:"force"`
	AutoApprove bool     `json:"autoApprove"`
}

func (s *Server) listRoutes(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleAccess(c, vs.ID) {
		return
	}
	var routes []models.Route
	s.DB.Where("vscale_id = ?", vs.ID).Find(&routes)
	c.JSON(http.StatusOK, routes)
}

func (s *Server) createRoute(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleManage(c, vs.ID) {
		return
	}
	var req createRouteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ViaType != models.RouteViaHub && req.ViaType != models.RouteViaDevice {
		c.JSON(http.StatusBadRequest, gin.H{"error": "viaType must be hub or device"})
		return
	}
	status := models.StatusPending
	if req.AutoApprove {
		status = models.StatusApproved
	}
	r := models.Route{
		VscaleID:    vs.ID,
		CIDR:        req.CIDR,
		ViaType:     req.ViaType,
		ViaID:       req.ViaID,
		Status:      status,
		Description: req.Description,
		Advertised:  true,
	}
	if err := s.DB.Create(&r).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if status == models.StatusApproved {
		res, err := s.Alloc.ReserveRoute(vs.ID, req.CIDR, r.ID, req.LocalCIDRs, req.Force)
		if err != nil {
			s.DB.Delete(&r)
			c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "conflicts": res})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"route": r, "allocation": res})
		return
	}
	// pending: still check and report
	cs := s.Alloc.CheckOverlap(req.CIDR, &vs.ID, req.LocalCIDRs)
	c.JSON(http.StatusCreated, gin.H{"route": r, "conflicts": cs})
}

func (s *Server) approveRoute(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleManage(c, vs.ID) {
		return
	}
	rid, ok := s.parseID(c, "routeId")
	if !ok {
		return
	}
	var r models.Route
	if err := s.DB.Where("vscale_id = ? AND id = ?", vs.ID, rid).First(&r).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
		return
	}
	var body struct {
		Force      bool     `json:"force"`
		LocalCIDRs []string `json:"localCidrs"`
	}
	_ = c.ShouldBindJSON(&body)
	res, err := s.Alloc.ReserveRoute(vs.ID, r.CIDR, r.ID, body.LocalCIDRs, body.Force)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "conflicts": res})
		return
	}
	r.Status = models.StatusApproved
	s.DB.Save(&r)
	c.JSON(http.StatusOK, gin.H{"route": r, "allocation": res})
}

func (s *Server) denyRoute(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleManage(c, vs.ID) {
		return
	}
	rid, ok := s.parseID(c, "routeId")
	if !ok {
		return
	}
	s.DB.Model(&models.Route{}).Where("vscale_id = ? AND id = ?", vs.ID, rid).Update("status", models.StatusDenied)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) deleteRoute(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleManage(c, vs.ID) {
		return
	}
	rid, ok := s.parseID(c, "routeId")
	if !ok {
		return
	}
	_ = s.Alloc.ReleaseByRef("route", rid)
	s.DB.Where("vscale_id = ? AND id = ?", vs.ID, rid).Delete(&models.Route{})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
