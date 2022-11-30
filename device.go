package main

import (
	"fmt"
	"net"
)

var shellies = map[string]string{
	"SHPLG-1":         "Shelly Plug",
	"SHPLG-S":         "Shelly Plug S",
	"SHPLG-IT1":       "Shelly Plug IT",
	"SHPLG-UK1":       "Shelly Plug UK",
	"SHPLG-AU1":       "Shelly Plug AU",
	"SHPLG-U1":        "Shelly Plug US",
	"SHPLG2-1":        "Shelly Plug",
	"SHSK-1":          "Shelly Socket",
	"SHSW-1":          "Shelly 1",
	"SHSW-PM":         "Shelly 1 PM",
	"SHSW-L":          "Shelly 1L",
	"SHAIR-1":         "Shelly Air",
	"SHAIR-2":         "Shelly Air Turbo",
	"SHSW-21":         "Shelly 2",
	"SHSW-22":         "Shelly HDPro",
	"SHSW-25":         "Shelly 25",
	"SHSW-44":         "Shelly 4Pro",
	"SHEM-1":          "Shelly EM",
	"SHEM-3":          "Shelly EM3",
	"SHEM":            "Shelly EM",
	"SHSEN-1":         "Shelly Sense",
	"SHGS-1":          "Shelly Gas",
	"SHSM-01":         "Shelly Smoke",
	"SHHT-1":          "Shelly T&H",
	"SHDW-1":          "Shelly Door",
	"SHDW-2":          "Shelly Door 2",
	"SHWT-1":          "Shelly Flood",
	"SHCL-255":        "Shelly Bulb",
	"SHBLB-1":         "Shelly Bulb",
	"SHRGBWW-01":      "Shelly RGBWW",
	"SHRGBW2":         "Shelly RGBW 2",
	"SH2LED-1":        "Shelly 2LED",
	"SHDM-1":          "Shelly Dimmer",
	"SHDM-2":          "Shelly Dimmer 2",
	"SHDIMW-1":        "Shelly Dimmer",
	"SHVIN-1":         "Shelly Vintage",
	"SHBDUO-1":        "Shelly Duo",
	"SHCB-1":          "Shelly Color Bulb",
	"SHBTN-1":         "Shelly Button",
	"SHBTN-2":         "Shelly Button",
	"SHIX3-1":         "Shelly i3",
	"SHSW-1S":         "Shelly Harvia RSS",
	"SHUNI-1":         "Shelly Uni",
	"SHMOS-01":        "Shelly Motion Sensor",
	"SHSPOT-1":        "Shelly Spot",
	"SHSPOT-2":        "Shelly Spot 2",
	"SHTRV-01":        "Shelly TRV",
	"SHMOS-02":        "Shelly Motion 2",
	"SNSW-001X16EU":   "Shelly Plus 1",
	"SNSW-001X15UL":   "Shelly Plus 1",
	"SNSW-001P16EU":   "Shelly Plus 1 PM",
	"SNSW-001P15UL":   "Shelly Plus 1 PM",
	"SNSW-002X16EU":   "Shelly Plus 2",
	"SNSW-002P16EU":   "Shelly Plus 2 PM",
	"SNSW-102P16EU":   "Shelly Plus 2 PM",
	"SHPSW04P":        "Shelly 4Pro",
	"SPSW-004PE16EU":  "Shelly Pro 4 PM",
	"SPSW-104PE16EU":  "Shelly Pro 4 PM",
	"SPSW-001PE16EU":  "Shelly Pro 1 PM",
	"SPSW-101PE16EU":  "Shelly Pro 1 PM",
	"SPSW-201PE16EU":  "Shelly Pro 1 PM",
	"SPSW-001XE16EU":  "Shelly Pro 1",
	"SPSW-101XE16EU":  "Shelly Pro 1",
	"SPSW-201XE16EU":  "Shelly Pro 1",
	"SPSW-002PE16EU":  "Shelly Pro 2 PM",
	"SPSW-102PE16EU":  "Shelly Pro 2 PM",
	"SPSW-202PE16EU":  "Shelly Pro 2 PM",
	"SPSW-002XE16EU":  "Shelly Pro 2",
	"SPSW-102XE16EU":  "Shelly Pro 2",
	"SPSW-202XE16EU":  "Shelly Pro 2",
	"SPSW-003XE16EU":  "Shelly Pro 3",
	"SNSN-0024X":      "Shelly Plus I4",
	"SNSN-0D24X":      "Shelly Plus I4 DC",
	"SNPL-00116US":    "Shelly Plus Plug US",
	"SNDM-0013US":     "Shelly Plus Wall Dimmer",
	"SNDM-9995WW":     "Shelly Plus Dimmer",
	"SNSN-0013A":      "Shelly Plus H&T",
	"IR_REM-0":        "Shelly Remote",
	"IR_REM-1-remote": "Shelly Remote",
}

// Device holds information about the device location, authentication
// requirements and firmware versions.
type Device struct {
	App              string
	CurrentFWVersion string
	FWFilename       string
	Generation       int
	HostName         string
	IP               net.IP
	Mac              string
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
		Mac  string `json:"mac"`
		Type string `json:"type"`
	} `json:"device"`
	FW string `json:"fw"`
}

// Settings is the structure holding information about the device
// model type and current firmware version.
type SettingsGen2 struct {
	App   string `json:"app"`
	Mac   string `json:"mac"`
	Model string `json:"model"`
	FW    string `json:"ver"`
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
