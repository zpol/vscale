package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zpol/vscale/internal/auth"
	"github.com/zpol/vscale/internal/models"
	"github.com/zpol/vscale/internal/wgkeys"
)

type createHubReq struct {
	Name       string `json:"name" binding:"required"`
	Endpoint   string `json:"endpoint" binding:"required"`
	ListenPort int    `json:"listenPort"`
	MTU        int    `json:"mtu"`
}

func (s *Server) listHubs(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleAccess(c, vs.ID) {
		return
	}
	var hubs []models.Hub
	if err := s.DB.Where("vscale_id = ?", vs.ID).Find(&hubs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, hubs)
}

func (s *Server) createHub(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleManage(c, vs.ID) {
		return
	}
	var req createHubReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	port := req.ListenPort
	if port == 0 {
		port = 51820
	}
	mtu := req.MTU
	if mtu == 0 {
		mtu = 1420
	}
	kp, err := wgkeys.Generate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	tok, err := auth.RandomToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	hub := models.Hub{
		VscaleID:   vs.ID,
		Name:       req.Name,
		PublicKey:  kp.PublicKey,
		PrivateKey: kp.PrivateKey,
		ListenPort: port,
		Endpoint:   req.Endpoint,
		Address:    "pending",
		MTU:        mtu,
		Status:     models.StatusActive,
		AgentToken: tok,
	}
	if err := s.DB.Create(&hub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	res, err := s.Alloc.AllocateHostIP(vs.ID, vs.OverlayCIDR, models.AllocKindHub, hub.ID)
	if err != nil {
		s.DB.Delete(&hub)
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	hub.Address = res.CIDR
	s.DB.Save(&hub)

	c.JSON(http.StatusCreated, gin.H{
		"hub":        hub,
		"agentToken": tok,
		"hint":       fmt.Sprintf("Run on VPS: vscale hub --server %s --token %s", s.Cfg.PublicURL, tok),
	})
}

func (s *Server) deleteHub(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleManage(c, vs.ID) {
		return
	}
	hubID, ok := s.parseID(c, "hubId")
	if !ok {
		return
	}
	_ = s.Alloc.ReleaseByRef(models.AllocKindHub, hubID)
	s.DB.Where("vscale_id = ? AND id = ?", vs.ID, hubID).Delete(&models.Hub{})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) rotateHubToken(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleManage(c, vs.ID) {
		return
	}
	hubID, ok := s.parseID(c, "hubId")
	if !ok {
		return
	}
	var hub models.Hub
	if err := s.DB.Where("vscale_id = ? AND id = ?", vs.ID, hubID).First(&hub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "hub not found"})
		return
	}
	tok, err := auth.RandomToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	hub.AgentToken = tok
	s.DB.Save(&hub)
	c.JSON(http.StatusOK, gin.H{"hubId": hub.ID, "agentToken": tok})
}
