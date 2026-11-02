package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"bufio"
	"strings"

	"github.com/zpol/vscale/internal/agent"
)

// Set via -ldflags "-X main.version=…"
var version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	statePath := agent.DefaultStatePath()
	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "join":
		join(statePath, args)
	case "setup":
		setup(statePath, args)
	case "hub":
		hubRegister(statePath, args)
	case "up":
		up(statePath)
	case "down":
		down(statePath)
	case "status":
		status(statePath)
	case "daemon":
		runDaemon(statePath)
	case "routes":
		routes(statePath)
	case "version":
		fmt.Println("vscale", version, runtime.GOOS+"/"+runtime.GOARCH)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `vscale — hub-spoke VPN agent

Usage:
  vscale setup [--server <url>] [--token <token>] [--name <name>]
  vscale join <token> --server <url> [--name <name>]
  vscale hub --server <url> --token <hub-agent-token>
  vscale up
  vscale down
  vscale status
  vscale daemon
  vscale routes
  vscale version
`)
}

func flagVal(args []string, name string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == name && i+1 < len(args) {
			return args[i+1]
		}
		if len(args[i]) > len(name)+1 && args[i][:len(name)+1] == name+"=" {
			return args[i][len(name)+1:]
		}
	}
	return ""
}

func setup(statePath string, args []string) {
	server := flagVal(args, "--server")
	token := flagVal(args, "--token")
	name := flagVal(args, "--name")
	if server == "" {
		server = os.Getenv("VSCALE_SERVER")
	}
	if token == "" {
		token = os.Getenv("VSCALE_JOIN_TOKEN")
	}

	in := bufio.NewReader(os.Stdin)
	prompt := func(label, def string) string {
		if def != "" {
			fmt.Printf("%s [%s]: ", label, def)
		} else {
			fmt.Printf("%s: ", label)
		}
		line, _ := in.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return def
		}
		return line
	}

	fmt.Println("vscale setup — connect this PC to your VPN")
	if server == "" {
		server = prompt("Server URL (e.g. https://vscale.example.com)", "https://vscale.0ffsec.com")
	}
	if token == "" {
		token = prompt("Join token (from your admin)", "")
	}
	if token == "" {
		fatal("join token is required")
	}
	if name == "" {
		host, _ := os.Hostname()
		name = prompt("Device name", host)
	}

	join(statePath, []string{token, "--server", server, "--name", name})
	fmt.Print("Bring the tunnel up now? [Y/n]: ")
	ans, _ := in.ReadString('\n')
	ans = strings.TrimSpace(strings.ToLower(ans))
	if ans == "" || ans == "y" || ans == "yes" {
		up(statePath)
		fmt.Println("Done. Check with: vscale status")
	} else {
		fmt.Println("Later run: vscale up")
	}
}

func join(statePath string, args []string) {
	if len(args) < 1 {
		fatal("join requires a token")
	}
	token := args[0]
	server := flagVal(args, "--server")
	if server == "" {
		server = os.Getenv("VSCALE_SERVER")
	}
	if server == "" {
		fatal("--server or VSCALE_SERVER required")
	}
	name := flagVal(args, "--name")
	client := agent.NewAPIClient(server, "")
	host, _ := os.Hostname()
	resp, err := client.Enroll(agent.EnrollRequest{
		Token: token, Name: name, Hostname: host, OS: runtime.GOOS,
	})
	if err != nil {
		fatal(err.Error())
	}
	st := &agent.State{
		ServerURL:  server,
		AgentToken: resp.AgentToken,
		DeviceID:   resp.DeviceID,
		VscaleID:   resp.VscaleID,
		Address:    resp.Address,
		PrivateKey: resp.PrivateKey,
		PublicKey:  resp.PublicKey,
		Role:       "device",
	}
	if err := agent.SaveState(statePath, st); err != nil {
		fatal(err.Error())
	}
	fmt.Printf("enrolled device %d address %s\n", resp.DeviceID, resp.Address)
	fmt.Println("run: vscale up")
}

func hubRegister(statePath string, args []string) {
	server := flagVal(args, "--server")
	tok := flagVal(args, "--token")
	if server == "" {
		server = os.Getenv("VSCALE_SERVER")
	}
	if server == "" || tok == "" {
		fatal("hub requires --server and --token")
	}
	st := &agent.State{
		ServerURL:  server,
		AgentToken: tok,
		Role:       "hub",
	}
	if err := agent.SaveState(statePath, st); err != nil {
		fatal(err.Error())
	}
	fmt.Println("hub credentials saved; run: vscale daemon")
	runDaemon(statePath)
}

func up(statePath string) {
	st, err := agent.LoadState(statePath)
	if err != nil {
		fatal(err.Error())
	}
	client := agent.NewAPIClient(st.ServerURL, st.AgentToken)
	cfg, err := client.Config()
	if err != nil {
		fatal(err.Error())
	}
	path, err := agent.ApplyUp(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	} else {
		fmt.Printf("tunnel up (%s)\n", path)
	}
	st.Up = true
	st.Address = cfg.Address
	_ = agent.SaveState(statePath, st)
	_ = client.Heartbeat(agent.HeartbeatRequest{
		LocalCIDRs: agent.LocalCIDRs(),
		Hostname:   hostname(),
		OS:         runtime.GOOS,
	})
}

func down(statePath string) {
	if err := agent.ApplyDown(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	} else {
		fmt.Println("tunnel down")
	}
	if st, err := agent.LoadState(statePath); err == nil {
		st.Up = false
		_ = agent.SaveState(statePath, st)
	}
}

func status(statePath string) {
	st, err := agent.LoadState(statePath)
	if err != nil {
		fatal("not enrolled: " + err.Error())
	}
	view := map[string]any{
		"role":     st.Role,
		"server":   st.ServerURL,
		"vscaleId": st.VscaleID,
		"deviceId": st.DeviceID,
		"address":  st.Address,
		"up":       st.Up,
		"os":       runtime.GOOS,
		"localCidrs": agent.LocalCIDRs(),
	}
	if st.AgentToken != "" {
		client := agent.NewAPIClient(st.ServerURL, st.AgentToken)
		if cfg, err := client.Config(); err == nil {
			view["peers"] = len(cfg.Peers)
			view["overlay"] = cfg.OverlayCIDR
			view["routes"] = cfg.Routes
			view["dns"] = cfg.DNS
		} else {
			view["configError"] = err.Error()
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(view)
}

func routes(statePath string) {
	st, err := agent.LoadState(statePath)
	if err != nil {
		fatal(err.Error())
	}
	client := agent.NewAPIClient(st.ServerURL, st.AgentToken)
	cfg, err := client.Config()
	if err != nil {
		fatal(err.Error())
	}
	for _, r := range cfg.Routes {
		fmt.Println(r)
	}
}

func runDaemon(statePath string) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	d := &agent.Daemon{StatePath: statePath, Interval: 30 * time.Second}
	fmt.Println("vscale daemon running")
	if err := d.Run(ctx); err != nil {
		fatal(err.Error())
	}
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "error:", msg)
	os.Exit(1)
}
