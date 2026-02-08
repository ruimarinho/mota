package main

import (
	"fmt"
	"net"
)

var shellies = map[string]string{
	"0-10VDimmerG3":   "Shelly 0-10V Dimmer Gen3",
	"1G3":             "Shelly 1 Gen3",
	"1G4":             "Shelly 1 Gen4",
	"1MiniG3":         "Shelly 1 Mini Gen3",
	"1MiniG4":         "Shelly 1 Mini Gen4",
	"1PMG3":           "Shelly 1 PM Gen3",
	"1PMG4":           "Shelly 1 PM Gen4",
	"1PMMiniG3":       "Shelly 1 PM Mini Gen3",
	"2PMG3":           "Shelly 2 PM Gen3",
	"2PMG4":           "Shelly 2 PM Gen4",
	"4Pro":            "Shelly 4Pro",
	"DimmerG3":        "Shelly Dimmer Gen3",
	"DimmerG4":        "Shelly Dimmer Gen4",
	"EMMiniG4":        "Shelly EM Mini Gen4",
	"EMXG3":           "Shelly EM X Gen3",
	"FloodG3":         "Shelly Flood Gen3",
	"FloodG4":         "Shelly Flood Gen4",
	"HTG3":            "Shelly H&T Gen3",
	"IR_REM-0":        "Shelly Remote",
	"IR_REM-1-remote": "Shelly Remote",
	"Mini1G3":         "Shelly Mini 1 Gen3",
	"Mini1PMG3":       "Shelly Mini 1 PM Gen3",
	"MiniPMG3":        "Shelly Mini PM Gen3",
	"PlugSG3":         "Shelly Plug S Gen3",
	"PlugSG4":         "Shelly Plug S Gen4",
	"PlugUS":          "Shelly Plus Plug US",
	"Plus1":           "Shelly Plus 1",
	"Plus10V":         "Shelly Plus 0-10V Dimmer",
	"Plus1Mini":       "Shelly Plus 1 Mini",
	"Plus1PM":         "Shelly Plus 1",
	"Plus1PMMini":     "Shelly Plus 1 PM Mini",
	"Plus2":           "Shelly Plus 2",
	"Plus2PM":         "Shelly Plus 2 PM",
	"PlusHT":          "Shelly Plus H&T",
	"PlusI4":          "Shelly Plus I4",
	"PlusPlugIT":      "Shelly Plus Plug IT",
	"PlusPlugS":       "Shelly Plus Plug S",
	"PlusPlugUK":      "Shelly Plus Plug UK",
	"PlusPMMini":      "Shelly Plus PM Mini",
	"PlusWallDimmer":  "Shelly Plus Wall Dimmer",
	"Pro1":            "Shelly Pro 1",
	"Pro1PM":          "Shelly Pro 1 PM",
	"Pro2":            "Shelly Pro 2",
	"Pro2PM":          "Shelly Pro 2 PM",
	"Pro3":            "Shelly Pro 3",
	"Pro3EM":          "Shelly Pro 3 EM",
	"Pro4PM":          "Shelly Pro 4 PM",
	"RGBWPMminiG3":   "Shelly RGBW PM Mini Gen3",
	"i4G3":            "Shelly i4 Gen3",
	"i4G4":            "Shelly i4 Gen4",
	"SH2LED-1":        "Shelly 2LED",
	"SHAIR-1":         "Shelly Air",
	"SHAIR-2":         "Shelly Air Turbo",
	"SHBDUO-1":        "Shelly Duo",
	"SHBLB-1":         "Shelly Bulb",
	"SHBTN-1":         "Shelly Button",
	"SHBTN-2":         "Shelly Button",
	"SHCB-1":          "Shelly Color Bulb",
	"SHCL-255":        "Shelly Bulb",
	"SHDIMW-1":        "Shelly Dimmer",
	"SHDM-1":          "Shelly Dimmer",
	"SHDM-2":          "Shelly Dimmer 2",
	"SHDW-1":          "Shelly Door",
	"SHDW-2":          "Shelly Door 2",
	"SHEM-1":          "Shelly EM",
	"SHEM-3":          "Shelly EM3",
	"SHEM":            "Shelly EM",
	"SHGS-1":          "Shelly Gas",
	"SHHT-1":          "Shelly T&H",
	"SHIX3-1":         "Shelly i3",
	"SHMOS-01":        "Shelly Motion Sensor",
	"SHMOS-02":        "Shelly Motion 2",
	"SHPLG-1":         "Shelly Plug",
	"SHPLG-AU1":       "Shelly Plug AU",
	"SHPLG-IT1":       "Shelly Plug IT",
	"SHPLG-S":         "Shelly Plug S",
	"SHPLG-U1":        "Shelly Plug US",
	"SHPLG-UK1":       "Shelly Plug UK",
	"SHPLG2-1":        "Shelly Plug",
	"SHRGBW2":         "Shelly RGBW 2",
	"SHRGBWW-01":      "Shelly RGBWW",
	"SHSEN-1":         "Shelly Sense",
	"SHSK-1":          "Shelly Socket",
	"SHSM-01":         "Shelly Smoke",
	"SHSPOT-1":        "Shelly Spot",
	"SHSPOT-2":        "Shelly Spot 2",
	"SHSW-1":          "Shelly 1",
	"SHSW-1S":         "Shelly Harvia RSS",
	"SHSW-21":         "Shelly 2",
	"SHSW-22":         "Shelly HDPro",
	"SHSW-25":         "Shelly 25",
	"SHSW-44":         "Shelly 4Pro",
	"SHSW-L":          "Shelly 1L",
	"SHSW-PM":         "Shelly 1 PM",
	"SHTRV-01":        "Shelly TRV",
	"SHUNI-1":         "Shelly Uni",
	"SHVIN-1":         "Shelly Vintage",
	"SHWT-1":          "Shelly Flood",
	"SNDM-9995WW":     "Shelly Plus Dimmer",
}

// Device holds information about the device location, authentication
// requirements and firmware versions.
type Device struct {
	FirmwareNewestVersion RemoteFirmware
	App                   string
	FirmwareVersion       string
	FirmwareFilename      string
	Generation            int
	ID                    string
	IP                    net.IP
	Model                 string
	Name                  string
	Password              string
	Port                  int
	Username              string
}

func (d *Device) BaseURL() string {
	return fmt.Sprintf("http://%v:%v@%v:%v", d.Username, d.Password, d.IP.String(), d.Port)
}

func (d *Device) OTAURL(otaServerHost string, otaServerPort int, otaFilename string) string {
	return fmt.Sprintf("%s/ota?url=http://%s:%d/%s", d.BaseURL(), otaServerHost, otaServerPort, otaFilename)
}

// FamilyFriendlyName returns a human-friendly version of the device's model,
// if available.
func (d *Device) FamilyFriendlyName() string {
	if d.Model != "" && shellies[d.Model] != "" {
		return shellies[d.Model]
	}

	return d.Model
}

func (d *Device) String() string {
	if d.Name == "" {
		return fmt.Sprintf("%v (%v@%v:%v)", d.FamilyFriendlyName(), d.ID, d.IP.String(), d.Port)
	}

	return fmt.Sprintf("%v (%v@%v:%v)", d.Name, d.ID, d.IP.String(), d.Port)
}
