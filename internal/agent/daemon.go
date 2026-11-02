package agent

import (
	"context"
	"log"
	"os"
	"runtime"
	"time"
)

type Daemon struct {
	StatePath string
	Interval  time.Duration
}

func (d *Daemon) Run(ctx context.Context) error {
	st, err := LoadState(d.StatePath)
	if err != nil {
		return err
	}
	if st.AgentToken == "" || st.ServerURL == "" {
		return ErrNotEnrolled
	}
	client := NewAPIClient(st.ServerURL, st.AgentToken)
	interval := d.Interval
	if interval == 0 {
		interval = 30 * time.Second
	}

	apply := func() error {
		cfg, err := client.Config()
		if err != nil {
			return err
		}
		var path string
		if st.Role == "hub" {
			path, err = SyncConf(cfg)
		} else {
			path, err = ApplyUp(cfg)
		}
		if err != nil {
			log.Printf("apply warning: %v", err)
		} else {
			log.Printf("tunnel applied (%s)", path)
		}
		st.Address = cfg.Address
		st.PrivateKey = cfg.PrivateKey
		st.Up = true
		_ = SaveState(d.StatePath, st)

		hb := HeartbeatRequest{
			LocalCIDRs: LocalCIDRs(),
			Hostname:   hostname(),
			OS:         runtime.GOOS,
		}
		if err := client.Heartbeat(hb); err != nil {
			log.Printf("heartbeat: %v", err)
		}
		return nil
	}

	if err := apply(); err != nil {
		log.Printf("initial apply: %v", err)
	}

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if err := apply(); err != nil {
				log.Printf("sync: %v", err)
			}
		}
	}
}

var ErrNotEnrolled = errNotEnrolled{}

type errNotEnrolled struct{}

func (errNotEnrolled) Error() string { return "not enrolled; run: vscale join <token> --server <url>" }

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}
