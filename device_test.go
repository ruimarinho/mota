package main

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviceBaseURL(t *testing.T) {
	d := Device{
		Username: "admin",
		Password: "secret",
		IP:       net.ParseIP("192.168.1.100"),
		Port:     80,
	}
	assert.Equal(t, "http://admin:secret@192.168.1.100:80", d.BaseURL())
}

func TestDeviceOTAURL(t *testing.T) {
	d := Device{
		Username: "admin",
		Password: "secret",
		IP:       net.ParseIP("192.168.1.100"),
		Port:     80,
	}
	expected := "http://admin:secret@192.168.1.100:80/ota?url=http://10.0.0.1:8080/firmware.zip"
	assert.Equal(t, expected, d.OTAURL("10.0.0.1", 8080, "firmware.zip"))
}

func TestDeviceFamilyFriendlyName(t *testing.T) {
	t.Run("known model", func(t *testing.T) {
		d := Device{Model: "SHSW-25"}
		assert.Equal(t, "Shelly 25", d.FamilyFriendlyName())
	})

	t.Run("unknown model", func(t *testing.T) {
		d := Device{Model: "UNKNOWN-MODEL"}
		assert.Equal(t, "UNKNOWN-MODEL", d.FamilyFriendlyName())
	})
}

func TestDeviceString(t *testing.T) {
	t.Run("with name", func(t *testing.T) {
		d := Device{
			Name: "Living Room",
			ID:   "shelly-ABC",
			IP:   net.ParseIP("192.168.1.100"),
			Port: 80,
		}
		assert.Equal(t, "Living Room (shelly-ABC@192.168.1.100:80)", d.String())
	})

	t.Run("without name", func(t *testing.T) {
		d := Device{
			Model: "SHSW-25",
			ID:    "shelly-ABC",
			IP:    net.ParseIP("192.168.1.100"),
			Port:  80,
		}
		assert.Equal(t, "Shelly 25 (shelly-ABC@192.168.1.100:80)", d.String())
	})
}
