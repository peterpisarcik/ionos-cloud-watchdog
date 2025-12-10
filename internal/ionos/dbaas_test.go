package ionos

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(serverURL string) *Client {
	return &Client{
		BaseURL:           serverURL,
		PostgreSQLBaseURL: serverURL,
		MongoDBBaseURL:    serverURL,
		MariaDBBaseURL:    serverURL,
		InMemoryDBBaseURL: serverURL,
		Token:             "token",
		HTTPClient:        &http.Client{},
	}
}

func TestListPostgreSQLClusters_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAuthHeader(t, r)
		if r.URL.Path != "/clusters" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		resp := PostgreSQLClustersResponse{
			Items: []PostgreSQLCluster{
				{
					ID: "pg-1",
					Properties: struct {
						DisplayName     string `json:"displayName"`
						PostgresVersion string `json:"postgresVersion"`
						Location        string `json:"location"`
						Instances       int    `json:"instances"`
					}{
						DisplayName:     "my-postgres",
						PostgresVersion: "15",
						Location:        "de/fra",
						Instances:       3,
					},
					Metadata: struct {
						State string `json:"state"`
					}{State: "AVAILABLE"},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	clusters, err := client.ListPostgreSQLClusters()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}

	cluster := clusters[0]
	if cluster.Properties.DisplayName != "my-postgres" {
		t.Fatalf("unexpected name: %s", cluster.Properties.DisplayName)
	}
	if cluster.Properties.PostgresVersion != "15" {
		t.Fatalf("unexpected version: %s", cluster.Properties.PostgresVersion)
	}
	if cluster.Metadata.State != "AVAILABLE" {
		t.Fatalf("unexpected state: %s", cluster.Metadata.State)
	}
}

func TestListMongoDBClusters_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAuthHeader(t, r)
		resp := MongoDBClustersResponse{
			Items: []MongoDBCluster{
				{
					ID: "mongo-1",
					Properties: struct {
						DisplayName    string `json:"displayName"`
						MongoDBVersion string `json:"mongoDBVersion"`
						Location       string `json:"location"`
						Instances      int    `json:"instances"`
						Edition        string `json:"edition"`
					}{
						DisplayName:    "my-mongo",
						MongoDBVersion: "6.0",
						Location:       "de/txl",
						Instances:      3,
						Edition:        "enterprise",
					},
					Metadata: struct {
						State string `json:"state"`
					}{State: "ACTIVE"},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	clusters, err := client.ListMongoDBClusters()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}

	cluster := clusters[0]
	if cluster.Properties.DisplayName != "my-mongo" {
		t.Fatalf("unexpected name: %s", cluster.Properties.DisplayName)
	}
	if cluster.Properties.Edition != "enterprise" {
		t.Fatalf("unexpected edition: %s", cluster.Properties.Edition)
	}
}

func TestListMariaDBClusters_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAuthHeader(t, r)
		resp := MariaDBClustersResponse{
			Items: []MariaDBCluster{
				{
					ID: "maria-1",
					Properties: struct {
						DisplayName    string `json:"displayName"`
						MariaDBVersion string `json:"mariadbVersion"`
						Location       string `json:"location"`
						Instances      int    `json:"instances"`
					}{
						DisplayName:    "my-mariadb",
						MariaDBVersion: "10.6",
						Location:       "us-ewr",
						Instances:      2,
					},
					Metadata: struct {
						State string `json:"state"`
					}{State: "AVAILABLE"},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	clusters, err := client.ListMariaDBClusters()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}

	cluster := clusters[0]
	if cluster.Properties.MariaDBVersion != "10.6" {
		t.Fatalf("unexpected version: %s", cluster.Properties.MariaDBVersion)
	}
}

func TestListInMemoryDBInstances_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAuthHeader(t, r)
		resp := InMemoryDBInstancesResponse{
			Items: []InMemoryDBInstance{
				{
					ID: "redis-1",
					Properties: struct {
						DisplayName string `json:"displayName"`
						Version     string `json:"version"`
						Location    string `json:"location"`
						Replicas    int    `json:"replicas"`
					}{
						DisplayName: "my-redis",
						Version:     "7.0",
						Location:    "de/fra",
						Replicas:    2,
					},
					Metadata: struct {
						State string `json:"state"`
					}{State: "AVAILABLE"},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	instances, err := client.ListInMemoryDBInstances()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(instances))
	}

	instance := instances[0]
	if instance.Properties.DisplayName != "my-redis" {
		t.Fatalf("unexpected name: %s", instance.Properties.DisplayName)
	}
	if instance.Properties.Replicas != 2 {
		t.Fatalf("unexpected replicas: %d", instance.Properties.Replicas)
	}
}

func TestDBaaS_404ReturnsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAuthHeader(t, r)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	clusters, err := client.ListPostgreSQLClusters()
	if err != nil {
		t.Fatalf("expected no error for 404, got: %v", err)
	}
	if len(clusters) != 0 {
		t.Fatalf("expected empty list for 404, got %d clusters", len(clusters))
	}
}

func TestDBaaS_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAuthHeader(t, r)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	_, err := client.ListPostgreSQLClusters()
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestCheckDBaaS_CollectsIssues(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAuthHeader(t, r)
		callCount++

		switch r.URL.Path {
		case "/clusters":
			switch callCount {
			case 1:
				resp := PostgreSQLClustersResponse{
					Items: []PostgreSQLCluster{
						{
							ID: "pg-ok",
							Properties: struct {
								DisplayName     string `json:"displayName"`
								PostgresVersion string `json:"postgresVersion"`
								Location        string `json:"location"`
								Instances       int    `json:"instances"`
							}{DisplayName: "pg-healthy"},
							Metadata: struct {
								State string `json:"state"`
							}{State: "AVAILABLE"},
						},
						{
							ID: "pg-bad",
							Properties: struct {
								DisplayName     string `json:"displayName"`
								PostgresVersion string `json:"postgresVersion"`
								Location        string `json:"location"`
								Instances       int    `json:"instances"`
							}{DisplayName: "pg-unhealthy"},
							Metadata: struct {
								State string `json:"state"`
							}{State: "BUSY"},
						},
					},
				}
				_ = json.NewEncoder(w).Encode(resp)
			case 2:
				resp := MongoDBClustersResponse{
					Items: []MongoDBCluster{
						{
							ID: "mongo-bad",
							Properties: struct {
								DisplayName    string `json:"displayName"`
								MongoDBVersion string `json:"mongoDBVersion"`
								Location       string `json:"location"`
								Instances      int    `json:"instances"`
								Edition        string `json:"edition"`
							}{DisplayName: "mongo-unhealthy"},
							Metadata: struct {
								State string `json:"state"`
							}{State: "UPDATING"},
						},
					},
				}
				_ = json.NewEncoder(w).Encode(resp)
			case 3:
				resp := MariaDBClustersResponse{Items: []MariaDBCluster{}}
				_ = json.NewEncoder(w).Encode(resp)
			}
		case "/instances":
			resp := InMemoryDBInstancesResponse{Items: []InMemoryDBInstance{}}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	status := client.CheckDBaaS()

	if len(status.PostgreSQL) != 2 {
		t.Fatalf("expected 2 PostgreSQL clusters, got %d", len(status.PostgreSQL))
	}
	if len(status.MongoDB) != 1 {
		t.Fatalf("expected 1 MongoDB cluster, got %d", len(status.MongoDB))
	}
	if len(status.MariaDB) != 0 {
		t.Fatalf("expected 0 MariaDB clusters, got %d", len(status.MariaDB))
	}
	if len(status.InMemoryDB) != 0 {
		t.Fatalf("expected 0 InMemoryDB instances, got %d", len(status.InMemoryDB))
	}

	if len(status.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d: %v", len(status.Issues), status.Issues)
	}

	assertContains(t, status.Issues, "PostgreSQL cluster pg-unhealthy state: BUSY")
	assertContains(t, status.Issues, "MongoDB cluster mongo-unhealthy state: UPDATING")
}

func TestCheckDBaaS_NoIssuesWhenAllHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAuthHeader(t, r)
		switch r.URL.Path {
		case "/clusters":
			resp := PostgreSQLClustersResponse{
				Items: []PostgreSQLCluster{
					{
						ID: "pg-1",
						Properties: struct {
							DisplayName     string `json:"displayName"`
							PostgresVersion string `json:"postgresVersion"`
							Location        string `json:"location"`
							Instances       int    `json:"instances"`
						}{DisplayName: "pg-healthy"},
						Metadata: struct {
							State string `json:"state"`
						}{State: "AVAILABLE"},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		case "/instances":
			resp := InMemoryDBInstancesResponse{Items: []InMemoryDBInstance{}}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	status := client.CheckDBaaS()

	if len(status.Issues) != 0 {
		t.Fatalf("expected no issues, got %d: %v", len(status.Issues), status.Issues)
	}
	if len(status.PostgreSQL) != 1 {
		t.Fatalf("expected 1 PostgreSQL cluster, got %d", len(status.PostgreSQL))
	}
}

func TestCheckDBaaS_ActiveStateIsHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAuthHeader(t, r)
		if r.URL.Path == "/clusters" {
			resp := MongoDBClustersResponse{
				Items: []MongoDBCluster{
					{
						ID: "mongo-1",
						Properties: struct {
							DisplayName    string `json:"displayName"`
							MongoDBVersion string `json:"mongoDBVersion"`
							Location       string `json:"location"`
							Instances      int    `json:"instances"`
							Edition        string `json:"edition"`
						}{DisplayName: "mongo-active"},
						Metadata: struct {
							State string `json:"state"`
						}{State: "ACTIVE"},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	status := client.CheckDBaaS()

	if len(status.Issues) != 0 {
		t.Fatalf("expected no issues for ACTIVE state, got: %v", status.Issues)
	}
}
