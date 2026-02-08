package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	log.SetOutput(io.Discard)
}

func TestNonUpgradable(t *testing.T) {
	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(mockSingleDeviceStableVersion("SHSW-25", "http://"+req.Host)))
			return
		}
		if strings.HasPrefix(req.URL.Path, "/update/") {
			w.Write([]byte(mockEmptyGen2FirmwareVersion()))
			return
		}
		assert.Fail(t, req.URL.Path)
	}))
	defer shellyCloudAPIServer.Close()

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/shelly":
			w.Write([]byte(`{"gen":1}`))
		case "/settings":
			w.Write([]byte(mockDeviceSettingsJSON("SHSW-25", "1CAAB5059F90", "20200309-104051/v1.6.0@43056d58")))
		default:
			assert.Fail(t, "unexpected device path: "+req.URL.Path)
		}
	}))
	defer deviceServer.Close()

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)

	otaUpdater, err := NewOTAService(
		WithAPIClient(
			NewAPIClient(
				WithBaseURL(shellyCloudAPIServer.URL),
				WithGen2BaseURL(shellyCloudAPIServer.URL),
			),
		),
		WithWaitTimeInSeconds(2),
		WithDevices([]string{fmt.Sprintf("127.0.0.1:%v", deviceServerPort)}),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	assert.Nil(t, err)

	devices, err := otaUpdater.DiscoverDevices()
	assert.Nil(t, err)
	assert.Len(t, devices, 1)

	for _, device := range devices {
		assert.Equal(t, deviceServerPort, device.Port)
		assert.Equal(t, deviceServerURL.Hostname(), device.IP.String())
		assert.Equal(t, "20200309-104051/v1.6.0@43056d58", device.FirmwareVersion)
		assert.Equal(t, RemoteFirmware{}, device.FirmwareNewestVersion)
	}
}

func TestUpgradable(t *testing.T) {
	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(mockSingleDeviceStableVersion("SHSW-25", "http://"+req.Host)))
			return
		}
		if strings.HasPrefix(req.URL.Path, "/update/") {
			w.Write([]byte(mockEmptyGen2FirmwareVersion()))
			return
		}
		if req.URL.Path == "/firmware/SHSW-25_build.zip" {
			w.Write([]byte(`{OK}`))
			return
		}
		assert.Fail(t, req.URL.Path)
	}))
	defer shellyCloudAPIServer.Close()

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/shelly":
			w.Write([]byte(`{"gen":1}`))
		case "/settings":
			w.Write([]byte(mockDeviceSettingsJSON("SHSW-25", "1CAAB5059F90", "20191127-095418/v1.5.6@0d769d69")))
		default:
			assert.Fail(t, "unexpected device path: "+req.URL.Path)
		}
	}))
	defer deviceServer.Close()

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)

	otaUpdater, err := NewOTAService(
		WithAPIClient(
			NewAPIClient(
				WithBaseURL(shellyCloudAPIServer.URL),
				WithGen2BaseURL(shellyCloudAPIServer.URL),
			),
		),
		WithWaitTimeInSeconds(2),
		WithDevices([]string{fmt.Sprintf("127.0.0.1:%v", deviceServerPort)}),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	assert.Nil(t, err)

	devices, err := otaUpdater.DiscoverDevices()
	assert.Nil(t, err)
	assert.Len(t, devices, 1)

	for _, device := range devices {
		assert.Equal(t, deviceServerPort, device.Port)
		assert.Equal(t, deviceServerURL.Hostname(), device.IP.String())
		assert.Equal(t, "20191127-095418/v1.5.6@0d769d69", device.FirmwareVersion)
		assert.Equal(t, "20200309-104051/v1.6.0@43056d58", device.FirmwareNewestVersion.Version)
	}
}

func TestUpgradableBeta(t *testing.T) {
	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(fmt.Sprintf(`{
				"isok": true,
				"data": {
					"SHSW-25": {
						"url": "%v/firmware/SHSW-25_build.zip",
						"version": "20200309-104051/v1.6.0@43056d58",
						"beta_url": "%v/firmware/SHSW-25_build_beta.zip",
						"beta_ver": "20210122-154345/v1.10.0-rc1@00eeaa9b"
					}
				}
			}`, "http://"+req.Host, "http://"+req.Host)))
			return
		}

		if strings.HasPrefix(req.URL.Path, "/update/") {
			w.Write([]byte(mockEmptyGen2FirmwareVersion()))
			return
		}

		if req.URL.Path == "/firmware/SHSW-25_build_beta.zip" {
			w.Write([]byte(`OK`))
			return
		}
		assert.Fail(t, req.URL.Path)
	}))
	defer shellyCloudAPIServer.Close()

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/shelly":
			w.Write([]byte(`{"gen":1}`))
		case "/settings":
			w.Write([]byte(mockDeviceSettingsJSON("SHSW-25", "1CAAB5059F90", "20191127-095418/v1.5.6@0d769d69")))
		default:
			assert.Fail(t, "unexpected device path: "+req.URL.Path)
		}
	}))
	defer deviceServer.Close()

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)

	otaUpdater, err := NewOTAService(
		WithAPIClient(
			NewAPIClient(
				WithBaseURL(shellyCloudAPIServer.URL),
				WithGen2BaseURL(shellyCloudAPIServer.URL),
			),
		),
		WithBetaVersions(true),
		WithWaitTimeInSeconds(2),
		WithDevices([]string{fmt.Sprintf("127.0.0.1:%v", deviceServerPort)}),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	assert.Nil(t, err)

	devices, err := otaUpdater.DiscoverDevices()
	assert.Nil(t, err)
	assert.Len(t, devices, 1)

	for _, device := range devices {
		assert.Equal(t, deviceServerPort, device.Port)
		assert.Equal(t, deviceServerURL.Hostname(), device.IP.String())
		assert.Equal(t, "20191127-095418/v1.5.6@0d769d69", device.FirmwareVersion)
		assert.Equal(t, "20200309-104051/v1.6.0@43056d58", device.FirmwareNewestVersion.Version)
		assert.Equal(t, "20210122-154345/v1.10.0-rc1@00eeaa9b", device.FirmwareNewestVersion.BetaVersion)
	}
}

func TestMalformedHosts(t *testing.T) {
	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(mockSingleDeviceStableVersion("SHSW-25", "http://"+req.Host)))
			return
		}
		if strings.HasPrefix(req.URL.Path, "/update/") {
			w.Write([]byte(mockEmptyGen2FirmwareVersion()))
			return
		}
		assert.Fail(t, req.URL.Path)
	}))
	defer shellyCloudAPIServer.Close()

	otaUpdater, err := NewOTAService(
		WithAPIClient(
			NewAPIClient(
				WithBaseURL(shellyCloudAPIServer.URL),
				WithGen2BaseURL(shellyCloudAPIServer.URL),
			),
		),
		WithWaitTimeInSeconds(2),
		WithDevices([]string{"*"}),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	assert.Nil(t, err)

	devices, err := otaUpdater.DiscoverDevices()
	assert.Nil(t, err)
	assert.Len(t, devices, 0)
}

func TestMalformedHostPort(t *testing.T) {
	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(mockSingleDeviceStableVersion("SHSW-25", "http://"+req.Host)))
			return
		}
		if strings.HasPrefix(req.URL.Path, "/update/") {
			w.Write([]byte(mockEmptyGen2FirmwareVersion()))
			return
		}
		assert.Fail(t, req.URL.Path)
	}))
	defer shellyCloudAPIServer.Close()

	otaUpdater, err := NewOTAService(
		WithAPIClient(
			NewAPIClient(
				WithBaseURL(shellyCloudAPIServer.URL),
				WithGen2BaseURL(shellyCloudAPIServer.URL),
			),
		),
		WithWaitTimeInSeconds(2),
		WithDevices([]string{"192.168.1.100::80"}),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	assert.Nil(t, err)

	devices, err := otaUpdater.DiscoverDevices()
	assert.Nil(t, err)
	assert.Len(t, devices, 0)
}

func TestGen2Upgradable(t *testing.T) {
	model := "Plus1"

	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(`{"isok": true, "data": {}}`))
			return
		}
		if req.URL.Path == "/update/"+model {
			w.Write([]byte(mockGen2FirmwareVersion(model, "http://"+req.Host)))
			return
		}
		w.Write([]byte(`{"stable":{"version":"0.0.0","build_id":"","url":""},"beta":{"version":"","build_id":"","url":""}}`))
	}))
	defer shellyCloudAPIServer.Close()

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/shelly":
			w.Write([]byte(fmt.Sprintf(`{"gen":2,"app":"%s"}`, model)))
		case "/rpc/Shelly.GetDeviceInfo":
			w.Write([]byte(mockGen2DeviceInfoJSON(model, "shellyplus1-AABBCC", "1.3.3")))
		default:
			assert.Fail(t, "unexpected device path: "+req.URL.Path)
		}
	}))
	defer deviceServer.Close()

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)

	otaUpdater, err := NewOTAService(
		WithAPIClient(
			NewAPIClient(
				WithBaseURL(shellyCloudAPIServer.URL),
				WithGen2BaseURL(shellyCloudAPIServer.URL),
			),
		),
		WithWaitTimeInSeconds(2),
		WithDevices([]string{fmt.Sprintf("127.0.0.1:%v", deviceServerPort)}),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	assert.Nil(t, err)

	devices, err := otaUpdater.DiscoverDevices()
	assert.Nil(t, err)
	assert.Len(t, devices, 1)

	for _, device := range devices {
		assert.Equal(t, model, device.Model)
		assert.Equal(t, "1.3.3", device.FirmwareVersion)
		assert.Equal(t, "1.5.0", device.FirmwareNewestVersion.Version)
		assert.Equal(t, 2, device.Generation)
	}
}

func TestGen3Upgradable(t *testing.T) {
	model := "1G3"

	apiModel := "S1G3" // 1G3 maps to S1G3 in the update API

	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(`{"isok": true, "data": {}}`))
			return
		}
		if req.URL.Path == "/update/"+apiModel {
			w.Write([]byte(mockGen2FirmwareVersion(model, "http://"+req.Host)))
			return
		}
		w.Write([]byte(`{"stable":{"version":"0.0.0","build_id":"","url":""},"beta":{"version":"","build_id":"","url":""}}`))
	}))
	defer shellyCloudAPIServer.Close()

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/shelly":
			w.Write([]byte(fmt.Sprintf(`{"gen":3,"app":"%s"}`, model)))
		case "/rpc/Shelly.GetDeviceInfo":
			w.Write([]byte(mockGen2DeviceInfoJSON(model, "shelly1g3-DDEEFF", "1.3.3")))
		default:
			assert.Fail(t, "unexpected device path: "+req.URL.Path)
		}
	}))
	defer deviceServer.Close()

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)

	otaUpdater, err := NewOTAService(
		WithAPIClient(
			NewAPIClient(
				WithBaseURL(shellyCloudAPIServer.URL),
				WithGen2BaseURL(shellyCloudAPIServer.URL),
			),
		),
		WithWaitTimeInSeconds(2),
		WithDevices([]string{fmt.Sprintf("127.0.0.1:%v", deviceServerPort)}),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	assert.Nil(t, err)

	devices, err := otaUpdater.DiscoverDevices()
	assert.Nil(t, err)
	assert.Len(t, devices, 1)

	for _, device := range devices {
		assert.Equal(t, model, device.Model)
		assert.Equal(t, "1.3.3", device.FirmwareVersion)
		assert.Equal(t, "1.5.0", device.FirmwareNewestVersion.Version)
		assert.Equal(t, 3, device.Generation)
	}
}

func TestGen4Upgradable(t *testing.T) {
	model := "1G4"
	apiModel := "S1G4"

	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(`{"isok": true, "data": {}}`))
			return
		}
		if req.URL.Path == "/update/"+apiModel {
			w.Write([]byte(mockGen2FirmwareVersion(model, "http://"+req.Host)))
			return
		}
		w.Write([]byte(`{"stable":{"version":"0.0.0","build_id":"","url":""},"beta":{"version":"","build_id":"","url":""}}`))
	}))
	defer shellyCloudAPIServer.Close()

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/shelly":
			w.Write([]byte(fmt.Sprintf(`{"gen":4,"app":"%s"}`, model)))
		case "/rpc/Shelly.GetDeviceInfo":
			w.Write([]byte(mockGen2DeviceInfoJSON(model, "shelly1g4-112233", "1.3.3")))
		default:
			assert.Fail(t, "unexpected device path: "+req.URL.Path)
		}
	}))
	defer deviceServer.Close()

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)

	otaUpdater, err := NewOTAService(
		WithAPIClient(
			NewAPIClient(
				WithBaseURL(shellyCloudAPIServer.URL),
				WithGen2BaseURL(shellyCloudAPIServer.URL),
			),
		),
		WithWaitTimeInSeconds(2),
		WithDevices([]string{fmt.Sprintf("127.0.0.1:%v", deviceServerPort)}),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	assert.Nil(t, err)

	devices, err := otaUpdater.DiscoverDevices()
	assert.Nil(t, err)
	assert.Len(t, devices, 1)

	for _, device := range devices {
		assert.Equal(t, model, device.Model)
		assert.Equal(t, "1.3.3", device.FirmwareVersion)
		assert.Equal(t, "1.5.0", device.FirmwareNewestVersion.Version)
		assert.Equal(t, 4, device.Generation)
	}
}

func TestForceUpgradeSkipsPrompts(t *testing.T) {
	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(mockSingleDeviceStableVersion("SHSW-25", "http://"+req.Host)))
			return
		}
		if strings.HasPrefix(req.URL.Path, "/update/") {
			w.Write([]byte(mockEmptyGen2FirmwareVersion()))
			return
		}
		if req.URL.Path == "/firmware/SHSW-25_build.zip" {
			w.Write([]byte(`firmware-data`))
			return
		}
	}))
	defer shellyCloudAPIServer.Close()

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/shelly":
			w.Write([]byte(`{"gen":1}`))
		case "/settings":
			w.Write([]byte(mockDeviceSettingsJSON("SHSW-25", "1CAAB5059F90", "20191127-095418/v1.5.6@0d769d69")))
		default:
			assert.Fail(t, "unexpected device path: "+req.URL.Path)
		}
	}))
	defer deviceServer.Close()

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)

	otaUpdater, err := NewOTAService(
		WithAPIClient(
			NewAPIClient(
				WithBaseURL(shellyCloudAPIServer.URL),
				WithGen2BaseURL(shellyCloudAPIServer.URL),
			),
		),
		WithForcedUpgrades(true),
		WithWaitTimeInSeconds(2),
		WithDevices([]string{fmt.Sprintf("127.0.0.1:%v", deviceServerPort)}),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	assert.Nil(t, err)

	// Verify the device was discovered and is upgradable.
	devices, err := otaUpdater.DiscoverDevices()
	assert.Nil(t, err)
	assert.Len(t, devices, 1)

	for _, device := range devices {
		assert.NotEqual(t, RemoteFirmware{}, device.FirmwareNewestVersion, "device should be upgradable")
	}

	// Verify force mode is set (force=true skips all survey prompts in PromptForUpgrades).
	assert.True(t, otaUpdater.force)
}

func TestSteppingStoneUpgrade(t *testing.T) {
	model := "Plus1"

	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(`{"isok": true, "data": {}}`))
			return
		}
		if req.URL.Path == "/update/"+model {
			w.Write([]byte(mockGen2FirmwareVersion(model, "http://"+req.Host)))
			return
		}
		w.Write([]byte(`{"stable":{"version":"0.0.0","build_id":"","url":""},"beta":{"version":"","build_id":"","url":""}}`))
	}))
	defer shellyCloudAPIServer.Close()

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/shelly":
			w.Write([]byte(fmt.Sprintf(`{"gen":2,"app":"%s"}`, model)))
		case "/rpc/Shelly.GetDeviceInfo":
			w.Write([]byte(mockGen2DeviceInfoJSON(model, "shellyplus1-AABBCC", "1.0.0")))
		default:
			assert.Fail(t, "unexpected device path: "+req.URL.Path)
		}
	}))
	defer deviceServer.Close()

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)

	otaUpdater, err := NewOTAService(
		WithAPIClient(
			NewAPIClient(
				WithBaseURL(shellyCloudAPIServer.URL),
				WithGen2BaseURL(shellyCloudAPIServer.URL),
			),
		),
		WithWaitTimeInSeconds(2),
		WithDevices([]string{fmt.Sprintf("127.0.0.1:%v", deviceServerPort)}),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	assert.Nil(t, err)

	devices, err := otaUpdater.DiscoverDevices()
	assert.Nil(t, err)
	assert.Len(t, devices, 1)

	for _, device := range devices {
		assert.Equal(t, model, device.Model)
		assert.Equal(t, "1.0.0", device.FirmwareVersion)
		assert.Equal(t, 2, device.Generation)
		// Should be set to stepping-stone 1.3.3, not the latest 1.5.0.
		assert.Equal(t, "1.3.3", device.FirmwareNewestVersion.Version)
		assert.Contains(t, device.FirmwareNewestVersion.URL, "fwcdn.shelly.cloud")
	}
}

func TestResetDiscovery(t *testing.T) {
	model := "Plus1"

	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(`{"isok": true, "data": {}}`))
			return
		}
		if req.URL.Path == "/update/"+model {
			w.Write([]byte(mockGen2FirmwareVersion(model, "http://"+req.Host)))
			return
		}
		w.Write([]byte(`{"stable":{"version":"0.0.0","build_id":"","url":""},"beta":{"version":"","build_id":"","url":""}}`))
	}))
	defer shellyCloudAPIServer.Close()

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/shelly":
			w.Write([]byte(fmt.Sprintf(`{"gen":2,"app":"%s"}`, model)))
		case "/rpc/Shelly.GetDeviceInfo":
			w.Write([]byte(mockGen2DeviceInfoJSON(model, "shellyplus1-AABBCC", "1.3.3")))
		default:
			assert.Fail(t, "unexpected device path: "+req.URL.Path)
		}
	}))
	defer deviceServer.Close()

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)

	otaUpdater, err := NewOTAService(
		WithAPIClient(
			NewAPIClient(
				WithBaseURL(shellyCloudAPIServer.URL),
				WithGen2BaseURL(shellyCloudAPIServer.URL),
			),
		),
		WithWaitTimeInSeconds(2),
		WithDevices([]string{fmt.Sprintf("127.0.0.1:%v", deviceServerPort)}),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	assert.Nil(t, err)

	devices, err := otaUpdater.DiscoverDevices()
	assert.Nil(t, err)
	assert.Len(t, devices, 1)

	// After reset, DiscoverDevices should re-discover.
	otaUpdater.ResetDiscovery()
	devices, err = otaUpdater.DiscoverDevices()
	assert.Nil(t, err)
	assert.Len(t, devices, 1)
}

func TestMultiPassUpgradeFlow(t *testing.T) {
	model := "Plus1"
	pass := 0

	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(`{"isok": true, "data": {}}`))
			return
		}
		if req.URL.Path == "/update/"+model {
			w.Write([]byte(mockGen2FirmwareVersion(model, "http://"+req.Host)))
			return
		}
		if strings.HasSuffix(req.URL.Path, "_stable.zip") {
			w.Write([]byte(`firmware-data`))
			return
		}
		w.Write([]byte(`{"stable":{"version":"0.0.0","build_id":"","url":""},"beta":{"version":"","build_id":"","url":""}}`))
	}))
	defer shellyCloudAPIServer.Close()

	steppingStoneFW := steppingStone133[model]

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/shelly":
			w.Write([]byte(fmt.Sprintf(`{"gen":2,"app":"%s"}`, model)))
		case "/rpc/Shelly.GetDeviceInfo":
			// First pass: device at 1.0.0 -> needs stepping stone
			// Second pass: device at 1.3.3 -> needs latest
			version := "1.0.0"
			if pass > 0 {
				version = "1.3.3"
			}
			w.Write([]byte(mockGen2DeviceInfoJSON(model, "shellyplus1-AABBCC", version)))
		default:
			// Accept OTA requests silently.
		}
	}))
	defer deviceServer.Close()

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)

	otaUpdater, err := NewOTAService(
		WithAPIClient(
			NewAPIClient(
				WithBaseURL(shellyCloudAPIServer.URL),
				WithGen2BaseURL(shellyCloudAPIServer.URL),
			),
		),
		WithForcedUpgrades(true),
		WithWaitTimeInSeconds(2),
		WithDevices([]string{fmt.Sprintf("127.0.0.1:%v", deviceServerPort)}),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	assert.Nil(t, err)

	// Pass 1: device needs stepping stone.
	devices, _ := otaUpdater.DiscoverDevices()
	assert.Len(t, devices, 1)
	for _, device := range devices {
		assert.Equal(t, steppingStoneFW.Version, device.FirmwareNewestVersion.Version)
	}

	// Simulate pass 1 completing and device rebooting at 1.3.3.
	pass = 1
	otaUpdater.ResetDiscovery()

	// Clear API firmware cache to re-fetch.
	otaUpdater.api.firmwares = nil

	err = otaUpdater.refreshFirmwareTargets()
	assert.Nil(t, err)

	// Pass 2: device at 1.3.3, should target latest (1.5.0).
	devices, _ = otaUpdater.DiscoverDevices()
	assert.Len(t, devices, 1)
	for _, device := range devices {
		assert.Equal(t, "1.3.3", device.FirmwareVersion)
		assert.Equal(t, "1.5.0", device.FirmwareNewestVersion.Version)
	}
}

func TestShutdownClosesServer(t *testing.T) {
	model := "Plus1"

	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(`{"isok": true, "data": {}}`))
			return
		}
		if req.URL.Path == "/update/"+model {
			w.Write([]byte(mockGen2FirmwareVersion(model, "http://"+req.Host)))
			return
		}
		w.Write([]byte(`{"stable":{"version":"0.0.0","build_id":"","url":""},"beta":{"version":"","build_id":"","url":""}}`))
	}))
	defer shellyCloudAPIServer.Close()

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/shelly":
			w.Write([]byte(fmt.Sprintf(`{"gen":2,"app":"%s"}`, model)))
		case "/rpc/Shelly.GetDeviceInfo":
			w.Write([]byte(mockGen2DeviceInfoJSON(model, "shellyplus1-AABBCC", "1.3.3")))
		}
	}))
	defer deviceServer.Close()

	deviceServerURL, _ := url.Parse(deviceServer.URL)
	deviceServerPort, _ := strconv.Atoi(deviceServerURL.Port())

	otaUpdater, err := NewOTAService(
		WithAPIClient(
			NewAPIClient(
				WithBaseURL(shellyCloudAPIServer.URL),
				WithGen2BaseURL(shellyCloudAPIServer.URL),
			),
		),
		WithWaitTimeInSeconds(2),
		WithDevices([]string{fmt.Sprintf("127.0.0.1:%v", deviceServerPort)}),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	assert.Nil(t, err)
	assert.NotNil(t, otaUpdater.server)

	// Verify the OTA server is listening.
	otaPort := otaUpdater.serverPort
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", otaPort))
	assert.Nil(t, err)
	resp.Body.Close()

	// Shutdown should close the server.
	otaUpdater.Shutdown()

	// After shutdown, connections should be refused.
	_, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/", otaPort))
	assert.NotNil(t, err)
}

func TestShutdownNilServer(t *testing.T) {
	// Calling Shutdown on an OTAService that never called Setup should not panic.
	otaUpdater := OTAService{}
	otaUpdater.Shutdown()
}

func TestFetchGen1Version(t *testing.T) {
	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/settings" {
			w.Write([]byte(`{"device":{"type":"SHSW-25","mac":"AABBCC","hostname":"shelly-AABBCC"},"fw":"20200309-104051/v1.6.0@43056d58","name":""}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer deviceServer.Close()

	otaUpdater := OTAService{}
	client := &http.Client{Timeout: 2 * time.Second}
	device := &Device{Model: "SHSW-25", Generation: 1}

	version := otaUpdater.fetchGen1Version(client, deviceServer.URL, device)
	assert.Equal(t, "20200309-104051/v1.6.0@43056d58", version)
}

func TestFetchGen1VersionBadResponse(t *testing.T) {
	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer deviceServer.Close()

	otaUpdater := OTAService{}
	client := &http.Client{Timeout: 2 * time.Second}
	device := &Device{Model: "SHSW-25", Generation: 1}

	version := otaUpdater.fetchGen1Version(client, deviceServer.URL, device)
	assert.Equal(t, "", version)
}

func TestFetchGen2Version(t *testing.T) {
	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/rpc/Shelly.GetDeviceInfo" {
			w.Write([]byte(`{"id":"shellyplus1-AABBCC","app":"Plus1","ver":"1.5.0","name":""}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer deviceServer.Close()

	otaUpdater := OTAService{}
	client := &http.Client{Timeout: 2 * time.Second}
	device := &Device{Model: "Plus1", Generation: 2}

	version := otaUpdater.fetchGen2Version(client, deviceServer.URL, device)
	assert.Equal(t, "1.5.0", version)
}

func TestFetchGen2VersionBadResponse(t *testing.T) {
	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer deviceServer.Close()

	otaUpdater := OTAService{}
	client := &http.Client{Timeout: 2 * time.Second}
	device := &Device{Model: "Plus1", Generation: 2}

	version := otaUpdater.fetchGen2Version(client, deviceServer.URL, device)
	assert.Equal(t, "", version)
}

func mockDeviceSettingsJSON(model string, mac string, version string) string {
	return fmt.Sprintf(`{
		"device": {
			"type": "%v",
			"mac": "%v",
			"hostname": "shelly-%v",
			"num_outputs": 2,
			"num_meters": 2,
			"num_rollers": 1
		},
		"name": "",
		"fw": "%v",
		"factory_reset_from_switch": true,
		"discoverable": false,
		"build_info": {
			"build_id": "%v",
			"build_timestamp": "2020-03-09T10:40:51Z",
			"build_version": "1.0"
		},
		"hwinfo": {
			"hw_revision": "prod-191217",
			"batch_id": 1
		}
	}`, model, mac, mac, version, version)
}

func mockSingleDeviceStableVersion(model string, serverURL string) string {
	return fmt.Sprintf(`{
		"isok": true,
		"data": {
			"%v": {
				"url": "%v/firmware/%v_build.zip",
				"version": "20200309-104051/v1.6.0@43056d58"
			}
		}
	}`, model, serverURL, model)
}

func mockGen2DeviceInfoJSON(model string, id string, version string) string {
	return fmt.Sprintf(`{
		"id": "%v",
		"app": "%v",
		"ver": "%v",
		"name": ""
	}`, id, model, version)
}

func mockEmptyGen2FirmwareVersion() string {
	return `{"stable":{"version":"0.0.0","build_id":"","url":""},"beta":{"version":"","build_id":"","url":""}}`
}

func mockGen2FirmwareVersion(model string, serverURL string) string {
	return fmt.Sprintf(`{
		"stable": {
			"version": "1.5.0",
			"build_id": "build-stable",
			"url": "%v/firmware/%v_stable.zip"
		},
		"beta": {
			"version": "1.6.0-beta",
			"build_id": "build-beta",
			"url": "%v/firmware/%v_beta.zip"
		}
	}`, serverURL, model, serverURL, model)
}
