package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// DockerComposeHelper helps manage Docker Compose services for E2E tests
type DockerComposeHelper struct {
	composeFile string
	projectDir string
	t          *testing.T
}

// NewDockerComposeHelper creates a new Docker Compose helper
func NewDockerComposeHelper(t *testing.T) *DockerComposeHelper {
	// Find project root by looking for docker-compose.yaml
	// We're in tests/api_e2e/, so go up to project root
	pathsToTry := []string{
		"../../config/docker-compose.yaml", // From tests/api_e2e/ to project root
		"../config/docker-compose.yaml",     // From tests/ to project root
		"config/docker-compose.yaml",        // Current directory
	}

	var projectRoot string
	for _, path := range pathsToTry {
		if _, err := os.Stat(path); err == nil {
			absPath, _ := filepath.Abs(path)
			projectRoot = filepath.Dir(absPath)
			break
		}
	}

	// Default to going up two levels if not found
	if projectRoot == "" {
		absPath, _ := filepath.Abs("../..")
		projectRoot = absPath
	}

	return &DockerComposeHelper{
		composeFile: "config/docker-compose.yaml",
		projectDir:  projectRoot,
		t:           t,
	}
}

// StartServices starts all services using docker-compose
func (h *DockerComposeHelper) StartServices() error {
	h.t.Log("Starting services with docker-compose...")
	cmd := exec.Command("docker-compose", "-f", h.composeFile, "up", "-d", "--build")
	cmd.Dir = h.projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		h.t.Logf("docker-compose output: %s", string(output))
		return fmt.Errorf("failed to start services: %w", err)
	}
	h.t.Log("Services started, waiting for them to be ready...")
	return h.waitForServices()
}

// StopServices stops all services
func (h *DockerComposeHelper) StopServices() error {
	h.t.Log("Stopping services...")
	cmd := exec.Command("docker-compose", "-f", h.composeFile, "down")
	cmd.Dir = h.projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		h.t.Logf("docker-compose output: %s", string(output))
		return fmt.Errorf("failed to stop services: %w", err)
	}
	return nil
}

// RestartServices restarts all services
func (h *DockerComposeHelper) RestartServices() error {
	if err := h.StopServices(); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return h.StartServices()
}

// waitForServices waits for services to be healthy
func (h *DockerComposeHelper) waitForServices() error {
	client := &http.Client{Timeout: 5 * time.Second}
	services := []struct {
		name string
		url  string
	}{
		{"API", "http://localhost:8090/health"},
		{"WebSocket Gateway", "http://localhost:8088/health"},
		{"Ingest", "http://localhost:8081/health"},
		{"Bars", "http://localhost:8083/health"},
		{"Indicator", "http://localhost:8085/health"},
		{"Scanner", "http://localhost:8087/health"},
		{"Alert", "http://localhost:8093/health"},
	}

	maxAttempts := 60
	for attempt := 0; attempt < maxAttempts; attempt++ {
		allReady := true
		for _, svc := range services {
			resp, err := client.Get(svc.url)
			if err != nil || resp.StatusCode != http.StatusOK {
				allReady = false
				if resp != nil {
					resp.Body.Close()
				}
				break
			}
			resp.Body.Close()
		}

		if allReady {
			h.t.Log("All services are ready!")
			return nil
		}

		time.Sleep(2 * time.Second)
		if attempt%5 == 0 {
			h.t.Logf("Waiting for services... (attempt %d/%d)", attempt+1, maxAttempts)
		}
	}

	return fmt.Errorf("services did not become ready within timeout")
}

// CheckServiceHealth checks if a specific service is healthy
func (h *DockerComposeHelper) CheckServiceHealth(url string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// APIClient is a helper for making API calls
type APIClient struct {
	baseURL    string
	httpClient *http.Client
	t          *testing.T
}

// NewAPIClient creates a new API client
func NewAPIClient(t *testing.T, baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		t: t,
	}
}

// Request makes an HTTP request
func (c *APIClient) Request(method, path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path
	var reqBody io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	c.t.Logf("%s %s", method, url)
	return c.httpClient.Do(req)
}

// Get makes a GET request
func (c *APIClient) Get(path string) (*http.Response, error) {
	return c.Request("GET", path, nil)
}

// Post makes a POST request
func (c *APIClient) Post(path string, body interface{}) (*http.Response, error) {
	return c.Request("POST", path, body)
}

// Put makes a PUT request
func (c *APIClient) Put(path string, body interface{}) (*http.Response, error) {
	return c.Request("PUT", path, body)
}

// Delete makes a DELETE request
func (c *APIClient) Delete(path string) (*http.Response, error) {
	return c.Request("DELETE", path, nil)
}

// ParseJSON parses JSON response
func (c *APIClient) ParseJSON(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

// AssertStatus asserts the response status code
func (c *APIClient) AssertStatus(resp *http.Response, expected int) error {
	if resp.StatusCode != expected {
		body := new(bytes.Buffer)
		body.ReadFrom(resp.Body)
		return fmt.Errorf("expected status %d, got %d. Body: %s", expected, resp.StatusCode, body.String())
	}
	return nil
}

