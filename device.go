package main

import (
	"fmt"
	"net"
)

var shellies = map[string]string{
	"SH2LED-1":   "Shelly 2 LED",
	"SHAIR-1":    "Shelly Air",
	"SHBDUO-1":   "Shelly Bulb Duo",
	"SHBLB-1":    "Shelly Bulb",
	"SHBTN-1":    "Shelly Button 1",
	"SHBTN-2":    "Shelly Button 1 (Rev. 2)",
	"SHCB-1":     "Shelly Color Bulb RGBW GU10",
	"SHCL-255":   "Shelly Color",
	"SHDIMW-1":   "Shelly Dimmer W1",
	"SHDM-1":     "Shelly Dimmer",
	"SHDM-2":     "Shelly Dimmer 2",
	"SHDW-1":     "Shelly Door/Window Sensor",
	"SHDW-2":     "Shelly Door/Window Sensor 2",
	"SHEM-3":     "Shelly 3EM",
	"SHEM":       "Shelly EM",
	"SHGS-1":     "Shelly Gas",
	"SHHT-1":     "Shelly H&T",
	"SHIX3-1":    "Shelly i3",
	"SHMOS-01":   "Shelly Motion",
	"SHPLG-1":    "Shelly Plug 1",
	"SHPLG-S":    "Shelly Plug S",
	"SHPLG-U1":   "Shelly Plug US",
	"SHPLG2-1":   "Shelly Plug 2",
	"SHRGBW2":    "Shelly RGBW2",
	"SHRGBWW-01": "Shelly RGBW",
	"SHSEN-1":    "Shelly Sense",
	"SHSM-01":    "Shelly Smoke",
	"SHSM-02":    "Shelly Smoke",
	"SHSPOT-1":   "Shelly Spot",
	"SHSPOT-2":   "Shelly Spot 2",
	"SHSW-1":     "Shelly 1",
	"SHSW-21":    "Shelly 2",
	"SHSW-22":    "Shelly HD",
	"SHSW-25":    "Shelly 2.5",
	"SHSW-44":    "Shelly 4 Pro",
	"SHSW-L":     "Shelly 1L",
	"SHSW-PM":    "Shelly 1PM",
	"SHUNI-1":    "Shelly Uni",
	"SHVIN-1":    "Shelly Vintage",
	"SHWT-1":     "Shelly Flood",
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
