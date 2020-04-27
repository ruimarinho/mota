package main

import (
	"fmt"
	"net"
)

var shellies = map[string]string{
	"SHSW-1":     "Shelly 1",
	"SHSW-PM":    "Shelly 1PM",
	"SHSW-21":    "Shelly 2",
	"SHSW-22":    "Shelly HD",
	"SHSW-25":    "Shelly 2.5",
	"SH2LED":     "Shelly 2 LED",
	"SHSW-44":    "Shelly 4 Pro",
	"SHEM":       "Shelly EM",
	"SHEM-3":     "Shelly 3EM",
	"SHPLG-1":    "Shelly Plug 1",
	"SHPLG2-1":   "Shelly Plug 2",
	"SHPLG-S":    "Shelly Plug S",
	"SHBLB-1":    "Shelly Bulb",
	"SHBDUO-1":   "Shelly Bulb Duo",
	"SHVIN-1":    "Shelly Vintage",
	"SHRGBWW-01": "Shelly RGBW",
	"SHRGBW2":    "Shelly RGBW2",
	"SHHT-1":     "Shelly H&T",
	"SHWT-1":     "Shelly Flood",
	"SHSM-01":    "Shelly Smoke",
	"SHSEN-1":    "Shelly Sense",
	"SHDM-1":     "Shelly Dimmer",
	"SHDW-1":     "Shelly Door/Window Sensor",
}

// Device holds information about the device location, authentication
// requirements and firmware versions.
type Device struct {
	CurrentFWVersion string
	HostName         string
	IP               net.IP
	Model            string
	NewFWVersion     string
	Password         string
	Port             int
	Username         string
}

// Settings is the structure holding information about the device
// model type and current firmware version.
type Settings struct {
	Device struct {
		Type string `json:"type"`
	} `json:"device"`
	FW string `json:"fw"`
}

// GetBaseURL returns the full URL required for API authentication,
// if needed.
func (d *Device) GetBaseURL() string {
	return fmt.Sprintf("http://%v:%v@%v:%v", d.Username, d.Password, d.IP.String(), d.Port)
}

// ModelName returns a human-friendly version of the device's model,
// if available.
func (d *Device) ModelName() string {
	if d.Model != "" && shellies[d.Model] != "" {
		return shellies[d.Model]
	}

	return d.Model
}

func (d *Device) String() string {
	return fmt.Sprintf("%v (%v:%v)", d.HostName, d.IP.String(), d.Port)
}
