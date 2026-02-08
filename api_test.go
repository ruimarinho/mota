package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchVersionsGen1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(fmt.Sprintf(`{
				"isok": true,
				"data": {
					"SHSW-25": {
						"url": "http://%s/firmware/SHSW-25_build.zip",
						"version": "20200309-104051/v1.6.0@43056d58"
					}
				}
			}`, req.Host)))
			return
		}
	}))
	defer server.Close()

	// Use a gen2 server that returns empty for all models
	gen2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(`{"stable":{"version":"1.0.0","build_id":"build1","url":"http://example.com/fw.zip"},"beta":{"version":"1.1.0-beta","build_id":"build2","url":"http://example.com/fw_beta.zip"}}`))
	}))
	defer gen2Server.Close()

	client := NewAPIClient(WithBaseURL(server.URL), WithGen2BaseURL(gen2Server.URL))
	firmwares, err := client.FetchVersions()
	assert.Nil(t, err)
	assert.NotNil(t, firmwares)

	fw, ok := firmwares["SHSW-25"]
	assert.True(t, ok)
	assert.Equal(t, "SHSW-25", fw.Model)
	assert.Equal(t, "20200309-104051/v1.6.0@43056d58", fw.Version)
}

func TestFetchVersionsGen2(t *testing.T) {
	gen1Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(`{"isok": true, "data": {}}`))
			return
		}
	}))
	defer gen1Server.Close()

	gen2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(fmt.Sprintf(`{
			"stable": {"version": "1.0.0", "build_id": "build1", "url": "http://%s/fw.zip"},
			"beta": {"version": "1.1.0-beta", "build_id": "build2", "url": "http://%s/fw_beta.zip"}
		}`, req.Host, req.Host)))
	}))
	defer gen2Server.Close()

	client := NewAPIClient(WithBaseURL(gen1Server.URL), WithGen2BaseURL(gen2Server.URL))
	firmwares, err := client.FetchVersions()
	assert.Nil(t, err)
	assert.NotNil(t, firmwares)

	// Check one of the Gen2+ models
	fw, ok := firmwares["Plus1"]
	assert.True(t, ok)
	assert.Equal(t, "Plus1", fw.Model)
	assert.Equal(t, "1.0.0", fw.Version)
	assert.Equal(t, "1.1.0-beta", fw.BetaVersion)
}

func TestFetchVersionsCaching(t *testing.T) {
	var callCount int32

	gen1Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		atomic.AddInt32(&callCount, 1)
		if req.URL.Path == "/files/firmware" {
			w.Write([]byte(`{"isok": true, "data": {}}`))
			return
		}
	}))
	defer gen1Server.Close()

	gen2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(`{"stable":{"version":"1.0.0","build_id":"b","url":"http://x/fw.zip"},"beta":{"version":"","build_id":"","url":""}}`))
	}))
	defer gen2Server.Close()

	client := NewAPIClient(WithBaseURL(gen1Server.URL), WithGen2BaseURL(gen2Server.URL))

	_, err := client.FetchVersions()
	assert.Nil(t, err)

	firstCallCount := atomic.LoadInt32(&callCount)

	_, err = client.FetchVersions()
	assert.Nil(t, err)

	// Server should not be called again due to caching
	assert.Equal(t, firstCallCount, atomic.LoadInt32(&callCount))
}

func TestDownloadFirmware(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("firmware-binary-data"))
	}))
	defer server.Close()

	client := NewAPIClient()
	tmpDir := t.TempDir()

	rf := RemoteFirmware{
		Model:   "SHSW-25",
		URL:     server.URL + "/firmware/SHSW-25_build.zip",
		Version: "1.0.0",
	}

	path, err := client.DownloadFirmware(rf, false, tmpDir)
	assert.Nil(t, err)
	assert.FileExists(t, path)

	content, err := os.ReadFile(path)
	assert.Nil(t, err)
	assert.Equal(t, "firmware-binary-data", string(content))
}

func TestDownloadFirmwareCaching(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Write([]byte("firmware-binary-data"))
	}))
	defer server.Close()

	client := NewAPIClient()
	tmpDir := t.TempDir()

	rf := RemoteFirmware{
		Model:   "SHSW-25",
		URL:     server.URL + "/firmware/SHSW-25_build.zip",
		Version: "1.0.0",
	}

	path1, err := client.DownloadFirmware(rf, false, tmpDir)
	assert.Nil(t, err)

	path2, err := client.DownloadFirmware(rf, false, tmpDir)
	assert.Nil(t, err)

	assert.Equal(t, path1, path2)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
}

func TestDownloadFirmwareHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewAPIClient()
	tmpDir := t.TempDir()

	rf := RemoteFirmware{
		Model:   "SHSW-25",
		URL:     server.URL + "/firmware/SHSW-25_build.zip",
		Version: "1.0.0",
	}

	_, err := client.DownloadFirmware(rf, false, tmpDir)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
}

func TestGetLatestFirmwareAvailable(t *testing.T) {
	t.Run("known model", func(t *testing.T) {
		gen1Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/files/firmware" {
				w.Write([]byte(fmt.Sprintf(`{
					"isok": true,
					"data": {
						"SHSW-25": {
							"url": "http://%s/firmware/SHSW-25_build.zip",
							"version": "1.6.0"
						}
					}
				}`, req.Host)))
				return
			}
		}))
		defer gen1Server.Close()

		gen2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Write([]byte(`{"stable":{"version":"1.0.0","build_id":"b","url":"http://x/fw.zip"},"beta":{"version":"","build_id":"","url":""}}`))
		}))
		defer gen2Server.Close()

		client := NewAPIClient(WithBaseURL(gen1Server.URL), WithGen2BaseURL(gen2Server.URL))

		fw, err := client.GetLatestFirmwareAvailable("SHSW-25")
		assert.Nil(t, err)
		assert.Equal(t, "1.6.0", fw.Version)
	})

	t.Run("unknown model", func(t *testing.T) {
		gen1Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/files/firmware" {
				w.Write([]byte(`{"isok": true, "data": {}}`))
				return
			}
		}))
		defer gen1Server.Close()

		gen2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Write([]byte(`{"stable":{"version":"1.0.0","build_id":"b","url":"http://x/fw.zip"},"beta":{"version":"","build_id":"","url":""}}`))
		}))
		defer gen2Server.Close()

		client := NewAPIClient(WithBaseURL(gen1Server.URL), WithGen2BaseURL(gen2Server.URL))

		_, err := client.GetLatestFirmwareAvailable("NONEXISTENT")
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
