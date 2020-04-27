package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
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
func (b *Browser) DiscoverDevices() ([]Device, error) {
	log.Infof("Discovering devices on the network for %v seconds...", b.waitTime)

	devices := make([]Device, 0)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return devices, err
	}

	devicesChan := make(chan Device)
	entriesChan := make(chan *zeroconf.ServiceEntry)
	go b.filterShellies(entriesChan, devicesChan)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(b.waitTime))
	defer cancel()

	err = resolver.Browse(ctx, b.service, b.domain, entriesChan)
	if err != nil {
		return devices, err
	}

	fetchedDevicesChan := make(chan Device)

	// Fetch settings as soon as devices are found.
	go b.fetchSettings(ctx, devicesChan, fetchedDevicesChan)

	for device := range fetchedDevicesChan {
		devices = append(devices, device)
	}

	log.Debug("All device settings fetched!")

	return devices, nil
}

// fetchSettings retrieves the model name and current firmware version
// via the Settings API from each Shelly discovered. If authentication
// is required, .netrc authentication is used, if available.
func (b *Browser) fetchSettings(ctx context.Context, foundDevicesChan chan Device, fetchedDevicesChan chan Device) {
	var done sync.WaitGroup
	searching := true

	var netrcFile *netrc.Netrc
	netrcPath, err := netrcPath()
	if err == nil {
		netrcFile, err = netrc.Parse(netrcPath)
	}

	for searching {
		select {
		case device := <-foundDevicesChan:
			done.Add(1)
			go func(device Device, fetchedDevicesChan chan Device) {
				log.Debugf("Fetching settings from %v", device.String())
				defer done.Done()

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

		case <-ctx.Done():
			// Stop waiting for more devices if the discovery time has passed.
			close(foundDevicesChan)
			searching = false
		}
	}

	done.Wait()
	close(fetchedDevicesChan)
}

// filterShellies rejects any non-Shelly devices from the discovered
// devices. Shellies announce their identifier (which always starts
// with shelly*) on the service metadata.
func (b *Browser) filterShellies(results <-chan *zeroconf.ServiceEntry, devicesChan chan Device) {
	for entry := range results {
		for _, str := range entry.Text {
			if strings.HasPrefix(str, "id=shelly") {
				IP := entry.AddrIPv4[0]

				log.Infof("Found device %v (%v)", entry.HostName, IP.String())

				devicesChan <- Device{IP: IP, HostName: entry.HostName, Port: entry.Port}
				break
			}
		}
	}

	log.Debug("No more discovered devices left to process")
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
