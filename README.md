# shelly-updater

![build status](https://github.com/ruimarinho/shelly-updater/workflows/Tests/badge.svg?branch=master)

`shelly-updater` is a [Shelly](https://shelly.cloud) device firmware updater based on zeroconf (or bonjour) discovery for local networks using the built-in Over-The-Air (OTA) update interface. It is suited for network setups where IoT devices do not have internet connectivity.

## Background

Shelly devices periodically ping the Shelly Cloud to check for firmware updates, but due to the vulnerable nature of their chipset (typically ESP8266 or ESP32), a multitude of security vulnerabilities exist [<sup>1</sup>](#reference-1) [<sup>2</sup>](#reference-2). Mongoose OS, the IoT framework that powers Shelly devices, is also not free of vulnerabilities [<sup>3</sup>](#reference-3), although at this time they are not as severe as the chipset ones.

Although Alterco Robotics, the makers of Shelly devices, frequently releases updates to their devices (unlike many other vendors), it is still considered best practice to keep your IoT devices away from the internet.

If you're planning on isolating your IoT network from the internet, then `shelly-updater` brings you managed updates at the local network level.

## Installation

Download a [binary release](https://github.com/ruimarinho/shelly-updater/releases) or, alternatively, install via go:

```sh
❯ go get -u github.com/ruimarinho/shelly-updater
❯ go install github.com/ruimarinho/shelly-updater
```

You can also use Docker (Linux only, as Host mode networking is not available on Windows or macOS):

```
docker run --rm --net=host ruimarinho/shelly-updater
```

### macOS

Using Homebrew:

```
brew tap ruimarinho/tap
brew install shelly-updater
```

## Usage

```sh
❯ shelly-updater
```

If local devices are found and new firmware versions are available for your devices, you will be prompted to interactively choose which devices to update.

### CLI

```sh
❯ shelly-updater -help

Usage of /shelly-updater:
  -domain string
    	Set the search domain for the local network. (default "local")
  -http-port int
    	HTTP port to listen for OTA requests. If not specified, a random port is chosen.
  -verbose
    	Enable verbose mode.
  -version
    	Show version information
  -wait int
      Duration in [s] to run discovery. (default 60)
```

## License

MIT

## References

<a class="anchor" id="reference-1" href="https://github.com/Matheus-Garbelini/esp32_esp8266_attacks"><sup>1</sup> Proof of Concept of ESP32/8266 Wi-Fi vulnerabilties (CVE-2019-12586, CVE-2019-12587, CVE-2019-12588)</a>

<a class="anchor" id="reference-2" href="https://limitedresults.com/2019/11/pwn-the-esp32-forever-flash-encryption-and-sec-boot-keys-extraction/"><sup>2</sup> Pwn the ESP32 Forever: Flash Encryption and Sec. Boot Keys Extraction</a>

<a class="anchor" id="reference-32" href="https://www.cvedetails.com/vulnerability-list/vendor_id-16334/product_id-37010/Cesanta-Mongoose-Os.html"><sup>3</sup> Cesanta Mongoose OS Security Vulnerabilities</a>
