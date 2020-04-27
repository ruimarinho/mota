package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	zeroconf "github.com/grandcat/zeroconf"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

func TestNonUpgradable(t *testing.T) {
	var firmwares string
	shellyAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(firmwares))
			return
		}

		assert.Fail(t, req.URL.Path)
	}))

	firmwares = fmt.Sprintf(`{
		"isok": true,
		"data": {
			"SHSW-1": {
				"url": "%v/firmware/SHSW-1_build.zip",
				"version": "20200320-123430/v1.6.2@514044b4"
			},
			"SHSW-25": {
				"url": "%v/firmware/SHSW-25_build.zip",
				"version": "20200309-104051/v1.6.0@43056d58"
			}
		}
	}`, shellyAPIServer.URL, shellyAPIServer.URL)

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/settings", req.URL.Path)
		w.Write([]byte(`{
			"device": {
				"type": "SHSW-25",
				"mac": "0D3595FDAE25",
				"hostname": "shellyswitch25-0D3595FDAE25",
				"num_outputs": 2,
				"num_meters": 2,
				"num_rollers": 1
			},
			"name": "",
			"fw": "20200309-104051/v1.6.0@43056d58",
			"factory_reset_from_switch": true,
			"discoverable": false,
			"build_info": {
				"build_id": "20200309-104051/v1.6.0@43056d58",
				"build_timestamp": "2020-03-09T10:40:51Z",
				"build_version": "1.0"
			},
			"hwinfo": {
				"hw_revision": "prod-191217",
				"batch_id": 1
			}
		}`))
	}))

	updaterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/SHSW-25", req.URL.Path)
		w.Write([]byte(`{OK}`))
	}))

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)

	updaterServerURL, err := url.Parse(updaterServer.URL)
	updaterServerPort, err := strconv.Atoi(updaterServerURL.Port())
	assert.Nil(t, err)

	zeroconfServer, err := zeroconf.RegisterProxy("shelly-non-upgradable", "_httptest._tcp.", "local.", deviceServerPort, "127.0.0.1", []string{"127.0.0.1"}, []string{"id=shellyswitch25-0D3595", "fw_id=20200309-104051/v1.6.0@43056d58", "arch=esp8266"}, nil)
	assert.Nil(t, err)
	defer zeroconfServer.Shutdown()

	updater, err := NewOTAUpdater(updaterServerPort, "_httptest._tcp.", "local.", 2, WithAPIClient(NewAPIClient(WithBaseURL(fmt.Sprintf(shellyAPIServer.URL)))))
	assert.Nil(t, err)

	err = updater.Start()
	assert.Nil(t, err)

	devices, err := updater.Devices()
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
	var firmwares string
	shellyAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(fmt.Sprintf(firmwares)))
			return
		}

		if req.URL.Path == "/firmware/SHSW-25_build.zip" {
			w.Write([]byte(`{OK}`))
			return
		}

		assert.Fail(t, req.URL.Path)
	}))

	firmwares = fmt.Sprintf(`{
		"isok": true,
		"data": {
			"SHSW-1": {
				"url": "%v/firmware/SHSW-1_build.zip",
				"version": "20200320-123430/v1.6.2@514044b4"
			},
			"SHSW-25": {
				"url": "%v/firmware/SHSW-25_build.zip",
				"version": "20200309-104051/v1.6.0@43056d58"
			}
		}
	}`, shellyAPIServer.URL, shellyAPIServer.URL)

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/settings", req.URL.Path)
		w.Write([]byte(`{
			"device": {
				"type": "SHSW-25",
				"mac": "1CAAB5059F90",
				"hostname": "shellyswitch25-1CAAB5059F90",
				"num_outputs": 2,
				"num_meters": 2,
				"num_rollers": 1
			},
			"name": "",
			"fw": "20191127-095418/v1.5.6@0d769d69",
			"factory_reset_from_switch": true,
			"discoverable": false,
			"build_info": {
				"build_id": "20191127-095418/v1.5.6@0d769d69",
				"build_timestamp": "2020-03-09T10:40:51Z",
				"build_version": "1.0"
			},
			"hwinfo": {
				"hw_revision": "prod-191217",
				"batch_id": 1
			}
		}`))
	}))

	updaterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/SHSW-25", req.URL.Path)
	}))

	deviceServerURL, err := url.Parse(deviceServer.URL)
	assert.Nil(t, err)
	deviceServerPort, err := strconv.Atoi(deviceServerURL.Port())
	assert.Nil(t, err)

	updaterServerURL, err := url.Parse(updaterServer.URL)
	updaterServerPort, err := strconv.Atoi(updaterServerURL.Port())
	assert.Nil(t, err)

	zeroconfServer, err := zeroconf.RegisterProxy("shelly-upgradable", "_httptest._tcp.", "local.", deviceServerPort, "127.0.0.1", []string{"127.0.0.1"}, []string{"id=shellyswitch25-1CAAB5", "fw_id=20191127-095418/v1.5.6@0d769d69", "arch=esp8266"}, nil)
	assert.Nil(t, err)
	defer zeroconfServer.Shutdown()

	updater, err := NewOTAUpdater(updaterServerPort, "_httptest._tcp.", "local.", 2, WithAPIClient(NewAPIClient(WithBaseURL(fmt.Sprintf(shellyAPIServer.URL)))))
	assert.Nil(t, err)

	err = updater.Start()
	assert.Nil(t, err)

	devices, err := updater.Devices()
	assert.Nil(t, err)
	assert.Len(t, devices, 1)

	for _, device := range devices {
		assert.Equal(t, device.Port, deviceServerPort)
		assert.Equal(t, device.IP.String(), deviceServerURL.Hostname())
		assert.Equal(t, "20191127-095418/v1.5.6@0d769d69", device.CurrentFWVersion)
		assert.Equal(t, "20200309-104051/v1.6.0@43056d58", device.NewFWVersion)
	}
}
