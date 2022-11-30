package main

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/davecgh/go-spew/spew"
)

// Firmware is a structure that holds information about a specific
// remote firmware file.
type Firmware struct {
	Model       string
	URL         string
	Version     string
	BetaURL     string `json:"beta_url"`
	BetaVersion string `json:"beta_ver"`
}

// APIClient is a struct that represents an API client that fetches
// information from the Shelly Cloud APIs.
type APIClient struct {
	baseURL      string
	includeBetas bool
	firmwares    map[string]Firmware
	httpClient   *http.Client
}

type response struct {
	IsOk bool                `json:"isok"`
	Data map[string]Firmware `json:"data"`
}

type gen2response struct {
	Stable struct {
		Version string `json:"version"`
		BuildID string `json:"build_id"`
		URL     string `json:"url"`
	} `json:"stable"`
	Beta struct {
		Version string `json:"version"`
		BuildID string `json:"build_id"`
		URL     string `json:"url"`
	} `json:"beta"`
}

// APIClientOption is an option interface for APIClient.
type APIClientOption func(*APIClient)

// WithAPIHTTPClient is an APIClient option that allows overriding the
// HTTP client used to make requests.
func WithAPIHTTPClient(httpClient *http.Client) APIClientOption {
	return func(client *APIClient) {
		client.httpClient = httpClient
	}
}

// WithBaseURL is an APIClient option that allows overriding the
// base URL used for remote calls.
func WithBaseURL(baseURL string) APIClientOption {
	return func(client *APIClient) {
		client.baseURL = baseURL
	}
}

// WithBetaFirmware is an APIClient option that enables beta firmware
// support when available
func WithBetaFirmware(includeBetas bool) APIClientOption {
	return func(client *APIClient) {
		client.includeBetas = includeBetas
	}
}

// NewAPIClient returns a new instance of the APIClient with default
// options.
func NewAPIClient(options ...APIClientOption) *APIClient {
	client := &APIClient{
		baseURL: "https://api.shelly.cloud",
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: 10 * time.Second,
		}}

	for _, option := range options {
		option(client)
	}

	return client
}

// FetchVersions returns a list of remotely available firmwares.
func (client *APIClient) FetchVersions() (map[string]Firmware, error) {
	if len(client.firmwares) > 0 {
		return client.firmwares, nil
	}

	// Gen1
	apiResponse, err := client.httpClient.Get(client.baseURL + "/files/firmware")
	if err != nil {
		return nil, err
	}

	var decoded response
	err = json.NewDecoder(apiResponse.Body).Decode(&decoded)
	if err != nil {
		return nil, err
	}

	client.firmwares = decoded.Data

	spew.Dump(client.firmwares)

	// Gen2
	gen2Devices := []string{"Plus1", "Plus1PM", "Plus2PM", "PlusI4", "Pro1", "Pro1PM", "Pro2", "Pro2PM", "Pro3", "Pro4PM", "PlugUS", "PlusHT", "PlusWallDimmer"}
	for _, gen2Device := range gen2Devices {
		apiResponse, err := client.httpClient.Get("https://updates.shelly.cloud/update/" + gen2Device)
		if err != nil {
			return nil, err
		}

		var decoded gen2response
		err = json.NewDecoder(apiResponse.Body).Decode(&decoded)
		if err != nil {
			return nil, err
		}

		client.firmwares[gen2Device] = Firmware{
			Model:       gen2Device,
			URL:         decoded.Stable.URL,
			Version:     decoded.Stable.Version,
			BetaURL:     decoded.Beta.URL,
			BetaVersion: decoded.Beta.Version,
		}

		spew.Dump(decoded)
	}

	return client.firmwares, nil
}

// FetchFirmware returns the binary data of a remote firmware for
// a specific model.
func (client *APIClient) FetchFirmware(model string) (io.ReadCloser, error) {
	url, err := client.GetURL(model)
	if err != nil {
		return nil, err
	}

	response, err := client.httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	return response.Body, nil
}

// GetVersion returns the most recent firmware version available for a model
func (client *APIClient) GetVersion(model string) (string, error) {
	firmwares, err := client.FetchVersions()
	if err != nil {
		return "", err
	}

	version := firmwares[model].Version
	if client.includeBetas && firmwares[model].BetaVersion != "" {
		version = firmwares[model].BetaVersion
	}

	return version, nil
}

// GetURL returns the most recent firmware download URL available for a model
func (client *APIClient) GetURL(model string) (string, error) {
	firmwares, err := client.FetchVersions()
	if err != nil {
		return "", err
	}

	version := firmwares[model].URL

	if client.includeBetas && firmwares[model].BetaURL != "" {
		version = firmwares[model].BetaURL
	}

	return version, nil
}
