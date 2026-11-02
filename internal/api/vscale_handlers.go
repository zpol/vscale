package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zpol/vscale/internal/auth"
	"github.com/zpol/vscale/internal/models"
	"github.com/zpol/vscale/internal/topology"
)

type createVscaleReq struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	OverlayCIDR  string `json:"overlayCidr"` // optional preferred
	DNS          string `json:"dns"`
}

func (s *Server) listVscales(c *gin.Context) {
	var list []models.Vscale
	if auth.IsGlobalAdmin(c) || auth.Role(c) == models.RoleOperator {
		if err := s.DB.Preload("Hubs").Preload("Devices").Find(&list).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		var memberships []models.Membership
		if err := s.DB.Where("user_id = ?", auth.UserID(c)).Find(&memberships).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ids := make([]uint, 0, len(memberships))
		for _, m := range memberships {
			ids = append(ids, m.VscaleID)
		}
		if len(ids) == 0 {
			c.JSON(http.StatusOK, []models.Vscale{})
			return
		}
		if err := s.DB.Preload("Hubs").Preload("Devices").Where("id IN ?", ids).Find(&list).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, list)
}

func (s *Server) createVscale(c *gin.Context) {
	var req createVscaleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// DNS is optional metadata only; clients do not get DNS= in WireGuard conf.
	dns := req.DNS
	vs := models.Vscale{
		Name:        req.Name,
		Description: req.Description,
		DNS:         dns,
		Status:      models.StatusActive,
		OverlayCIDR: "pending",
	}
	if err := s.DB.Create(&vs).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	res, err := s.Alloc.AllocateVscaleOverlay(vs.ID, req.OverlayCIDR)
	if err != nil {
		s.DB.Delete(&vs)
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "conflicts": res})
		return
	}
	vs.OverlayCIDR = res.CIDR
	if err := s.DB.Save(&vs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// creator becomes owner
	m := models.Membership{
		VscaleID: vs.ID,
		UserID:   auth.UserID(c),
		Role:     models.MembershipOwner,
	}
	_ = s.DB.Create(&m)
	c.JSON(http.StatusCreated, gin.H{"vscale": vs, "allocation": res})
}

func (s *Server) getVscale(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleAccess(c, vs.ID) {
		return
	}
	s.DB.Preload("Hubs").Preload("Devices").Preload("Routes").Preload("Memberships").First(vs, vs.ID)
	c.JSON(http.StatusOK, vs)
}

type updateVscaleReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	DNS         string `json:"dns"`
	Status      string `json:"status"`
}

func (s *Server) updateVscale(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleManage(c, vs.ID) {
		return
	}
	var req updateVscaleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name != "" {
		vs.Name = req.Name
	}
	if req.Description != "" {
		vs.Description = req.Description
	}
	if req.DNS != "" {
		vs.DNS = req.DNS
	}
	if req.Status != "" {
		vs.Status = req.Status
	}
	if err := s.DB.Save(vs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, vs)
}

func (s *Server) deleteVscale(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	tx := s.DB.Begin()
	tx.Where("vscale_id = ?", vs.ID).Delete(&models.Device{})
	tx.Where("vscale_id = ?", vs.ID).Delete(&models.Hub{})
	tx.Where("vscale_id = ?", vs.ID).Delete(&models.Route{})
	tx.Where("vscale_id = ?", vs.ID).Delete(&models.JoinToken{})
	tx.Where("vscale_id = ?", vs.ID).Delete(&models.Membership{})
	tx.Where("vscale_id = ?", vs.ID).Delete(&models.IpAllocation{})
	tx.Delete(vs)
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) getTopology(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleAccess(c, vs.ID) {
		return
	}
	g, err := topology.Build(s.DB, vs.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, g)
}

func (s *Server) listMembers(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleAccess(c, vs.ID) {
		return
	}
	var members []models.Membership
	if err := s.DB.Preload("User").Where("vscale_id = ?", vs.ID).Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, members)
}

type memberReq struct {
	UserID uint   `json:"userId" binding:"required"`
	Role   string `json:"role"`
}

func (s *Server) addMember(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleManage(c, vs.ID) {
		return
	}
	var req memberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := req.Role
	if role == "" {
		role = models.MembershipMember
	}
	m := models.Membership{VscaleID: vs.ID, UserID: req.UserID, Role: role}
	if err := s.DB.Create(&m).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, m)
}

func (s *Server) removeMember(c *gin.Context) {
	vs, ok := s.loadVscale(c)
	if !ok {
		return
	}
	if !s.requireVscaleManage(c, vs.ID) {
		return
	}
	uid, ok := s.parseID(c, "userId")
	if !ok {
		return
	}
	s.DB.Where("vscale_id = ? AND user_id = ?", vs.ID, uid).Delete(&models.Membership{})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type checkCIDRReq struct {
	CIDR       string   `json:"cidr" binding:"required"`
	VscaleID   *uint    `json:"vscaleId"`
	LocalCIDRs []string `json:"localCidrs"`
}

func (s *Server) checkCIDR(c *gin.Context) {
	var req checkCIDRReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cs := s.Alloc.CheckOverlap(req.CIDR, req.VscaleID, req.LocalCIDRs)
	c.JSON(http.StatusOK, gin.H{"cidr": req.CIDR, "conflicts": cs})
}
