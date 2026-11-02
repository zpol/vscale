package agent

import (
	"net"
)

// LocalCIDRs returns IPv4 CIDRs of non-loopback interfaces for conflict detection.
func LocalCIDRs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ipnet *net.IPNet
			switch v := a.(type) {
			case *net.IPNet:
				ipnet = v
			case *net.IPAddr:
				ipnet = &net.IPNet{IP: v.IP, Mask: v.IP.DefaultMask()}
			}
			if ipnet == nil || ipnet.IP.To4() == nil {
				continue
			}
			// skip link-local
			if ipnet.IP.IsLinkLocalUnicast() {
				continue
			}
			out = append(out, ipnet.String())
		}
	}
	return out
}
