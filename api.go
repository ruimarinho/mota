package main

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
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
	baseURL    string
	betas      bool
	firmwares  map[string]Firmware
	httpClient *http.Client
}

type response struct {
	IsOk bool                `json:"isok"`
	Data map[string]Firmware `json:"data"`
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

// WithBetaFirmware is an APIClient option that enables beta version
// support
func WithBetaFirmware(betas bool) APIClientOption {
	return func(client *APIClient) {
		client.betas = betas
	}
}

// NewAPIClient returns a new instance of the APIClient with default
// options.
func NewAPIClient(options ...APIClientOption) *APIClient {
	client := &APIClient{
		baseURL: "https://api.shelly.cloud",
		httpClient: &http.Client{
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

	apiResponse, err := client.httpClient.Get(client.baseURL + "/files/firmware")
	if err != nil {
		return nil, err
	}

	var decoded response
	err = json.NewDecoder(apiResponse.Body).Decode(&decoded)
	if err != nil {
		return nil, err
	}

	return decoded.Data, nil
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

// GetVersion
func (client *APIClient) GetVersion(model string) (string, error) {
	firmwares, err := client.FetchVersions()
	if err != nil {
		return "", err
	}

	version := firmwares[model].Version

	if client.betas && firmwares[model].BetaVersion != "" {
		version = firmwares[model].BetaVersion
	}

	return version, nil
}

// GetURL
func (client *APIClient) GetURL(model string) (string, error) {
	firmwares, err := client.FetchVersions()
	if err != nil {
		return "", err
	}

	version := firmwares[model].URL

	if client.betas && firmwares[model].BetaURL != "" {
		version = firmwares[model].BetaURL
	}

	return version, nil
}
