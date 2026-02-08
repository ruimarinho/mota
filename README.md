<p align="center">
    <img src="images/logo/cover.png" height="180">
  <p align="center">🛵 <strong>M</strong>ass <strong>O</strong>ver-<strong>T</strong>he-<strong>A</strong>ir updater for Shelly devices</p>
</p>

![build status](https://github.com/ruimarinho/mota/workflows/Tests/badge.svg?branch=master)

🛵 `mota` is a mass [Shelly](https://shelly.cloud) device firmware updater for local networks using the built-in Over-The-Air (OTA) update interface. It supports Gen1, Gen2 (Plus/Pro), Gen3 and Gen4 devices.

Discovery uses mDNS/zeroconf combined with HTTP subnet scanning, making it reliable even on macOS where `mDNSResponder` can prevent Go-based mDNS libraries from seeing all devices. It is particularly suited for network setups using VLANs where IoT devices do not have internet connectivity.

## Features

- Supports all Shelly generations (Gen1, Gen2 Plus/Pro, Gen3, Gen4) including Zigbee variants
- mDNS discovery supplemented by HTTP subnet scanning for reliable detection
- Scan additional subnets with `--subnet` for multi-VLAN setups
- Stepping-stone upgrades for Gen2+ devices that require firmware 1.3.3 before upgrading to 1.4.0+
- Filter by model or exclude devices by glob pattern
- JSON output for scripting (`--json`)
- `.netrc` authentication support
- Interactive or forced bulk upgrades

## Background

Shelly devices periodically ping the Shelly Cloud to check for firmware updates, but due to the vulnerable nature of their chipset (typically ESP8266 or ESP32), a multitude of security vulnerabilities exist [<sup>1</sup>](#reference-1) [<sup>2</sup>](#reference-2). MongooseOS, the IoT framework that powers Shelly devices, is also not free of vulnerabilities [<sup>3</sup>](#reference-3), although at this time they are not as severe as the chipset ones.

Although Allterco Robotics, the makers of Shelly devices, frequently releases updates to their devices (unlike many other vendors), it is still considered best practice to keep your IoT devices away from the internet.

If you're planning on isolating your IoT network from the internet, then `mota` brings you managed updates at the local network level either interactively or in bulk.

## Installation

Download a [binary release](https://github.com/ruimarinho/mota/releases) or, alternatively, install via go:

```sh
go install github.com/ruimarinho/mota@latest
```

You can also use Docker (Linux only, as Host mode networking is not available on Windows or macOS):

```
docker run --rm --net=host ruimarinho/mota
```

### macOS

Using Homebrew:

```
brew tap ruimarinho/tap
brew install mota
```

## Usage

### Discover and upgrade

```sh
mota
```

This scans for local Shelly devices and prompts you for interactive updates if new firmware versions are available.

### List available updates

```sh
mota list
```

### Force upgrades without confirmation

```sh
mota upgrade --force
```

> [!TIP]
> Over the years, Shelly has enhanced its OTA firmware updating process, making it significantly more dependable. Nonetheless, certain devices might not successfully update if they are overloaded with custom scripts or if there is insufficient free memory. For such cases, it is recommended to reboot these devices to facilitate a successful update.

### CLI

```
mota [command] [flags]

Commands:
  upgrade     Discover devices and upgrade firmware (default)
  list        Discover devices and list available updates
  version     Show version information

Flags:
      --beta              Include beta firmwares in the list of available updates
      --device strings    Use device IP address(es) instead of discovery
      --domain string     Set the search domain for the local network (default "local")
      --exclude strings   Exclude devices matching glob pattern(s)
  -f, --force             Force upgrades without asking for confirmation
  -p, --http-port int     HTTP port to listen for OTA requests (default: random)
      --json              Output results as JSON
      --model strings     Only include devices matching model name(s)
      --subnet strings    Additional subnet(s) to scan in CIDR notation
      --verbose           Enable verbose mode
  -w, --wait int          Duration in [s] to run discovery (default 60)
```

### Scanning additional subnets

When your Shelly devices are on different VLANs from the machine running `mota`, use `--subnet` to scan those networks:

```sh
mota list --subnet 192.168.100.0/24,192.168.10.0/24
```

### Updating specific devices

If you'd like to skip discovery, you may specify one or more devices to check individually:

```sh
mota --device 192.168.100.10 --device 192.168.100.30
```

### Filtering by model

Only upgrade devices of a specific model:

```sh
mota upgrade --model Plus1,Plus2PM
```

### Excluding devices

Exclude devices by glob pattern:

```sh
mota upgrade --exclude "Kitchen*" --exclude "shellyplug*"
```

### Authentication

If you have setup web access authentication (you should!), `mota` can automatically read and parse the standard `~/.netrc` (macOS/Linux) and `%HOME%/_netrc` (Windows) files. Create this file on your home folder and add your Shelly information in the following format:

```
machine <shelly_IP_1>
login <username_1>
password <password_1>

machine <shelly_IP_2>
login <username_2>
password <password_2>
```

### Beta firmwares

You may enable support for beta firmwares (if available):

```sh
mota --beta
```

### JSON output

Output device status as JSON for scripting:

```sh
mota list --json
```

### Stepping-stone upgrades

Gen2+ devices running firmware below 1.3.3 cannot jump directly to 1.4.0+. `mota` automatically detects this and upgrades to the mandatory 1.3.3 intermediate version first, then continues to the latest firmware on a second pass.

## License

MIT

## References

<a class="anchor" id="reference-1" href="https://github.com/Matheus-Garbelini/esp32_esp8266_attacks"><sup>1</sup> Proof of Concept of ESP32/8266 Wi-Fi vulnerabilties (CVE-2019-12586, CVE-2019-12587, CVE-2019-12588)</a>

<a class="anchor" id="reference-2" href="https://limitedresults.com/2019/11/pwn-the-esp32-forever-flash-encryption-and-sec-boot-keys-extraction/"><sup>2</sup> Pwn the ESP32 Forever: Flash Encryption and Sec. Boot Keys Extraction</a>

<a class="anchor" id="reference-32" href="https://www.cvedetails.com/vulnerability-list/vendor_id-16334/product_id-37010/Cesanta-Mongoose-Os.html"><sup>3</sup> Cesanta Mongoose OS Security Vulnerabilities</a>
