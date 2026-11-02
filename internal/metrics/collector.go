package metrics

import (
	"context"
	"log"
	"time"

	"github.com/zpol/vscale/internal/models"
	"gorm.io/gorm"
)

type Collector struct {
	DB       *gorm.DB
	Interval time.Duration
	Retain   time.Duration
}

func (c *Collector) Run(ctx context.Context) {
	interval := c.Interval
	if interval <= 0 {
		interval = time.Minute
	}
	retain := c.Retain
	if retain <= 0 {
		retain = 8 * 24 * time.Hour
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	c.sample()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.sample()
			cutoff := time.Now().Add(-retain)
			_ = c.DB.Where("taken_at < ?", cutoff).Delete(&models.MetricSnapshot{})
		}
	}
}

func (c *Collector) sample() {
	now := time.Now().UTC()
	var vscales, hubs, devices, routes, hubsOnline, devicesOnline int64
	c.DB.Model(&models.Vscale{}).Where("status = ?", models.StatusActive).Count(&vscales)
	c.DB.Model(&models.Hub{}).Where("status = ?", models.StatusActive).Count(&hubs)
	c.DB.Model(&models.Device{}).Where("status = ?", models.StatusActive).Count(&devices)
	c.DB.Model(&models.Route{}).Where("status = ?", models.StatusApproved).Count(&routes)
	cutoff := now.Add(-2 * time.Minute)
	c.DB.Model(&models.Hub{}).Where("status = ? AND last_heartbeat >= ?", models.StatusActive, cutoff).Count(&hubsOnline)
	c.DB.Model(&models.Device{}).Where("status = ? AND last_heartbeat >= ?", models.StatusActive, cutoff).Count(&devicesOnline)

	type sumRow struct{ Rx, Tx int64 }
	var dSum, hSum sumRow
	c.DB.Model(&models.Device{}).Select("COALESCE(SUM(transfer_rx),0) as rx, COALESCE(SUM(transfer_tx),0) as tx").Scan(&dSum)
	c.DB.Model(&models.Hub{}).Select("COALESCE(SUM(transfer_rx),0) as rx, COALESCE(SUM(transfer_tx),0) as tx").Scan(&hSum)
	rx := dSum.Rx + hSum.Rx
	tx := dSum.Tx + hSum.Tx

	var prev models.MetricSnapshot
	var rxRate, txRate int64
	if err := c.DB.Order("taken_at desc").First(&prev).Error; err == nil {
		dt := now.Sub(prev.TakenAt).Seconds()
		if dt > 0 {
			if rx >= prev.TransferRx {
				rxRate = int64(float64(rx-prev.TransferRx) / dt)
			}
			if tx >= prev.TransferTx {
				txRate = int64(float64(tx-prev.TransferTx) / dt)
			}
		}
	}

	snap := models.MetricSnapshot{
		TakenAt: now, Vscales: vscales, Hubs: hubs, HubsOnline: hubsOnline,
		Devices: devices, DevicesOnline: devicesOnline, Routes: routes,
		TransferRx: rx, TransferTx: tx, TransferRxRate: rxRate, TransferTxRate: txRate,
	}
	if err := c.DB.Create(&snap).Error; err != nil {
		log.Printf("metrics sample: %v", err)
	}
}
