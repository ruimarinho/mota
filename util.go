package main

import (
	"errors"
	"fmt"
	"net"
)

// LocalIP get the host machine local IP address
func ServerIP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if isPrivateIP(ip) {
				return ip, nil
			}
		}
	}

	return nil, errors.New("no IP")
}

var privateIPBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		// don't check loopback ips
		//"127.0.0.0/8",    // IPv4 loopback
		//"::1/128",        // IPv6 loopback
		//"fe80::/10",      // IPv6 link-local
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

func isPrivateIP(ip net.IP) bool {
	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}

	return false
}

// AllLocalSubnets returns all host IPs across all private /24 subnets
// found on the machine's network interfaces.
func AllLocalSubnets() []net.IP {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var ips []net.IP

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP.To4()
			if ip == nil || !isPrivateIP(ip) {
				continue
			}

			// Use /24 prefix as dedup key to avoid scanning the same subnet twice.
			prefix := fmt.Sprintf("%d.%d.%d", ip[0], ip[1], ip[2])
			if seen[prefix] {
				continue
			}
			seen[prefix] = true

			for i := 1; i < 255; i++ {
				candidate := make(net.IP, 4)
				copy(candidate, ip)
				candidate[3] = byte(i)
				ips = append(ips, candidate)
			}
		}
	}

	return ips
}

// ExpandCIDR parses a CIDR string (e.g. "192.168.100.0/24") and returns
// all usable host IPs in that range (excluding network and broadcast).
func ExpandCIDR(cidr string) ([]net.IP, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	// Compute the broadcast address for this subnet.
	broadcast := make(net.IP, len(ipNet.IP))
	for i := range broadcast {
		broadcast[i] = ipNet.IP[i] | ^ipNet.Mask[i]
	}

	// Start from the network address.
	network := make(net.IP, len(ipNet.IP))
	copy(network, ipNet.IP)

	var ips []net.IP
	for candidate := network; ipNet.Contains(candidate); incrementIP(candidate) {
		if candidate.Equal(ipNet.IP) || candidate.Equal(broadcast) {
			continue
		}
		dup := make(net.IP, len(candidate))
		copy(dup, candidate)
		ips = append(ips, dup)
	}

	return ips, nil
}

func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// ServerPort attempts to retrieve a free open port.
func ServerPort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
