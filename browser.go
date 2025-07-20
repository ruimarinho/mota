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
	"time"

	zeroconf "github.com/grandcat/zeroconf"
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
}

// DiscoverDevices performs discovery of local devices using the zeroconf (or
// bonjour) protocol. The lookup is executed against a domain and Shellies
// are discovered via their web browser service announcement.
func (b *Browser) DiscoverDevices(hosts []string) ([]Device, error) {
	devices := make([]Device, 0)
	entriesChan := make(chan *zeroconf.ServiceEntry)
	devicesChan := make(chan Device)
	fetchedDevicesChan := make(chan Device)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(b.waitTime))
	defer cancel()

	// Filter devices found to shellies only.
	go b.filterShellies(entriesChan, devicesChan)

	// Fetch settings as soon as devices are found.
	go b.fetchSettings(devicesChan, fetchedDevicesChan)

	if len(hosts) == 0 {
		log.Infof("Discovering devices on the network for %v seconds...", b.waitTime)

		resolver, err := zeroconf.NewResolver(nil)
		if err != nil {
			return devices, err
		}

		err = resolver.Browse(ctx, b.service, b.domain, entriesChan)
		if err != nil {
			return devices, err
		}
	} else {
		log.Infof("Preparing to update devices with hosts %v", hosts)

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

			entriesChan <- &zeroconf.ServiceEntry{
				HostName: host,
				Port:     port,
				AddrIPv4: resolvedIPs,
				Text:     []string{fmt.Sprintf("id=shelly-%s", host)},
			}
		}

		close(entriesChan)
	}

	for device := range fetchedDevicesChan {
		devices = append(devices, device)
	}

	log.Debug("All device settings fetched!")

	return devices, nil
}

// fetchSettings retrieves the model name and current firmware version
// via the Settings API from each Shelly discovered. If authentication
// is required, .netrc authentication is used, if available.
func (b *Browser) fetchSettings(foundDevicesChan chan Device, fetchedDevicesChan chan Device) {
	var done sync.WaitGroup
	var netrcFile *netrc.Netrc
	netrcPath, err := netrcPath()
	if err == nil {
		netrcFile, err = netrc.Parse(netrcPath)
	}
	for device := range foundDevicesChan {
		done.Add(1)
		go func(device Device, fetchedDevicesChan chan Device) {
			log.Infof("Fetching settings from %v", device.String())
			defer done.Done()

			// try to load general credentials from the user config if available
			path, err := UserConfigPath()
			if err != nil {
				log.Debug(err)
			} else {
				userConfig, err := LoadUserConfig(path)
				if err != nil {
					log.Debug(err)
				}
				if userConfig != nil {
					device.Username = userConfig.GlobalConfig.DefaultCredentials.Username
					device.Password = userConfig.GlobalConfig.DefaultCredentials.Password
				}
			}

			// if there is a netrc fle that defines specific credentials, override the globa credentials
			if netrcFile != nil && netrcFile.Machine(device.IP.String()) != nil {
				log.Debugf("Found netrc entry for device %v", device.String())

				device.Username = netrcFile.Machine(device.IP.String()).Get("login")
				device.Password = url.QueryEscape(netrcFile.Machine(device.IP.String()).Get("password"))
			}

			client := http.Client{
				Timeout: 5 * time.Second,
			}

			response, err := client.Get(device.GetBaseURL() + "/settings")
			if err != nil {
				log.Debug(err)
				return
			}

			defer response.Body.Close()

			if response.StatusCode != 200 {
				log.Errorf("Unable to fetch settings from %v due to incorrect or missing username/password", device.String())
				return
			}

			var settings Settings
			err = json.NewDecoder(response.Body).Decode(&settings)
			if err != nil {
				fmt.Println("Error parsing JSON: ", err)
				return
			}

			// Update the device's model type (e.g. SHSW-25) and current firmware.
			device.Model = settings.Device.Type
			device.CurrentFWVersion = settings.FW

			log.Debugf("Parsed settings from device %v", device.String())

			fetchedDevicesChan <- device
		}(device, fetchedDevicesChan)
	}

	done.Wait()
	close(fetchedDevicesChan)
}

// filterShellies rejects any non-Shelly devices from the discovered
// devices. Shellies announce their identifier (which always starts
// with shelly*) on the service metadata.
func (b *Browser) filterShellies(entriesChan <-chan *zeroconf.ServiceEntry, devicesChan chan Device) {
	for entry := range entriesChan {
		for _, str := range entry.Text {
			if strings.HasPrefix(str, "id=shelly") {
				IP := entry.AddrIPv4[0]

				log.Infof("Found device %v (%v)", entry.HostName, IP.String())

				devicesChan <- Device{IP: IP, HostName: entry.HostName, Port: entry.Port}
				break
			}
		}
	}

	log.Debug("No more discovered devices left to filter")

	close(devicesChan)
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
