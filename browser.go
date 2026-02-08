package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	zeroconf "github.com/libp2p/zeroconf/v2"
	"github.com/jdxcode/netrc"
	log "github.com/sirupsen/logrus"
)

// Browser holds information about the discovery request, including the
// domain where the search is performed, the service type (usually
// the Shelly's integrated web server) and wait time.
type Browser struct {
	domain   string
	service  string
	waitTime int
	subnets  []string
	username string
	password string
}

type gen2PlusShellyInfoResponse struct {
	Generation int    `json:"gen"`
	Model      string `json:"app"`
}

// ListenForAnnouncements performs discovery of local devices using mDNS/zeroconf.
// The lookup is executed against a domain and Shellies
// are discovered via their web browser service announcement.
func (b *Browser) ListenForAnnouncements(hosts []string) ([]Device, error) {
	devices := make([]Device, 0)
	entriesChan := make(chan *zeroconf.ServiceEntry)
	devicesChan := make(chan DeviceAnnouncement)
	fetchedDevicesChan := make(chan Device)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(b.waitTime))
	defer cancel()

	var deviceCount int32

	// Filter devices found to shellies only.
	go b.findShellies(entriesChan, devicesChan, &deviceCount)

	// Fetch settings as soon as devices are found.
	go b.fetchSettings(devicesChan, fetchedDevicesChan)

	if len(hosts) == 0 {
		log.Infof("Discovering devices on the network for %v seconds...", b.waitTime)

		err := zeroconf.Browse(ctx, b.service, b.domain, entriesChan)
		if err != nil {
			log.Debug(err)
			return devices, err
		}

		// Log periodic discovery progress until the context expires.
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
			case <-ticker.C:
				n := atomic.LoadInt32(&deviceCount)
				log.Infof("Discovery in progress... %d device(s) found so far", n)
				continue
			}
			break
		}
	} else {
		log.Infof("Looking for specific devices %v", hosts)

		for _, host := range hosts {
			if !strings.Contains(host, ":") {
				host = fmt.Sprintf("%s:80", host)
			}

			hostString, portString, err := net.SplitHostPort(host)
			if err != nil {
				log.Errorf("Host %v is invalid (%v), skipping", host, err)
				continue
			}

			port, err := strconv.Atoi(portString)
			if err != nil {
				log.Errorf("Port for host %v is invalid (%v), skipping", host, err)
				continue
			}

			var resolvedIPs []net.IP
			parsedIP := net.ParseIP(hostString)
			if parsedIP != nil {
				resolvedIPs = append(resolvedIPs, parsedIP)
			} else {
				log.Debugf("Host %v does not look like an IP, attempting to resolve as host...", host)

				resolvedIPs, err = net.LookupIP(host)
				if err != nil {
					log.Errorf("Host %v is invalid (%v), skipping...", host, err)
					continue
				}
			}

			client := http.Client{
				Timeout: 10 * time.Second,
			}

			response, err := client.Get(fmt.Sprintf("http://%v/shelly", host))
			if err != nil {
				log.Debug(err)
				continue
			}

			var decoded gen2PlusShellyInfoResponse
			err = json.NewDecoder(response.Body).Decode(&decoded)
			response.Body.Close()
			if err != nil {
				log.Debug(err)
				continue
			}

			var header string
			switch decoded.Generation {
			case 4:
				header = Gen4AnnouncementHeader
			case 3:
				header = Gen3AnnouncementHeader
			case 2:
				header = Gen2AnnouncementHeader
			default:
				header = fmt.Sprintf("%v-%v", Gen1AnnouncementHeader, host)
			}

			if header == "" {
				log.Errorf("Unknown generation! Please open an issue at https://github.com/ruimarinho/mota/issues/new")
			}

			entriesChan <- &zeroconf.ServiceEntry{
				HostName: host,
				Port:     port,
				AddrIPv4: resolvedIPs,
				Text:     []string{header},
			}
		}

		close(entriesChan)
	}

	for device := range fetchedDevicesChan {
		devices = append(devices, device)
	}

	log.Debug("All device settings fetched!")

	// Phase 2: If auto-discovery was used, supplement with an HTTP subnet
	// scan to find devices (especially Gen2+) that mDNS may miss on macOS.
	if len(hosts) == 0 {
		seen := make(map[string]bool)
		for _, d := range devices {
			seen[d.IP.String()] = true
		}

		extra, err := b.scanSubnetForDevices(seen)
		if err != nil {
			log.Debugf("Subnet scan failed: %v", err)
		} else if len(extra) > 0 {
			log.Infof("Subnet scan found %d additional device(s)", len(extra))
			devices = append(devices, extra...)
		}
	}

	return devices, nil
}

// findShellies rejects any non-Shelly devices from the discovered
// devices. Shellies announce their identifier (which always starts
// with "shelly*" for gen1 or "gen=2" for gen2) on the service metadata.
func (b *Browser) findShellies(entriesChan <-chan *zeroconf.ServiceEntry, devicesChan chan DeviceAnnouncement, deviceCount *int32) {
	seen := make(map[string]bool)

	for entry := range entriesChan {
		if len(entry.AddrIPv4) == 0 {
			continue
		}

		ip := entry.AddrIPv4[0].String()
		if seen[ip] {
			log.Debugf("Skipping duplicate device at %v", ip)
			continue
		}

		for _, value := range entry.Text {
			if strings.HasPrefix(value, Gen1AnnouncementHeader) || value == Gen2AnnouncementHeader || value == Gen3AnnouncementHeader || value == Gen4AnnouncementHeader {
				IP := entry.AddrIPv4[0]
				generation := 1
				if value == Gen2AnnouncementHeader {
					generation = 2
				} else if value == Gen3AnnouncementHeader {
					generation = 3
				} else if value == Gen4AnnouncementHeader {
					generation = 4
				}

				seen[ip] = true
				devicesChan <- DeviceAnnouncement{
					IP:         IP,
					HostName:   entry.HostName,
					Port:       entry.Port,
					Generation: generation,
				}

				n := atomic.AddInt32(deviceCount, 1)
				log.Infof("Found device %v (%v) [%d found]", entry.HostName, IP.String(), n)
				break
			}
		}
	}

	log.Debug("No more discovered devices left to find")

	close(devicesChan)
}

// scanSubnetForDevices probes all IPs in the local /24 subnet via HTTP
// to find Shelly devices missed by mDNS. Returns fully fetched Device
// entries, skipping any IPs in the seen map.
func (b *Browser) scanSubnetForDevices(seen map[string]bool) ([]Device, error) {
	ips, localCIDRs := AllLocalSubnets()

	for _, cidr := range localCIDRs {
		log.Infof("Detected local subnet %s", cidr)
	}

	// Append IPs from explicitly configured subnets (--subnet flag).
	for _, cidr := range b.subnets {
		extra, err := ExpandCIDR(cidr)
		if err != nil {
			log.Warnf("Invalid subnet %q: %v", cidr, err)
			continue
		}
		log.Infof("Adding %d IPs from subnet %s", len(extra), cidr)
		ips = append(ips, extra...)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no subnets to scan")
	}

	log.Infof("Scanning %d IPs for additional devices...", len(ips))

	const maxWorkers = 50
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var devices []Device

	shellyClient := http.Client{Timeout: 2 * time.Second}
	settingsClient := http.Client{Timeout: 5 * time.Second}

	var netrcFile *netrc.Netrc
	netrcPath, nerr := netrcPath()
	if nerr == nil {
		netrcFile, _ = netrc.Parse(netrcPath)
	}

	for _, ip := range ips {
		if seen[ip.String()] {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(ip net.IP) {
			defer wg.Done()
			defer func() { <-sem }()

			host := ip.String()
			resp, err := shellyClient.Get(fmt.Sprintf("http://%v/shelly", host))
			if err != nil {
				return
			}

			var info gen2PlusShellyInfoResponse
			err = json.NewDecoder(resp.Body).Decode(&info)
			resp.Body.Close()
			if err != nil {
				return
			}

			generation := info.Generation
			if generation == 0 {
				generation = 1
			}

			da := DeviceAnnouncement{
				IP:         ip,
				HostName:   fmt.Sprintf("%v:80", host),
				Port:       80,
				Generation: generation,
			}

			var username, password string
			if netrcFile != nil && netrcFile.Machine(host) != nil {
				username = netrcFile.Machine(host).Get("login")
				password = url.QueryEscape(netrcFile.Machine(host).Get("password"))
			} else if b.username != "" || b.password != "" {
				username = b.username
				password = url.QueryEscape(b.password)
			}

			settingsResp, err := settingsClient.Get(da.DeviceInformationURL(username, password))
			if err != nil {
				log.Debugf("Subnet scan: failed to fetch settings from %v: %v", host, err)
				return
			}
			defer settingsResp.Body.Close()

			if settingsResp.StatusCode != 200 {
				log.Debugf("Subnet scan: auth failed for %v", host)
				return
			}

			var device Device
			if generation == 1 {
				var settings Gen1Settings
				if err := json.NewDecoder(settingsResp.Body).Decode(&settings); err != nil {
					return
				}
				device = Device{
					ID: settings.Device.Hostname, Name: settings.Name,
					Model: settings.Device.Model, FirmwareVersion: settings.Firmware,
					Generation: generation, IP: ip, Port: 80,
					Username: username, Password: password,
				}
			} else {
				var settings Gen2Settings
				if err := json.NewDecoder(settingsResp.Body).Decode(&settings); err != nil {
					return
				}
				device = Device{
					ID: settings.ID, Name: settings.Name,
					Model: settings.Model, FirmwareVersion: settings.Firmware,
					Generation: generation, IP: ip, Port: 80,
					Username: username, Password: password,
				}
			}

			log.Infof("Found device %v (%v) via subnet scan", device.String(), host)

			mu.Lock()
			devices = append(devices, device)
			mu.Unlock()
		}(ip)
	}

	wg.Wait()
	log.Debugf("Subnet scan complete, found %d additional device(s)", len(devices))
	return devices, nil
}

// fetchSettings retrieves the model name and current firmware version
// via the Settings API from each Shelly discovered. If authentication
// is required, .netrc authentication is used, if available.
func (b *Browser) fetchSettings(foundDevicesChan chan DeviceAnnouncement, fetchedDevicesChan chan Device) {
	var done sync.WaitGroup
	var netrcFile *netrc.Netrc

	netrcPath, err := netrcPath()
	if err == nil {
		netrcFile, err = netrc.Parse(netrcPath)
	}

	if err != nil {
		log.Errorf("Netrc appears to be malformed")
	}

	for deviceAnnouncement := range foundDevicesChan {
		done.Add(1)
		go func(deviceAnnouncement DeviceAnnouncement, fetchedDevicesChan chan Device) {
			defer done.Done()

			var username string
			var password string

			log.Infof("Fetching settings from %v", deviceAnnouncement.String())

			if netrcFile != nil && netrcFile.Machine(deviceAnnouncement.IP.String()) != nil {
				log.Debugf("Found netrc entry for device %v", deviceAnnouncement.String())

				username = netrcFile.Machine(deviceAnnouncement.IP.String()).Get("login")
				password = url.QueryEscape(netrcFile.Machine(deviceAnnouncement.IP.String()).Get("password"))
			} else if b.username != "" || b.password != "" {
				log.Debugf("Using global credentials for device %v", deviceAnnouncement.String())

				username = b.username
				password = url.QueryEscape(b.password)
			}

			client := http.Client{
				Timeout: 5 * time.Second,
			}

			response, err := client.Get(deviceAnnouncement.DeviceInformationURL(username, password))
			if err != nil {
				log.Warnf("Failed to fetch settings from %v: %v", deviceAnnouncement.String(), err)
				return
			}

			defer response.Body.Close()

			if response.StatusCode != 200 {
				log.Errorf("Unable to fetch settings from %v due to incorrect or missing username/password", deviceAnnouncement.String())
				return
			}

			var device Device
			if deviceAnnouncement.Generation == 1 {
				var settings Gen1Settings
				err = json.NewDecoder(response.Body).Decode(&settings)
				if err != nil {
					log.Errorf("Error parsing JSON: %v", err)
					return
				}

				// Update the device's model type (e.g. SHSW-25) and current firmware.
				device = Device{
					ID:              settings.Device.Hostname,
					Name:            settings.Name,
					Model:           settings.Device.Model,
					FirmwareVersion: settings.Firmware,
					Generation:      deviceAnnouncement.Generation,
					IP:              deviceAnnouncement.IP,
					Port:            deviceAnnouncement.Port,
					Username:        username,
					Password:        password,
				}

			} else {
				var settings Gen2Settings
				err = json.NewDecoder(response.Body).Decode(&settings)
				if err != nil {
					log.Errorf("Error parsing JSON: %v", err)
					return
				}

				// Update the device's model type (e.g. Plus2PM) and current firmware.
				device = Device{
					ID:              settings.ID,
					Name:            settings.Name,
					Model:           settings.Model,
					FirmwareVersion: settings.Firmware,
					Generation:      deviceAnnouncement.Generation,
					IP:              deviceAnnouncement.IP,
					Port:            deviceAnnouncement.Port,
					Username:        username,
					Password:        password,
				}
			}

			log.Debugf("Parsed settings from device %v", device.String())

			fetchedDevicesChan <- device
		}(deviceAnnouncement, fetchedDevicesChan)
	}

	done.Wait()
	close(fetchedDevicesChan)
}

// netrcPath attempts to find the .netrc file path depending
// on the OS. Code extracted from
// https://golang.org/src/cmd/go/internal/auth/netrc.go.
func netrcPath() (string, error) {
	if env := os.Getenv("NETRC"); env != "" {
		return env, nil
	}
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	base := ".netrc"
	if runtime.GOOS == "windows" {
		base = "_netrc"
	}
	return filepath.Join(dir, base), nil
}
