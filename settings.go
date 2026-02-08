package main

// Gen1Settings is the structure holding information about the device
// model type and current firmware version.
type Gen1Settings struct {
	Device struct {
		Hostname string `json:"hostname"`
		Model    string `json:"type"`
	} `json:"device"`
	Firmware string `json:"fw"`
	Name     string `json:"name"`
}

// Settings is the structure holding information about the device
// model type and current firmware version.
type Gen2Settings struct {
	ID       string `json:"id"`
	Model    string `json:"app"`
	Firmware string `json:"ver"`
	Name     string `json:"name"`
}
