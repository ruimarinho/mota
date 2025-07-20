<p align="center">
    <img src="images/logo/cover.png" height="180">
  <p align="center">🛵 <strong>M</strong>ass <strong>O</strong>ver-<strong>T</strong>he-<strong>A</strong>ir updater for Shelly devices</p>
</p>

![build status](https://github.com/ruimarinho/mota/workflows/Tests/badge.svg?branch=master)

🛵 `mota`  is a mass [Shelly](https://shelly.cloud) device firmware updater based on zeroconf (or bonjour) discovery for local networks using the built-in Over-The-Air (OTA) update interface. It is particularly suited for network setups using VLANs where IoT devices do not have internet connectivity.


## Background

Shelly devices periodically ping the Shelly Cloud to check for firmware updates, but due to the vulnerable nature of their chipset (typically ESP8266 or ESP32), a multitude of security vulnerabilities exist [<sup>1</sup>](#reference-1) [<sup>2</sup>](#reference-2). MongooseOS, the IoT framework that powers Shelly devices, is also not free of vulnerabilities [<sup>3</sup>](#reference-3), although at this time they are not as severe as the chipset ones.

Although Allterco Robotics, the makers of Shelly devices, frequently releases updates to their devices (unlike many other vendors), it is still considered best practice to keep your IoT devices away from the internet.

If you're planning on isolating your IoT network from the internet, then `mota` brings you managed updates at the local network level either interactively or in bulk.

## Installation

Download a [binary release](https://github.com/ruimarinho/mota/releases) or, alternatively, install via go:

```sh
❯ go get -u github.com/ruimarinho/mota
❯ go install github.com/ruimarinho/mota
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

```sh
❯ mota
```

If local devices are found and new firmware versions are available for your devices, you will be prompted to interactively choose which devices to update.

Sometimes Shellies appear to ignore OTA requests and may require multiple attempts to finally update to the requested version. At this time, it is my belief this is an issue with the OTA routines on the OS that powers Shellies.

### CLI

```sh
❯ mota -help

Usage of mota:
      --beta            Use beta firmwares if available
      --domain string   Set the search domain for the local network. (default "local")
  -f, --force           Force upgrades without asking for confirmation
      --host strings    Use host/IP address(es) instead of device discovery (can be specified multiple times or be comma-separated)
  -p, --http-port int   HTTP port to listen for OTA requests. If not specified, a random port is chosen.
      --verbose         Enable verbose mode.
  -v, --version         Show version information
  -w, --wait int        Duration in [s] to run discovery. (default 60)
```

### Authentication

You can either provide global credentials or device based credentials. Device based credentials always take precedence.

#### Global credentials

Create a file in your home folder called `mota.yml` and put this into our file

```yaml
global:
  credentials:
    username: MyName
    password: verysecret
```

#### Device based credentials

If you have setup web access authentication (you should!), `mota` can automatically read and parse the standard `~/.netrc` (macOS/Linux) and `%HOME%/_netrc` (Windows) files. Create this file on your home folder and add your Shelly information in the following format:

```
machine <shelly_IP_1>
login <username_1>
password <password_1>

machine <shelly_IP_2>
login <username_2>
password <password_2>
```

### Updating Specific Hosts

If you'd like to skip bonjour discovery, you may specify one or more devices to check individually:

```sh
mota --host=192.168.100.10 --host=192.168.100.30
```

### Beta Firmwares

You may enable support for beta firmwares (if available):

```sh
mota --beta
```

## License

MIT

## References

<a class="anchor" id="reference-1" href="https://github.com/Matheus-Garbelini/esp32_esp8266_attacks"><sup>1</sup> Proof of Concept of ESP32/8266 Wi-Fi vulnerabilties (CVE-2019-12586, CVE-2019-12587, CVE-2019-12588)</a>

<a class="anchor" id="reference-2" href="https://limitedresults.com/2019/11/pwn-the-esp32-forever-flash-encryption-and-sec-boot-keys-extraction/"><sup>2</sup> Pwn the ESP32 Forever: Flash Encryption and Sec. Boot Keys Extraction</a>

<a class="anchor" id="reference-32" href="https://www.cvedetails.com/vulnerability-list/vendor_id-16334/product_id-37010/Cesanta-Mongoose-Os.html"><sup>3</sup> Cesanta Mongoose OS Security Vulnerabilities</a>
