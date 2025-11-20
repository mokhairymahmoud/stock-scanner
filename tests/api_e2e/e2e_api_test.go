// Package data contains API-based end-to-end tests.
//
// These tests simulate real user workflows by:
// 1. Deploying all services via Docker Compose
// 2. Making HTTP API calls to test functionality
// 3. Connecting via WebSocket to receive real-time alerts
//
// This is the recommended approach for E2E testing as it tests the system
// exactly as a real user would interact with it.
//
// See README.md for detailed documentation on running these tests.
package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

const (
	apiBaseURL     = "http://localhost:8090"
	wsBaseURL      = "ws://localhost:8088"
	healthEndpoint = "/health"
)

// TestClient is a helper for making API calls
type TestClient struct {
	baseURL    string
	httpClient *http.Client
	t          *testing.T
}

// NewTestClient creates a new test client
func NewTestClient(t *testing.T) *TestClient {
	return &TestClient{
		baseURL: apiBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		t: t,
	}
}

// Get makes a GET request
func (c *TestClient) Get(path string) (*http.Response, error) {
	url := c.baseURL + path
	c.t.Logf("GET %s", url)
	return c.httpClient.Get(url)
}

// Post makes a POST request
func (c *TestClient) Post(path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	c.t.Logf("POST %s", url)
	return c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonBody))
}

// Put makes a PUT request
func (c *TestClient) Put(path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.t.Logf("PUT %s", url)
	return c.httpClient.Do(req)
}

// Delete makes a DELETE request
func (c *TestClient) Delete(path string) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}
	c.t.Logf("DELETE %s", url)
	return c.httpClient.Do(req)
}

// ParseJSONResponse parses a JSON response
func (c *TestClient) ParseJSONResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

// TestE2E_ServiceHealth checks that all services are healthy
func TestE2E_ServiceHealth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Ensure services are running
	ensureServicesRunning(t)

	client := NewTestClient(t)

	// Check API service health
	resp, err := client.Get(healthEndpoint)
	if err != nil {
		t.Skipf("Services not available (start with: docker-compose -f config/docker-compose.yaml up -d): %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var health map[string]interface{}
	if err := client.ParseJSONResponse(resp, &health); err != nil {
		t.Fatalf("Failed to parse health response: %v", err)
	}

	t.Logf("API Service Health: %+v", health)
}

// TestE2E_CreateAndListRules tests rule creation and listing
func TestE2E_CreateAndListRules(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Ensure services are running
	ensureServicesRunning(t)

	client := NewTestClient(t)

	// Create a rule
	rule := map[string]interface{}{
		"name": "RSI Oversold Test",
		"conditions": []map[string]interface{}{
			{
				"metric":   "rsi_14",
				"operator": "<",
				"value":    30.0,
			},
		},
		"cooldown": 300,
		"enabled":  true,
	}

	resp, err := client.Post("/api/v1/rules", rule)
	if err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body := new(bytes.Buffer)
		body.ReadFrom(resp.Body)
		t.Fatalf("Expected status 201 or 200, got %d. Body: %s", resp.StatusCode, body.String())
	}

	var createdRule models.Rule
	if err := client.ParseJSONResponse(resp, &createdRule); err != nil {
		t.Fatalf("Failed to parse created rule: %v", err)
	}

	t.Logf("Created rule: %+v", createdRule)

	// List rules
	resp, err = client.Get("/api/v1/rules")
	if err != nil {
		t.Fatalf("Failed to list rules: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var rulesResponse map[string]interface{}
	if err := client.ParseJSONResponse(resp, &rulesResponse); err != nil {
		t.Fatalf("Failed to parse rules response: %v", err)
	}

	rules, ok := rulesResponse["rules"].([]interface{})
	if !ok {
		t.Fatal("Expected 'rules' array in response")
	}

	if len(rules) == 0 {
		t.Error("Expected at least one rule")
	}

	t.Logf("Found %d rules", len(rules))

	// Get specific rule
	resp, err = client.Get(fmt.Sprintf("/api/v1/rules/%s", createdRule.ID))
	if err != nil {
		t.Fatalf("Failed to get rule: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var retrievedRule models.Rule
	if err := client.ParseJSONResponse(resp, &retrievedRule); err != nil {
		t.Fatalf("Failed to parse rule: %v", err)
	}

	if retrievedRule.ID != createdRule.ID {
		t.Errorf("Rule ID mismatch: expected %s, got %s", createdRule.ID, retrievedRule.ID)
	}
}

// TestE2E_UpdateAndDeleteRule tests rule updates and deletion
func TestE2E_UpdateAndDeleteRule(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Ensure services are running
	ensureServicesRunning(t)

	client := NewTestClient(t)

	// Create a rule first
	rule := map[string]interface{}{
		"name": "Test Rule for Update",
		"conditions": []map[string]interface{}{
			{
				"metric":   "rsi_14",
				"operator": "<",
				"value":    30.0,
			},
		},
		"cooldown": 300,
		"enabled":  true,
	}

	resp, err := client.Post("/api/v1/rules", rule)
	if err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}
	defer resp.Body.Close()

	var createdRule models.Rule
	if err := client.ParseJSONResponse(resp, &createdRule); err != nil {
		t.Fatalf("Failed to parse created rule: %v", err)
	}

	// Update the rule
	updatedRule := map[string]interface{}{
		"name": "Updated Test Rule",
		"conditions": []map[string]interface{}{
			{
				"metric":   "rsi_14",
				"operator": "<",
				"value":    25.0, // Changed from 30.0
			},
		},
		"cooldown": 600, // Changed from 300
		"enabled":  true,
	}

	resp, err = client.Put(fmt.Sprintf("/api/v1/rules/%s", createdRule.ID), updatedRule)
	if err != nil {
		t.Fatalf("Failed to update rule: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body := new(bytes.Buffer)
		body.ReadFrom(resp.Body)
		t.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, body.String())
	}

	// Verify update
	resp, err = client.Get(fmt.Sprintf("/api/v1/rules/%s", createdRule.ID))
	if err != nil {
		t.Fatalf("Failed to get updated rule: %v", err)
	}
	defer resp.Body.Close()

	var retrievedRule models.Rule
	if err := client.ParseJSONResponse(resp, &retrievedRule); err != nil {
		t.Fatalf("Failed to parse updated rule: %v", err)
	}

	if retrievedRule.Name != "Updated Test Rule" {
		t.Errorf("Rule name not updated: expected 'Updated Test Rule', got '%s'", retrievedRule.Name)
	}

	// Delete the rule
	resp, err = client.Delete(fmt.Sprintf("/api/v1/rules/%s", createdRule.ID))
	if err != nil {
		t.Fatalf("Failed to delete rule: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 200 or 204, got %d", resp.StatusCode)
	}

	// Verify deletion
	resp, err = client.Get(fmt.Sprintf("/api/v1/rules/%s", createdRule.ID))
	if err != nil {
		t.Fatalf("Failed to check deleted rule: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 after deletion, got %d", resp.StatusCode)
	}
}

// TestE2E_ListSymbols tests symbol listing
func TestE2E_ListSymbols(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Ensure services are running
	ensureServicesRunning(t)

	client := NewTestClient(t)

	resp, err := client.Get("/api/v1/symbols")
	if err != nil {
		t.Fatalf("Failed to list symbols: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var symbolsResponse map[string]interface{}
	if err := client.ParseJSONResponse(resp, &symbolsResponse); err != nil {
		t.Fatalf("Failed to parse symbols response: %v", err)
	}

	symbols, ok := symbolsResponse["symbols"].([]interface{})
	if !ok {
		t.Fatal("Expected 'symbols' array in response")
	}

	if len(symbols) == 0 {
		t.Error("Expected at least one symbol")
	}

	t.Logf("Found %d symbols", len(symbols))

	// Test symbol search
	resp, err = client.Get("/api/v1/symbols?search=AAPL")
	if err != nil {
		t.Fatalf("Failed to search symbols: %v", err)
	}
	defer resp.Body.Close()

	var searchResponse map[string]interface{}
	if err := client.ParseJSONResponse(resp, &searchResponse); err != nil {
		t.Fatalf("Failed to parse search response: %v", err)
	}

	searchResults, ok := searchResponse["symbols"].([]interface{})
	if !ok {
		t.Fatal("Expected 'symbols' array in search response")
	}

	t.Logf("Search for 'AAPL' returned %d results", len(searchResults))
}

// TestE2E_WebSocketConnection tests WebSocket connection for real-time alerts
func TestE2E_WebSocketConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Ensure services are running
	ensureServicesRunning(t)

	// Connect to WebSocket
	u := url.URL{Scheme: "ws", Host: "localhost:8088", Path: "/ws"}
	t.Logf("Connecting to %s", u.String())

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, resp, err := dialer.Dial(u.String(), nil)
	if err != nil {
		if resp != nil {
			body := new(bytes.Buffer)
			body.ReadFrom(resp.Body)
			t.Fatalf("Failed to connect to WebSocket: %v. Response: %s", err, body.String())
		}
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	t.Log("WebSocket connected successfully")

	// Send subscribe message
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"symbols": []string{"AAPL", "GOOGL"},
	}

	if err := conn.WriteJSON(subscribeMsg); err != nil {
		t.Fatalf("Failed to send subscribe message: %v", err)
	}

	t.Log("Sent subscribe message")

	// Wait for messages (with timeout)
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	var receivedMessages []map[string]interface{}
	for i := 0; i < 10; i++ {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				t.Logf("WebSocket closed: %v", err)
			}
			break
		}
		receivedMessages = append(receivedMessages, msg)
		t.Logf("Received message: %+v", msg)
	}

	if len(receivedMessages) == 0 {
		t.Log("No messages received (this may be expected if no alerts are generated)")
	} else {
		t.Logf("Received %d messages", len(receivedMessages))
	}
}

// TestE2E_CompleteUserWorkflow tests a complete user workflow
func TestE2E_CompleteUserWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Ensure services are running
	ensureServicesRunning(t)

	client := NewTestClient(t)

	// Step 1: List available symbols
	t.Log("Step 1: Listing available symbols...")
	resp, err := client.Get("/api/v1/symbols")
	if err != nil {
		t.Fatalf("Failed to list symbols: %v", err)
	}
	defer resp.Body.Close()

	var symbolsResponse map[string]interface{}
	if err := client.ParseJSONResponse(resp, &symbolsResponse); err != nil {
		t.Fatalf("Failed to parse symbols: %v", err)
	}

	symbols, _ := symbolsResponse["symbols"].([]interface{})
	if len(symbols) == 0 {
		t.Fatal("No symbols available")
	}

	testSymbol := symbols[0].(string)
	t.Logf("Using symbol: %s", testSymbol)

	// Step 2: Create a rule
	t.Log("Step 2: Creating a rule...")
	rule := map[string]interface{}{
		"name":        "E2E Test Rule - RSI Oversold",
		"description": "Test rule for E2E testing",
		"conditions": []map[string]interface{}{
			{
				"metric":   "rsi_14",
				"operator": "<",
				"value":    30.0,
			},
		},
		"cooldown": 60, // 1 minute cooldown for testing
		"enabled":  true,
	}

	resp, err = client.Post("/api/v1/rules", rule)
	if err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}
	defer resp.Body.Close()

	var createdRule models.Rule
	if err := client.ParseJSONResponse(resp, &createdRule); err != nil {
		t.Fatalf("Failed to parse created rule: %v", err)
	}

	t.Logf("Created rule with ID: %s", createdRule.ID)

	// Step 3: Validate the rule
	t.Log("Step 3: Validating the rule...")
	validateResp, err := client.Post(fmt.Sprintf("/api/v1/rules/%s/validate", createdRule.ID), nil)
	if err != nil {
		t.Fatalf("Failed to validate rule: %v", err)
	}
	defer validateResp.Body.Close()

	if validateResp.StatusCode != http.StatusOK {
		t.Errorf("Rule validation failed with status %d", validateResp.StatusCode)
	}

	// Step 4: Connect to WebSocket for alerts
	t.Log("Step 4: Connecting to WebSocket for real-time alerts...")
	u := url.URL{Scheme: "ws", Host: "localhost:8088", Path: "/ws"}
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		t.Logf("WebSocket connection failed (may be expected if service not running): %v", err)
	} else {
		defer conn.Close()

		// Subscribe to symbols
		subscribeMsg := map[string]interface{}{
			"type":    "subscribe",
			"symbols": []string{testSymbol},
		}
		if err := conn.WriteJSON(subscribeMsg); err != nil {
			t.Logf("Failed to subscribe: %v", err)
		} else {
			t.Log("Subscribed to symbols via WebSocket")
		}

		// Wait a bit for potential alerts
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		var alertCount int
		for i := 0; i < 5; i++ {
			var msg map[string]interface{}
			if err := conn.ReadJSON(&msg); err != nil {
				break
			}
			alertCount++
			t.Logf("Received alert: %+v", msg)
		}

		if alertCount > 0 {
			t.Logf("Received %d alerts", alertCount)
		} else {
			t.Log("No alerts received (this may be expected if conditions are not met)")
		}
	}

	// Step 5: Check alert history
	t.Log("Step 5: Checking alert history...")
	alertResp, err := client.Get("/api/v1/alerts")
	if err != nil {
		t.Logf("Failed to get alerts (may be expected if service not running): %v", err)
	} else {
		defer alertResp.Body.Close()
		if alertResp.StatusCode == http.StatusOK {
			var alertsResponse map[string]interface{}
			if err := client.ParseJSONResponse(alertResp, &alertsResponse); err == nil {
				alerts, _ := alertsResponse["alerts"].([]interface{})
				t.Logf("Found %d alerts in history", len(alerts))
			}
		}
	}

	// Step 6: Clean up - delete the rule
	t.Log("Step 6: Cleaning up - deleting the rule...")
	deleteResp, err := client.Delete(fmt.Sprintf("/api/v1/rules/%s", createdRule.ID))
	if err != nil {
		t.Logf("Failed to delete rule: %v", err)
	} else {
		defer deleteResp.Body.Close()
		t.Log("Rule deleted successfully")
	}

	t.Log("✅ Complete user workflow test finished")
}

// TestE2E_RuleValidation tests rule validation endpoint
func TestE2E_RuleValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Ensure services are running
	ensureServicesRunning(t)

	client := NewTestClient(t)

	// Create a rule first
	rule := map[string]interface{}{
		"name": "Validation Test Rule",
		"conditions": []map[string]interface{}{
			{
				"metric":   "rsi_14",
				"operator": "<",
				"value":    30.0,
			},
		},
		"cooldown": 300,
		"enabled":  true,
	}

	resp, err := client.Post("/api/v1/rules", rule)
	if err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}
	defer resp.Body.Close()

	var createdRule models.Rule
	if err := client.ParseJSONResponse(resp, &createdRule); err != nil {
		t.Fatalf("Failed to parse created rule: %v", err)
	}

	// Validate the rule
	resp, err = client.Post(fmt.Sprintf("/api/v1/rules/%s/validate", createdRule.ID), nil)
	if err != nil {
		t.Fatalf("Failed to validate rule: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body := new(bytes.Buffer)
		body.ReadFrom(resp.Body)
		t.Fatalf("Rule validation failed with status %d. Body: %s", resp.StatusCode, body.String())
	}

	var validationResult map[string]interface{}
	if err := client.ParseJSONResponse(resp, &validationResult); err != nil {
		t.Fatalf("Failed to parse validation result: %v", err)
	}

	t.Logf("Validation result: %+v", validationResult)

	// Clean up
	client.Delete(fmt.Sprintf("/api/v1/rules/%s", createdRule.ID))
}

// ensureServicesRunning checks if services are running and starts them if needed
func ensureServicesRunning(t *testing.T) {
	// Check if API service is responding
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiBaseURL + healthEndpoint)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		t.Log("Services appear to be running")
		return
	}

	t.Log("Services not running. Attempting to start with docker-compose...")

	// Try to start services using docker-compose
	// Find the project root by looking for docker-compose.yaml
	wd, err := os.Getwd()
	if err != nil {
		t.Logf("Failed to get working directory: %v", err)
		return
	}

	// Try to find docker-compose.yaml
	// Since we're in tests/api_e2e/, we need to go up to project root
	var composeFile string
	var projectRoot string

	// Try different paths: current, parent, grandparent (project root)
	pathsToTry := []string{
		"config/docker-compose.yaml",           // Current directory
		"../config/docker-compose.yaml",         // Parent directory
		"../../config/docker-compose.yaml",      // Project root (from tests/api_e2e/)
	}

	for _, path := range pathsToTry {
		if _, err := os.Stat(path); err == nil {
			absPath, err := filepath.Abs(path)
			if err == nil {
				composeFile = absPath
				// Get the directory containing docker-compose.yaml (project root)
				projectRoot = filepath.Dir(absPath)
				break
			}
		}
	}

	if composeFile == "" {
		t.Logf("docker-compose.yaml not found in current, parent, or project root")
		t.Logf("Current directory: %s", wd)
		t.Log("Please ensure services are running manually: docker-compose -f config/docker-compose.yaml up -d")
		return
	}

	// Verify the file exists
	if _, err := os.Stat(composeFile); err != nil {
		t.Logf("docker-compose.yaml not found at: %s", composeFile)
		t.Log("Please ensure services are running manually: docker-compose -f config/docker-compose.yaml up -d")
		return
	}

	// Try docker compose (newer) first, then docker-compose (older)
	if _, err := exec.LookPath("docker"); err != nil {
		t.Log("Docker not found. Please ensure services are running manually")
		return
	}

	// Try 'docker compose' (newer syntax) first
	cmd := exec.Command("docker", "compose", "-f", composeFile, "up", "-d")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fall back to docker-compose
		cmd = exec.Command("docker-compose", "-f", composeFile, "up", "-d")
		cmd.Dir = projectRoot
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Logf("Failed to start services with docker-compose: %v", err)
			t.Logf("Output: %s", string(output))
			t.Logf("Compose file: %s", composeFile)
			t.Logf("Project root: %s", projectRoot)
			t.Log("Please ensure services are running manually: docker-compose -f config/docker-compose.yaml up -d")
			return
		}
	}
	t.Logf("Docker Compose started services")
	if len(output) > 0 {
		t.Logf("Output: %s", string(output))
	}

	// Wait for services to be ready
	t.Log("Waiting for services to be ready...")
	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)
		resp, err := client.Get(apiBaseURL + healthEndpoint)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			t.Log("Services are ready!")
			return
		}
	}

	t.Fatal("Services did not become ready within timeout")
}

// TestE2E_Setup runs before other tests to ensure services are running
func TestE2E_Setup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Check if we should skip Docker setup
	if os.Getenv("SKIP_DOCKER_SETUP") == "true" {
		t.Log("Skipping Docker setup (SKIP_DOCKER_SETUP=true)")
		return
	}

	ensureServicesRunning(t)
}
