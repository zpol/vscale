package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zpol/vscale/internal/allocator"
	"github.com/zpol/vscale/internal/api"
	"github.com/zpol/vscale/internal/auth"
	"github.com/zpol/vscale/internal/config"
	"github.com/zpol/vscale/internal/metrics"
	"github.com/zpol/vscale/internal/store"
)

func main() {
	cfg := config.Load()
	db, err := store.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	if err := store.AutoMigrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := store.EnsureAdmin(db, cfg.DefaultAdminUser, cfg.DefaultAdminPass, cfg.DefaultAdminMail); err != nil {
		log.Fatalf("admin: %v", err)
	}
	alloc, err := allocator.New(db, cfg.GlobalPool, cfg.VscalePrefixLen)
	if err != nil {
		log.Fatalf("allocator: %v", err)
	}
	authSvc := auth.NewService(db, cfg.JWTSecret, cfg.JWTTTL)
	srv := api.NewServer(db, authSvc, alloc, cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go (&metrics.Collector{DB: db, Interval: time.Minute, Retain: 8 * 24 * time.Hour}).Run(ctx)

	log.Printf("vscale control-plane listening on %s", cfg.HTTPAddr)
	go func() {
		if err := srv.Run(); err != nil {
			log.Fatal(err)
		}
	}()
	<-ctx.Done()
	log.Println("shutting down")
}
