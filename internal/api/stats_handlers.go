package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zpol/vscale/internal/models"
)

func (s *Server) stats(c *gin.Context) {
	var vscaleCount, hubCount, deviceCount, routeCount int64
	s.DB.Model(&models.Vscale{}).Where("status = ?", models.StatusActive).Count(&vscaleCount)
	s.DB.Model(&models.Hub{}).Where("status = ?", models.StatusActive).Count(&hubCount)
	s.DB.Model(&models.Device{}).Where("status = ?", models.StatusActive).Count(&deviceCount)
	s.DB.Model(&models.Route{}).Where("status = ?", models.StatusApproved).Count(&routeCount)

	cutoff := time.Now().Add(-2 * time.Minute)
	var onlineDevices int64
	s.DB.Model(&models.Device{}).
		Where("status = ? AND last_heartbeat >= ?", models.StatusActive, cutoff).
		Count(&onlineDevices)
	var onlineHubs int64
	s.DB.Model(&models.Hub{}).
		Where("status = ? AND last_heartbeat >= ?", models.StatusActive, cutoff).
		Count(&onlineHubs)

	var rx, tx int64
	type sumRow struct{ Rx, Tx int64 }
	var dSum, hSum sumRow
	s.DB.Model(&models.Device{}).Select("COALESCE(SUM(transfer_rx),0) as rx, COALESCE(SUM(transfer_tx),0) as tx").Scan(&dSum)
	s.DB.Model(&models.Hub{}).Select("COALESCE(SUM(transfer_rx),0) as rx, COALESCE(SUM(transfer_tx),0) as tx").Scan(&hSum)
	rx = dSum.Rx + hSum.Rx
	tx = dSum.Tx + hSum.Tx

	c.JSON(http.StatusOK, gin.H{
		"vscales":       vscaleCount,
		"hubs":          hubCount,
		"hubsOnline":    onlineHubs,
		"devices":       deviceCount,
		"devicesOnline": onlineDevices,
		"routes":        routeCount,
		"transferRx":    rx,
		"transferTx":    tx,
	})
}

func (s *Server) statsHistory(c *gin.Context) {
	rangeKey := c.DefaultQuery("range", "24h")
	var since time.Duration
	switch rangeKey {
	case "1h":
		since = time.Hour
	case "7h":
		since = 7 * time.Hour
	case "12h":
		since = 12 * time.Hour
	case "24h":
		since = 24 * time.Hour
	case "7d":
		since = 7 * 24 * time.Hour
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "range must be 1h|7h|12h|24h|7d"})
		return
	}
	from := time.Now().UTC().Add(-since)
	var snaps []models.MetricSnapshot
	if err := s.DB.Where("taken_at >= ?", from).Order("taken_at asc").Find(&snaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"range":  rangeKey,
		"from":   from,
		"points": snaps,
	})
}
