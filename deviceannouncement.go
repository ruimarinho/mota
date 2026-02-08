package main

import (
	"fmt"
	"net"
)

const Gen1AnnouncementHeader = "id=shelly"
const Gen2AnnouncementHeader = "gen=2"
const Gen3AnnouncementHeader = "gen=3"
const Gen4AnnouncementHeader = "gen=4"

// DeviceAnnouncement holds information about the device discovery.
type DeviceAnnouncement struct {
	IP         net.IP
	HostName   string
	Port       int
	Generation int
}

func (da *DeviceAnnouncement) String() string {
	return fmt.Sprintf("%v (%v:%v)", da.HostName, da.IP.String(), da.Port)
}

func (da *DeviceAnnouncement) DeviceInformationURL(username string, password string) string {
	if da.Generation == 1 {
		return fmt.Sprintf("%s%s", da.BaseURL(username, password), "/settings")
	}

	return fmt.Sprintf("%s%s", da.BaseURL(username, password), "/rpc/Shelly.GetDeviceInfo")
}

// BaseURL returns the full URL required for API authentication,
// if needed.
func (da *DeviceAnnouncement) BaseURL(username string, password string) string {
	return fmt.Sprintf("http://%v:%v@%v:%v", username, password, da.IP.String(), da.Port)
}
