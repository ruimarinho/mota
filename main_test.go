package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	zeroconf "github.com/grandcat/zeroconf"
	"github.com/stretchr/testify/assert"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

func TestNonUpgradable(t *testing.T) {
	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(mockSingleDeviceStableVersion("SHSW-25", "http://"+req.Host)))
			return
		}
		assert.Fail(t, req.URL.Path)
	}))

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/settings", req.URL.Path)
		w.Write([]byte(mockDeviceSettingsJSON("SHSW-25", "1CAAB5059F90", "20200309-104051/v1.6.0@43056d58")))
	}))

	otaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/SHSW-25", req.URL.Path)
		w.Write([]byte(`{OK}`))
	}))

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)
	otaServerURL, err := url.Parse(otaServer.URL)
	assert.Nil(t, err)
	otaServerPort, err := strconv.Atoi(otaServerURL.Port())
	assert.Nil(t, err)

	zeroconfServer, err := zeroconf.RegisterProxy("shelly-non-upgradable", "_httptest._tcp.", "local.", deviceServerPort, "shellyswitch25-0D3595FDAE25", []string{"127.0.0.1"}, []string{"id=shellyswitch25-0D3595FDAE25", "fw_id=20200309-104051/v1.6.0@43056d58", "arch=esp8266"}, nil)
	assert.Nil(t, err)
	defer zeroconfServer.Shutdown()

	otaUpdater, err := NewOTAUpdater(
		WithAPIClient(
			NewAPIClient(WithBaseURL(fmt.Sprintf(shellyCloudAPIServer.URL))),
		),
		WithServerPort(otaServerPort),
		WithService("_httptest._tcp."),
		WithWaitTimeInSeconds(2),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	if err != nil {
		log.Fatal(err)
	}

	devices, err := otaUpdater.Devices()
	assert.Nil(t, err)
	assert.Len(t, devices, 1)

	for _, device := range devices {
		assert.Equal(t, device.Port, deviceServerPort)
		assert.Equal(t, device.IP.String(), deviceServerURL.Hostname())
		assert.Equal(t, "20200309-104051/v1.6.0@43056d58", device.CurrentFWVersion)
		assert.Equal(t, "20200309-104051/v1.6.0@43056d58", device.NewFWVersion)
	}
}

func TestUpgradable(t *testing.T) {
	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(mockSingleDeviceStableVersion("SHSW-25", "http://"+req.Host)))
			return
		}

		if req.URL.Path == "/firmware/SHSW-25_build.zip" {
			w.Write([]byte(`{OK}`))
			return
		}
		assert.Fail(t, req.URL.Path)
	}))

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/settings", req.URL.Path)
		w.Write([]byte(mockDeviceSettingsJSON("SHSW-25", "1CAAB5059F90", "20191127-095418/v1.5.6@0d769d69")))
	}))

	otaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/SHSW-25", req.URL.Path)
	}))

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)
	otaServerURL, err := url.Parse(otaServer.URL)
	assert.Nil(t, err)
	otaServerPort, err := strconv.Atoi(otaServerURL.Port())
	assert.Nil(t, err)

	zeroconfServer, err := zeroconf.RegisterProxy("shelly-upgradable", "_httptest._tcp.", "local.", deviceServerPort, "shellyswitch25-1CAAB5", []string{"127.0.0.1"}, []string{"id=shellyswitch25-1CAAB5", "fw_id=20191127-095418/v1.5.6@0d769d69", "arch=esp8266"}, nil)
	assert.Nil(t, err)
	defer zeroconfServer.Shutdown()

	otaUpdater, err := NewOTAUpdater(
		WithAPIClient(
			NewAPIClient(WithBaseURL(fmt.Sprintf(shellyCloudAPIServer.URL))),
		),
		WithServerPort(otaServerPort),
		WithService("_httptest._tcp."),
		WithWaitTimeInSeconds(2),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	if err != nil {
		log.Fatal(err)
	}

	devices, err := otaUpdater.Devices()
	assert.Nil(t, err)
	assert.Len(t, devices, 1)

	for _, device := range devices {
		assert.Equal(t, device.Port, deviceServerPort)
		assert.Equal(t, device.IP.String(), deviceServerURL.Hostname())
		assert.Equal(t, "20191127-095418/v1.5.6@0d769d69", device.CurrentFWVersion)
		assert.Equal(t, "20200309-104051/v1.6.0@43056d58", device.NewFWVersion)
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

		if req.URL.Path == "/firmware/SHSW-25_build_beta.zip" {
			w.Write([]byte(`OK`))
			return
		}
		assert.Fail(t, req.URL.Path)
	}))

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/settings", req.URL.Path)
		w.Write([]byte(mockDeviceSettingsJSON("SHSW-25", "1CAAB5059F90", "20191127-095418/v1.5.6@0d769d69")))
	}))

	otaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/SHSW-25", req.URL.Path)
		w.Write([]byte(`{OK}`))
	}))

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)
	otaServerURL, err := url.Parse(otaServer.URL)
	assert.Nil(t, err)
	otaServerPort, err := strconv.Atoi(otaServerURL.Port())
	assert.Nil(t, err)

	zeroconfServer, err := zeroconf.RegisterProxy("shelly-upgradable", "_httptest._tcp.", "local.", deviceServerPort, "shellyswitch25-1CAAB5059F90", []string{"127.0.0.1"}, []string{"id=shellyswitch25-1CAAB5059F90", "fw_id=20191127-095418/v1.5.6@0d769d69", "arch=esp8266"}, nil)
	assert.Nil(t, err)
	defer zeroconfServer.Shutdown()

	otaUpdater, err := NewOTAUpdater(
		WithAPIClient(
			NewAPIClient(WithBaseURL(fmt.Sprintf(shellyCloudAPIServer.URL))),
		),
		WithBetaVersions(true),
		WithServerPort(otaServerPort),
		WithService("_httptest._tcp."),
		WithWaitTimeInSeconds(2),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	if err != nil {
		log.Fatal(err)
	}

	devices, err := otaUpdater.Devices()
	assert.Nil(t, err)
	assert.Len(t, devices, 1)

	for _, device := range devices {
		assert.Equal(t, device.Port, deviceServerPort)
		assert.Equal(t, device.IP.String(), deviceServerURL.Hostname())
		assert.Equal(t, "20191127-095418/v1.5.6@0d769d69", device.CurrentFWVersion)
		assert.Equal(t, "20210122-154345/v1.10.0-rc1@00eeaa9b", device.NewFWVersion)
	}
}

func TestHosts(t *testing.T) {
	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(mockSingleDeviceStableVersion("SHSW-25", "http://"+req.Host)))
			return
		}

		if req.URL.Path == "/firmware/SHSW-25_build.zip" {
			w.Write([]byte(`{OK}`))
			return
		}
		assert.Fail(t, req.URL.Path)
	}))

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/settings", req.URL.Path)
		w.Write([]byte(mockDeviceSettingsJSON("SHSW-25", "1CAAB5059F90", "20191127-095418/v1.5.6@0d769d69")))
	}))

	otaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/SHSW-25", req.URL.Path)
		w.Write([]byte(`{OK}`))
	}))

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)
	otaServerURL, err := url.Parse(otaServer.URL)
	assert.Nil(t, err)
	otaServerPort, err := strconv.Atoi(otaServerURL.Port())
	assert.Nil(t, err)

	zeroconfServer, err := zeroconf.RegisterProxy("shelly-upgradable", "_httptest._tcp.", "local.", deviceServerPort, "shellyswitch25-1CAAB5059F90", []string{"127.0.0.1"}, []string{"id=shellyswitch25-1CAAB5059F90", "fw_id=20191127-095418/v1.5.6@0d769d69", "arch=esp8266"}, nil)
	assert.Nil(t, err)
	defer zeroconfServer.Shutdown()

	otaUpdater, err := NewOTAUpdater(
		WithAPIClient(
			NewAPIClient(WithBaseURL(fmt.Sprintf(shellyCloudAPIServer.URL))),
		),
		WithServerPort(otaServerPort),
		WithService("_httptest._tcp."),
		WithWaitTimeInSeconds(2),
		WithHosts([]string{fmt.Sprintf("127.0.0.1:%v", deviceServerPort)}),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	if err != nil {
		log.Fatal(err)
	}

	devices, err := otaUpdater.Devices()
	assert.Nil(t, err)
	assert.Len(t, devices, 1)

	for _, device := range devices {
		assert.Equal(t, device.Port, deviceServerPort)
		assert.Equal(t, device.IP.String(), deviceServerURL.Hostname())
		assert.Equal(t, "20191127-095418/v1.5.6@0d769d69", device.CurrentFWVersion)
		assert.Equal(t, "20200309-104051/v1.6.0@43056d58", device.NewFWVersion)
	}
}

func TestMalformedHosts(t *testing.T) {
	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(mockSingleDeviceStableVersion("SHSW-25", "http://"+req.Host)))
			return
		}

		if req.URL.Path == "/firmware/SHSW-25_build.zip" {
			w.Write([]byte(`{OK}`))
			return
		}
		assert.Fail(t, req.URL.Path)
	}))

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/settings", req.URL.Path)
		w.Write([]byte(mockDeviceSettingsJSON("SHSW-25", "1CAAB5059F90", "20191127-095418/v1.5.6@0d769d69")))
	}))

	otaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/SHSW-25", req.URL.Path)
		w.Write([]byte(`{OK}`))
	}))

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)
	otaServerURL, err := url.Parse(otaServer.URL)
	assert.Nil(t, err)
	otaServerPort, err := strconv.Atoi(otaServerURL.Port())
	assert.Nil(t, err)

	zeroconfServer, err := zeroconf.RegisterProxy("shelly-upgradable", "_httptest._tcp.", "local.", deviceServerPort, "shellyswitch25-1CAAB5059F90", []string{"127.0.0.1"}, []string{"id=shellyswitch25-1CAAB5059F90", "fw_id=20191127-095418/v1.5.6@0d769d69", "arch=esp8266"}, nil)
	assert.Nil(t, err)
	defer zeroconfServer.Shutdown()

	otaUpdater, err := NewOTAUpdater(
		WithAPIClient(
			NewAPIClient(WithBaseURL(fmt.Sprintf(shellyCloudAPIServer.URL))),
		),
		WithServerPort(otaServerPort),
		WithService("_httptest._tcp."),
		WithWaitTimeInSeconds(2),
		WithHosts([]string{"*"}),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	if err != nil {
		log.Fatal(err)
	}

	devices, err := otaUpdater.Devices()
	assert.Nil(t, err)
	assert.Len(t, devices, 0)
}

func TestMalformedHostPort(t *testing.T) {
	shellyCloudAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(mockSingleDeviceStableVersion("SHSW-25", "http://"+req.Host)))
			return
		}

		if req.URL.Path == "/firmware/SHSW-25_build.zip" {
			w.Write([]byte(`{OK}`))
			return
		}
		assert.Fail(t, req.URL.Path)
	}))

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/settings", req.URL.Path)
		w.Write([]byte(mockDeviceSettingsJSON("SHSW-25", "1CAAB5059F90", "20191127-095418/v1.5.6@0d769d69")))
	}))

	otaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/SHSW-25", req.URL.Path)
		w.Write([]byte(`{OK}`))
	}))

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)
	otaServerURL, err := url.Parse(otaServer.URL)
	assert.Nil(t, err)
	otaServerPort, err := strconv.Atoi(otaServerURL.Port())
	assert.Nil(t, err)

	zeroconfServer, err := zeroconf.RegisterProxy("shelly-upgradable", "_httptest._tcp.", "local.", deviceServerPort, "shellyswitch25-1CAAB5059F90", []string{"127.0.0.1"}, []string{"id=shellyswitch25-1CAAB5059F90", "fw_id=20191127-095418/v1.5.6@0d769d69", "arch=esp8266"}, nil)
	assert.Nil(t, err)
	defer zeroconfServer.Shutdown()

	otaUpdater, err := NewOTAUpdater(
		WithAPIClient(
			NewAPIClient(WithBaseURL(fmt.Sprintf(shellyCloudAPIServer.URL))),
		),
		WithServerPort(otaServerPort),
		WithService("_httptest._tcp."),
		WithWaitTimeInSeconds(2),
		WithHosts([]string{"192.168.1.100::80"}),
	)
	assert.Nil(t, err)

	err = otaUpdater.Setup()
	if err != nil {
		log.Fatal(err)
	}

	devices, err := otaUpdater.Devices()
	assert.Nil(t, err)
	assert.Len(t, devices, 0)
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
