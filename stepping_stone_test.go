package main

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input   string
		major   int
		minor   int
		patch   int
		wantErr bool
	}{
		{"1.3.3", 1, 3, 3, false},
		{"1.0.0", 1, 0, 0, false},
		{"2.10.5", 2, 10, 5, false},
		{"0.0.1", 0, 0, 1, false},
		{"", 0, 0, 0, true},
		{"abc", 0, 0, 0, true},
		{"1.3", 0, 0, 0, true},
		{"1.3.3.4", 0, 0, 0, true},
		{"a.b.c", 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			major, minor, patch, err := parseVersion(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.major, major)
				assert.Equal(t, tt.minor, minor)
				assert.Equal(t, tt.patch, patch)
			}
		})
	}
}

func TestIsVersionLessThan(t *testing.T) {
	tests := []struct {
		a      string
		b      string
		expect bool
	}{
		{"1.0.0", "1.3.3", true},
		{"1.3.2", "1.3.3", true},
		{"1.3.3", "1.3.3", false},
		{"1.3.4", "1.3.3", false},
		{"1.4.0", "1.3.3", false},
		{"2.0.0", "1.3.3", false},
		{"0.9.9", "1.0.0", true},
		{"1.2.0", "1.3.0", true},
		// Invalid versions return false.
		{"invalid", "1.3.3", false},
		{"1.3.3", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			assert.Equal(t, tt.expect, isVersionLessThan(tt.a, tt.b))
		})
	}
}

func TestNeedsSteppingStone(t *testing.T) {
	device := &Device{
		Model:           "Plus1",
		FirmwareVersion: "1.0.0",
		Generation:      2,
		ID:              "shellyplus1-aabbcc",
		IP:              net.ParseIP("192.168.1.100"),
		Port:            80,
	}

	fw, needed := NeedsSteppingStone(device)
	assert.True(t, needed)
	assert.Equal(t, "1.3.3", fw.Version)
	assert.Equal(t, "Plus1", fw.Model)
	assert.Contains(t, fw.URL, "fwcdn.shelly.cloud")
}

func TestNeedsSteppingStoneNotNeeded(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"at_threshold", "1.3.3"},
		{"above_threshold", "1.4.0"},
		{"well_above", "2.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &Device{
				Model:           "Plus1",
				FirmwareVersion: tt.version,
				Generation:      2,
				ID:              "shellyplus1-aabbcc",
				IP:              net.ParseIP("192.168.1.100"),
				Port:            80,
			}

			fw, needed := NeedsSteppingStone(device)
			assert.False(t, needed)
			assert.Equal(t, RemoteFirmware{}, fw)
		})
	}
}

func TestNeedsSteppingStoneGen1Ignored(t *testing.T) {
	device := &Device{
		Model:           "SHSW-25",
		FirmwareVersion: "1.0.0",
		Generation:      1,
		ID:              "shelly25-aabbcc",
		IP:              net.ParseIP("192.168.1.100"),
		Port:            80,
	}

	fw, needed := NeedsSteppingStone(device)
	assert.False(t, needed)
	assert.Equal(t, RemoteFirmware{}, fw)
}

func TestNeedsSteppingStoneUnknownModel(t *testing.T) {
	device := &Device{
		Model:           "UnknownModelXYZ",
		FirmwareVersion: "1.0.0",
		Generation:      2,
		ID:              "shellyunknown-aabbcc",
		IP:              net.ParseIP("192.168.1.100"),
		Port:            80,
	}

	fw, needed := NeedsSteppingStone(device)
	assert.False(t, needed)
	assert.Equal(t, RemoteFirmware{}, fw)
}

func TestNeedsSteppingStoneAllModels(t *testing.T) {
	for model := range steppingStone133 {
		t.Run(model, func(t *testing.T) {
			device := &Device{
				Model:           model,
				FirmwareVersion: "1.0.0",
				Generation:      2,
				ID:              "test-" + model,
				IP:              net.ParseIP("192.168.1.100"),
				Port:            80,
			}

			fw, needed := NeedsSteppingStone(device)
			assert.True(t, needed)
			assert.Equal(t, "1.3.3", fw.Version)
			assert.Equal(t, model, fw.Model)
		})
	}
}

func TestSteppingStoneURLFormat(t *testing.T) {
	for model, fw := range steppingStone133 {
		t.Run(model, func(t *testing.T) {
			assert.Contains(t, fw.URL, "fwcdn.shelly.cloud/gen2/")
			assert.NotContains(t, fw.URL, ".zip", "CDN URLs should not have .zip extension")
		})
	}
}

func TestNeedsSteppingStoneGen4Ignored(t *testing.T) {
	device := &Device{
		Model:           "1G4",
		FirmwareVersion: "1.4.0",
		Generation:      4,
		ID:              "shelly1g4-aabbcc",
		IP:              net.ParseIP("192.168.1.100"),
		Port:            80,
	}

	fw, needed := NeedsSteppingStone(device)
	assert.False(t, needed)
	assert.Equal(t, RemoteFirmware{}, fw)
}
