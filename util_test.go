package main

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerPort(t *testing.T) {
	port, err := ServerPort()
	assert.Nil(t, err)
	assert.Greater(t, port, 0)
}

func TestExpandCIDR(t *testing.T) {
	ips, err := ExpandCIDR("192.168.100.0/24")
	assert.Nil(t, err)
	assert.Equal(t, 254, len(ips))
	assert.Equal(t, "192.168.100.1", ips[0].String())
	assert.Equal(t, "192.168.100.254", ips[len(ips)-1].String())
}

func TestExpandCIDRSmall(t *testing.T) {
	ips, err := ExpandCIDR("10.0.0.0/30")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(ips))
	assert.Equal(t, "10.0.0.1", ips[0].String())
	assert.Equal(t, "10.0.0.2", ips[1].String())
}

func TestExpandCIDRInvalid(t *testing.T) {
	_, err := ExpandCIDR("not-a-cidr")
	assert.NotNil(t, err)
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"10.x private", "10.0.0.1", true},
		{"172.16.x private", "172.16.0.1", true},
		{"192.168.x private", "192.168.1.1", true},
		{"public IP", "8.8.8.8", false},
		{"loopback", "127.0.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			assert.Equal(t, tt.expected, isPrivateIP(ip))
		})
	}
}
