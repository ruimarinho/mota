package main

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// steppingStone133 maps Gen2+ model names to their mandatory 1.3.3 firmware.
// Shelly Gen2+ devices running firmware below 1.3.3 cannot jump directly to
// 1.4.0+ — the Gen2 changelog states "1.3.3 is a mandatory update before 1.4.0."
// URLs follow the format https://fwcdn.shelly.cloud/gen2/{CDNModel}/{sha256}.
//
// CDN model names usually match the device app name, except for a few Gen3
// models where the name changed between firmware versions:
//   - i4G3 uses CDN name I4G3
//   - 1G3 uses CDN name S1G3
//   - 1PMG3 uses CDN name S1PMG3
//
// Models NOT listed here either shipped with firmware >= 1.3.3 or their
// firmware hash is unknown. The following Gen2 models are known to have
// shipped with firmware < 1.3.3 but their 1.3.3 CDN hashes have not been
// located: PlugUS, Plus10V, Plus1Mini, PlusHT, PlusPlugIT, PlusPlugUK,
// PlusPMMini, PlusWallDimmer, Pro1, Pro1PM, Pro2, Pro2PM, Pro3, Pro3EM,
// Pro4PM. Contributions with verified hashes are welcome.
//
// Gen4 devices shipped after firmware 1.4.0 and do not need entries.
// Newer Gen3 models (MiniPMG3, 1MiniG3, 1PMMiniG3, 2PMG3, 0-10VDimmerG3,
// RGBWPMminiG3, EMXG3, HTG3, FloodG3, PlugSG3, DimmerG3) also shipped
// with firmware >= 1.3.3 and do not need entries.
var steppingStone133 = map[string]RemoteFirmware{
	"Plus1": {
		Model:   "Plus1",
		Version: "1.3.3",
		URL:     "https://fwcdn.shelly.cloud/gen2/Plus1/ddd5a7b49ff3e65240d1264eb531f82da2aa86d3d05d045c5226a81e7ea2e43d",
	},
	"Plus1PM": {
		Model:   "Plus1PM",
		Version: "1.3.3",
		URL:     "https://fwcdn.shelly.cloud/gen2/Plus1PM/cc34adf8e45a3765b3f05efcd9a4322efd99c50c52ec9434fa51beb3b56217e1",
	},
	"Plus1PMMini": {
		Model:   "Plus1PMMini",
		Version: "1.3.3",
		URL:     "https://fwcdn.shelly.cloud/gen2/Plus1PMMini/72efef59bf19303ab32be3bc5e303e1fdf15cf6608698a73c5e3ffdbfa17e61e",
	},
	"Plus2PM": {
		Model:   "Plus2PM",
		Version: "1.3.3",
		URL:     "https://fwcdn.shelly.cloud/gen2/Plus2PM/eea874bcfee2b4876901948159b80bd9d2fc719300982f3ee489fa2168d400ea",
	},
	"PlusI4": {
		Model:   "PlusI4",
		Version: "1.3.3",
		URL:     "https://fwcdn.shelly.cloud/gen2/PlusI4/a341e6b3ab556ebfcc442311f65dc1e1c5fd01ec7e926617b8eb2589d0d00a8b",
	},
	"PlusPlugS": {
		Model:   "PlusPlugS",
		Version: "1.3.3",
		URL:     "https://fwcdn.shelly.cloud/gen2/PlusPlugS/b537c97799933584593641ea0f7ca7d3750b4020ce134d641953b92df5845220",
	},
	"Mini1G3": {
		Model:   "Mini1G3",
		Version: "1.3.3",
		URL:     "https://fwcdn.shelly.cloud/gen2/Mini1G3/ad6a38015d22503f95e4435d9a15342b7c721f30b4caf7e93f195428aa3b3ed0",
	},
	"Mini1PMG3": {
		Model:   "Mini1PMG3",
		Version: "1.3.3",
		URL:     "https://fwcdn.shelly.cloud/gen2/Mini1PMG3/ac3e0a3dcbf2809d0509b9b2335276fe76dcf51662df32a22677f64be58f4e54",
	},
	"i4G3": {
		Model:   "i4G3",
		Version: "1.3.3",
		URL:     "https://fwcdn.shelly.cloud/gen2/I4G3/cff09b114d5ff6980b1f4858cf80b9d37948371f64a4b4305ba3dc82507521d7",
	},
	"1G3": {
		Model:   "1G3",
		Version: "1.3.3",
		URL:     "https://fwcdn.shelly.cloud/gen2/S1G3/0021ac4946f8406df5f99e33d2fb2e37e4a5a5152f91dbbdcf5dd62d548b407d",
	},
	"1PMG3": {
		Model:   "1PMG3",
		Version: "1.3.3",
		URL:     "https://fwcdn.shelly.cloud/gen2/S1PMG3/0527974777080c85f3250c99f33ea3adff7da4ee02f03609b3fc03020ded9666",
	},
}

const steppingStoneVersion = "1.3.3"

// parseVersion parses a semver string "major.minor.patch" into its components.
func parseVersion(v string) (major, minor, patch int, err error) {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid version format: %q", v)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version in %q: %w", v, err)
	}

	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version in %q: %w", v, err)
	}

	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch version in %q: %w", v, err)
	}

	return major, minor, patch, nil
}

// isVersionLessThan returns true if version a is strictly less than version b.
func isVersionLessThan(a, b string) bool {
	aMajor, aMinor, aPatch, err := parseVersion(a)
	if err != nil {
		return false
	}

	bMajor, bMinor, bPatch, err := parseVersion(b)
	if err != nil {
		return false
	}

	if aMajor != bMajor {
		return aMajor < bMajor
	}
	if aMinor != bMinor {
		return aMinor < bMinor
	}
	return aPatch < bPatch
}

// NeedsSteppingStone checks if a Gen2+ device requires a stepping-stone
// upgrade to 1.3.3 before it can be upgraded to 1.4.0+. Returns the
// stepping-stone RemoteFirmware and true if needed, zero value and false otherwise.
func NeedsSteppingStone(device *Device) (RemoteFirmware, bool) {
	if device.Generation < 2 {
		return RemoteFirmware{}, false
	}

	if !isVersionLessThan(device.FirmwareVersion, steppingStoneVersion) {
		return RemoteFirmware{}, false
	}

	if fw, ok := steppingStone133[device.Model]; ok {
		return fw, true
	}

	log.Warnf("%v is running firmware %v (below %v) but no stepping-stone firmware is available for model %v. "+
		"Manual upgrade to %v may be required. Check https://shelly-api-docs.shelly.cloud for instructions.",
		device.String(), device.FirmwareVersion, steppingStoneVersion, device.Model, steppingStoneVersion)

	return RemoteFirmware{}, false
}
