package ionos

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNewClientFromEnv_WithToken(t *testing.T) {
	setEnv(t, "IONOS_TOKEN", "token-value")
	setEnv(t, "IONOS_USERNAME", "")
	setEnv(t, "IONOS_PASSWORD", "")
	setEnv(t, "IONOS_API_URL", "https://custom.example.com")

	client, err := NewClientFromEnv()
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	if client.Token != "token-value" {
		t.Fatalf("expected token to be set")
	}
	if client.BaseURL != "https://custom.example.com/cloudapi/v6" {
		t.Fatalf("unexpected BaseURL: %s", client.BaseURL)
	}
}

func TestNewClientFromEnv_WithUsernamePassword(t *testing.T) {
	setEnv(t, "IONOS_TOKEN", "")
	setEnv(t, "IONOS_USERNAME", "user")
	setEnv(t, "IONOS_PASSWORD", "pass")
	setEnv(t, "IONOS_API_URL", "")

	client, err := NewClientFromEnv()
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	if client.Token != "" {
		t.Fatalf("expected token to be empty")
	}
	if client.Username != "user" || client.Password != "pass" {
		t.Fatalf("expected username/password to be set")
	}
	if client.BaseURL != DefaultAPIURL {
		t.Fatalf("unexpected BaseURL: %s", client.BaseURL)
	}
}

func TestNewClientFromEnv_MissingCredentials(t *testing.T) {
	setEnv(t, "IONOS_TOKEN", "")
	setEnv(t, "IONOS_USERNAME", "")
	setEnv(t, "IONOS_PASSWORD", "")

	if _, err := NewClientFromEnv(); err == nil {
		t.Fatalf("expected error for missing credentials")
	}
}

func TestCheckAuthentication_WithToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected Authorization header: %s", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		Token:      "test-token",
		HTTPClient: server.Client(),
	}

	result := client.CheckAuthentication()

	if !result.OK {
		t.Fatalf("expected authentication to succeed, got: %s", result.Message)
	}
}

func TestCheckAuthentication_WithBasicAuth(t *testing.T) {
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != expectedAuth {
			t.Fatalf("unexpected Authorization header: %s", got)
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		Username:   "user",
		Password:   "pass",
		HTTPClient: server.Client(),
	}

	result := client.CheckAuthentication()

	if result.OK {
		t.Fatalf("expected authentication to fail")
	}
	if result.Message != "Authentication failed: invalid credentials" {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}

func TestCheckAuthentication_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		Token:      "token",
		HTTPClient: server.Client(),
	}

	result := client.CheckAuthentication()

	if result.OK {
		t.Fatalf("expected authentication to fail with forbidden")
	}
	if result.Message != "Authentication failed: insufficient permissions" {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}

func TestCheckDatacenters_CollectsIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/datacenters":
			requireAuthHeader(t, r)
			resp := DataCentersResponse{
				Items: []DataCenter{{
					ID: "dc1",
					Properties: struct {
						Name     string "json:\"name\""
						Location string "json:\"location\""
					}{Name: "DC One", Location: "de/fra"},
				}},
			}
			_ = json.NewEncoder(w).Encode(resp)
		case "/datacenters/dc1/servers":
			requireAuthHeader(t, r)
			resp := ServersResponse{
				Items: []Server{
					{
						ID: "srv-ok",
						Properties: struct {
							Name    string "json:\"name\""
							Cores   int    "json:\"cores\""
							Ram     int    "json:\"ram\""
							VMState string "json:\"vmState\""
						}{Name: "web-1", VMState: "AVAILABLE"},
						Metadata: struct {
							State string "json:\"state\""
						}{State: "AVAILABLE"},
					},
					{
						ID: "srv-busy",
						Properties: struct {
							Name    string "json:\"name\""
							Cores   int    "json:\"cores\""
							Ram     int    "json:\"ram\""
							VMState string "json:\"vmState\""
						}{Name: "web-2"},
						Metadata: struct {
							State string "json:\"state\""
						}{State: "BUSY"},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		case "/datacenters/dc1/volumes":
			requireAuthHeader(t, r)
			resp := VolumesResponse{
				Items: []Volume{
					{
						ID: "vol-ok",
						Properties: struct {
							Name string  "json:\"name\""
							Size float64 "json:\"size\""
							Type string  "json:\"type\""
						}{Name: "vol1", Size: 10, Type: "HDD"},
						Metadata: struct {
							State string "json:\"state\""
						}{State: "AVAILABLE"},
					},
					{
						ID: "vol-busy",
						Properties: struct {
							Name string  "json:\"name\""
							Size float64 "json:\"size\""
							Type string  "json:\"type\""
						}{Name: "vol2", Size: 20, Type: "SSD"},
						Metadata: struct {
							State string "json:\"state\""
						}{State: "BUSY"},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		Token:      "token",
		HTTPClient: server.Client(),
	}

	statuses, err := client.CheckDatacenters()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(statuses) != 1 {
		t.Fatalf("expected 1 datacenter, got %d", len(statuses))
	}

	issues := statuses[0].Issues
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d: %v", len(issues), issues)
	}
	assertContains(t, issues, "Server web-2 state: BUSY")
	assertContains(t, issues, "Volume vol2 state: BUSY")
}

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	orig := os.Getenv(key)
	if value == "" {
		_ = os.Unsetenv(key)
	} else {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("failed to set env %s: %v", key, err)
		}
	}
	t.Cleanup(func() {
		if orig == "" {
			_ = os.Unsetenv(key)
			return
		}
		_ = os.Setenv(key, orig)
	})
}

func requireAuthHeader(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Header.Get("Authorization") == "" {
		t.Fatalf("authorization header not set")
	}
}

func assertContains(t *testing.T, list []string, expected string) {
	t.Helper()
	for _, item := range list {
		if item == expected {
			return
		}
	}
	t.Fatalf("expected %q in %v", expected, list)
}
