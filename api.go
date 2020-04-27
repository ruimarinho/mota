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
	Model   string
	URL     string
	Version string
}

// APIClient is a struct that represents an API client that fetches
// information from the Shelly Cloud APIs.
type APIClient struct {
	baseURL    string
	firmwares  map[string]Firmware
	httpClient *http.Client
}

type response struct {
	IsOk bool                `json:"isok"`
	Data map[string]Firmware `json:"data"`
}

// WithAPIHTTPClient is an APIClient option that allows overriding the
// HTTP client used to make requests.
func WithAPIHTTPClient(httpClient *http.Client) func(*APIClient) {
	return func(client *APIClient) {
		client.httpClient = httpClient
	}
}

// WithBaseURL is an APIClient option that allows overriding the
// base URL used for remote calls.
func WithBaseURL(baseURL string) func(*APIClient) {
	return func(client *APIClient) {
		client.baseURL = baseURL
	}
}

// NewAPIClient returns a new instance of the APIClient with default
// options.
func NewAPIClient(options ...func(*APIClient)) *APIClient {
	client := &APIClient{
		baseURL: "https://api.shelly.cloud",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		}}

	for _, o := range options {
		o(client)
	}

	return client
}

// FetchFirmwares returns a list of remotely available firmwares.
func (client *APIClient) FetchFirmwares() (map[string]Firmware, error) {
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

// GetFirmware returns the binary data of a remote firmware for
// a specific model.
func (client *APIClient) GetFirmware(model string) (io.ReadCloser, error) {
	firmwares, err := client.FetchFirmwares()
	if err != nil {
		return nil, err
	}

	response, err := client.httpClient.Get(firmwares[model].URL)
	if err != nil {
		return nil, err
	}

	return response.Body, nil
}
