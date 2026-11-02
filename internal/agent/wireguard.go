package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func ConfPath() string {
	if p := os.Getenv("VSCALE_WG_CONF"); p != "" {
		return p
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(os.TempDir(), "vscale0.conf")
	}
	return "/etc/wireguard/vscale0.conf"
}

func userConfPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "vscale", "vscale0.conf")
}

func renderWGConfig(cfg *ConfigResponse) string {
	var b strings.Builder
	b.WriteString("[Interface]\n")
	b.WriteString(fmt.Sprintf("PrivateKey = %s\n", cfg.PrivateKey))
	b.WriteString(fmt.Sprintf("Address = %s\n", cfg.Address))
	if cfg.ListenPort > 0 {
		b.WriteString(fmt.Sprintf("ListenPort = %d\n", cfg.ListenPort))
	}
	if cfg.DNS != "" {
		b.WriteString(fmt.Sprintf("DNS = %s\n", cfg.DNS))
	}
	if cfg.MTU > 0 {
		b.WriteString(fmt.Sprintf("MTU = %d\n", cfg.MTU))
	}
	if cfg.Role == "hub" && runtime.GOOS == "linux" {
		// Do not fail interface bring-up if sysctl is read-only (containers).
		b.WriteString("PostUp = sysctl -w net.ipv4.ip_forward=1 || true; iptables -C FORWARD -i %i -j ACCEPT 2>/dev/null || iptables -A FORWARD -i %i -j ACCEPT; iptables -C FORWARD -o %i -j ACCEPT 2>/dev/null || iptables -A FORWARD -o %i -j ACCEPT\n")
		b.WriteString("PostDown = iptables -D FORWARD -i %i -j ACCEPT 2>/dev/null || true; iptables -D FORWARD -o %i -j ACCEPT 2>/dev/null || true\n")
	}
	for _, p := range cfg.Peers {
		b.WriteString("\n[Peer]\n")
		b.WriteString(fmt.Sprintf("PublicKey = %s\n", p.PublicKey))
		if p.Endpoint != "" {
			b.WriteString(fmt.Sprintf("Endpoint = %s\n", p.Endpoint))
		}
		if len(p.AllowedIPs) > 0 {
			b.WriteString(fmt.Sprintf("AllowedIPs = %s\n", strings.Join(p.AllowedIPs, ", ")))
		}
		if p.Keepalive > 0 {
			b.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", p.Keepalive))
		}
	}
	return b.String()
}

func WriteWGConfig(cfg *ConfigResponse) (string, error) {
	body := []byte(renderWGConfig(cfg))
	candidates := []string{ConfPath(), userConfPath()}
	var lastErr error
	for _, path := range candidates {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && !os.IsExist(err) {
			lastErr = err
			continue
		}
		if err := os.WriteFile(path, body, 0o600); err != nil {
			lastErr = err
			continue
		}
		return path, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no writable WireGuard config path")
	}
	return "", lastErr
}

// ApplyUp brings the tunnel up using wg-quick when available.
func ApplyUp(cfg *ConfigResponse) (string, error) {
	path, err := WriteWGConfig(cfg)
	if err != nil {
		return "", err
	}
	if _, err := exec.LookPath("wg-quick"); err == nil {
		iface := strings.TrimSuffix(filepath.Base(path), ".conf")
		// down first to allow resync
		_ = exec.Command("wg-quick", "down", path).Run()
		cmd := exec.Command("wg-quick", "up", path)
		out, err := cmd.CombinedOutput()
		if err != nil {
			// try by interface name if path-style fails
			_ = exec.Command("wg-quick", "down", iface).Run()
			cmd2 := exec.Command("wg-quick", "up", iface)
			out2, err2 := cmd2.CombinedOutput()
			if err2 != nil {
				return path, fmt.Errorf("wg-quick up failed: %v (%s); fallback: %v (%s). Config written to %s", err, string(out), err2, string(out2), path)
			}
			return path, nil
		}
		_ = out
		return path, nil
	}
	return path, fmt.Errorf("wg-quick not found; WireGuard config written to %s — import it with the WireGuard app", path)
}

func ApplyDown() error {
	path := ConfPath()
	if _, err := os.Stat(path); err != nil {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".config", "vscale", "vscale0.conf")
	}
	if _, err := exec.LookPath("wg-quick"); err == nil {
		if err := exec.Command("wg-quick", "down", path).Run(); err == nil {
			return nil
		}
		iface := strings.TrimSuffix(filepath.Base(path), ".conf")
		return exec.Command("wg-quick", "down", iface).Run()
	}
	return fmt.Errorf("wg-quick not found; disable the tunnel in your WireGuard client")
}

// SyncConf rewrites config and tries wg syncconf for live peer updates (hub).
// wg syncconf updates peers but does not install AllowedIPs routes — we add those
// explicitly so overlay traffic is not blackholed via the default gateway.
func SyncConf(cfg *ConfigResponse) (string, error) {
	path, err := WriteWGConfig(cfg)
	if err != nil {
		return "", err
	}
	iface := "vscale0"
	if _, err := exec.LookPath("wg"); err == nil {
		stripped := path + ".strip"
		// wg syncconf needs stripped config without Address/DNS/MTU Address lines sometimes —
		// generate a peers-only conf via wg-quick strip if available
		if _, err := exec.LookPath("wg-quick"); err == nil {
			out, err := exec.Command("wg-quick", "strip", path).CombinedOutput()
			if err == nil {
				_ = os.WriteFile(stripped, out, 0o600)
				if err := exec.Command("wg", "syncconf", iface, stripped).Run(); err == nil {
					ensurePeerRoutes(iface, cfg)
					return path, nil
				}
			}
		}
	}
	// fallback: full recreate
	return ApplyUp(cfg)
}

func ensurePeerRoutes(iface string, cfg *ConfigResponse) {
	if runtime.GOOS == "windows" {
		return
	}
	if _, err := exec.LookPath("ip"); err != nil {
		return
	}
	for _, p := range cfg.Peers {
		for _, cidr := range p.AllowedIPs {
			cidr = strings.TrimSpace(cidr)
			if cidr == "" {
				continue
			}
			_ = exec.Command("ip", "route", "replace", cidr, "dev", iface).Run()
		}
	}
}
