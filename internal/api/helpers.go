package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zpol/vscale/internal/auth"
	"github.com/zpol/vscale/internal/models"
	"gorm.io/gorm"
)

func (s *Server) parseID(c *gin.Context, name string) (uint, bool) {
	id64, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + name})
		return 0, false
	}
	return uint(id64), true
}

func (s *Server) canAccessVscale(c *gin.Context, vscaleID uint) bool {
	if auth.IsGlobalAdmin(c) || auth.Role(c) == models.RoleOperator {
		return true
	}
	var m models.Membership
	err := s.DB.Where("vscale_id = ? AND user_id = ?", vscaleID, auth.UserID(c)).First(&m).Error
	return err == nil
}

func (s *Server) canManageVscale(c *gin.Context, vscaleID uint) bool {
	if auth.IsGlobalAdmin(c) || auth.Role(c) == models.RoleOperator {
		return true
	}
	var m models.Membership
	err := s.DB.Where("vscale_id = ? AND user_id = ? AND role IN ?", vscaleID, auth.UserID(c),
		[]string{models.MembershipOwner, models.MembershipAdmin}).First(&m).Error
	return err == nil
}

func (s *Server) requireVscaleAccess(c *gin.Context, vscaleID uint) bool {
	if !s.canAccessVscale(c, vscaleID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "no access to this vscale"})
		return false
	}
	return true
}

func (s *Server) requireVscaleManage(c *gin.Context, vscaleID uint) bool {
	if !s.canManageVscale(c, vscaleID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot manage this vscale"})
		return false
	}
	return true
}

func (s *Server) loadVscale(c *gin.Context) (*models.Vscale, bool) {
	id, ok := s.parseID(c, "id")
	if !ok {
		return nil, false
	}
	var vs models.Vscale
	if err := s.DB.First(&vs, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "vscale not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return nil, false
	}
	return &vs, true
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
