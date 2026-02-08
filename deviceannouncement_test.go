package main

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviceAnnouncementString(t *testing.T) {
	da := DeviceAnnouncement{
		IP:       net.ParseIP("192.168.1.100"),
		HostName: "shellyswitch25-ABC",
		Port:     80,
	}
	assert.Equal(t, "shellyswitch25-ABC (192.168.1.100:80)", da.String())
}

func TestDeviceInformationURLGen1(t *testing.T) {
	da := DeviceAnnouncement{
		IP:         net.ParseIP("192.168.1.100"),
		Port:       80,
		Generation: 1,
	}
	url := da.DeviceInformationURL("admin", "pass")
	assert.Contains(t, url, "/settings")
	assert.NotContains(t, url, "/rpc/Shelly.GetDeviceInfo")
}

func TestDeviceInformationURLGen2(t *testing.T) {
	da := DeviceAnnouncement{
		IP:         net.ParseIP("192.168.1.100"),
		Port:       80,
		Generation: 2,
	}
	url := da.DeviceInformationURL("admin", "pass")
	assert.Contains(t, url, "/rpc/Shelly.GetDeviceInfo")
	assert.NotContains(t, url, "/settings")
}

func TestDeviceInformationURLGen3(t *testing.T) {
	da := DeviceAnnouncement{
		IP:         net.ParseIP("192.168.1.100"),
		Port:       80,
		Generation: 3,
	}
	url := da.DeviceInformationURL("admin", "pass")
	assert.Contains(t, url, "/rpc/Shelly.GetDeviceInfo")
}

func TestDeviceInformationURLGen4(t *testing.T) {
	da := DeviceAnnouncement{
		IP:         net.ParseIP("192.168.1.100"),
		Port:       80,
		Generation: 4,
	}
	url := da.DeviceInformationURL("admin", "pass")
	assert.Contains(t, url, "/rpc/Shelly.GetDeviceInfo")
}
