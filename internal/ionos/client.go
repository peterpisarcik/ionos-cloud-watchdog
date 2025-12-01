package ionos

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	DefaultAPIURL = "https://api.ionos.com/cloudapi/v6"
)

type Client struct {
	BaseURL    string
	Token      string
	Username   string
	Password   string
	HTTPClient *http.Client
}

type CheckResult struct {
	OK      bool
	Message string
}

func NewClientFromEnv() (*Client, error) {
	client := &Client{
		BaseURL: DefaultAPIURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	if url := os.Getenv("IONOS_API_URL"); url != "" {
		client.BaseURL = url + "/cloudapi/v6"
	}

	if token := os.Getenv("IONOS_TOKEN"); token != "" {
		client.Token = token
		return client, nil
	}

	username := os.Getenv("IONOS_USERNAME")
	password := os.Getenv("IONOS_PASSWORD")

	if username == "" || password == "" {
		return nil, fmt.Errorf("IONOS credentials not found: set IONOS_TOKEN or IONOS_USERNAME/IONOS_PASSWORD")
	}

	client.Username = username
	client.Password = password

	return client, nil
}

func (c *Client) CheckConnectivity() CheckResult {
	req, err := http.NewRequest("GET", c.BaseURL, nil)
	if err != nil {
		return CheckResult{OK: false, Message: fmt.Sprintf("Failed to create request: %v", err)}
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return CheckResult{OK: false, Message: fmt.Sprintf("API unreachable: %v", err)}
	}
	defer func() { _ = resp.Body.Close() }()

	return CheckResult{OK: true, Message: "IONOS API is reachable"}
}

func (c *Client) CheckAuthentication() CheckResult {
	req, err := http.NewRequest("GET", c.BaseURL+"/datacenters?depth=0&limit=1", nil)
	if err != nil {
		return CheckResult{OK: false, Message: fmt.Sprintf("Failed to create request: %v", err)}
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	} else {
		auth := base64.StdEncoding.EncodeToString([]byte(c.Username + ":" + c.Password))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return CheckResult{OK: false, Message: fmt.Sprintf("Request failed: %v", err)}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 401 {
		return CheckResult{OK: false, Message: "Authentication failed: invalid credentials"}
	}

	if resp.StatusCode == 403 {
		return CheckResult{OK: false, Message: "Authentication failed: insufficient permissions"}
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return CheckResult{OK: true, Message: "Authentication successful"}
	}

	return CheckResult{OK: false, Message: fmt.Sprintf("Unexpected status code: %d", resp.StatusCode)}
}

type DataCenter struct {
	ID         string `json:"id"`
	Properties struct {
		Name     string `json:"name"`
		Location string `json:"location"`
	} `json:"properties"`
}

type DataCentersResponse struct {
	Items []DataCenter `json:"items"`
}

func (c *Client) ListDatacenters() ([]DataCenter, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/datacenters?depth=1", nil)
	if err != nil {
		return nil, err
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	} else {
		auth := base64.StdEncoding.EncodeToString([]byte(c.Username + ":" + c.Password))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result DataCentersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

type Server struct {
	ID         string `json:"id"`
	Properties struct {
		Name    string `json:"name"`
		Cores   int    `json:"cores"`
		Ram     int    `json:"ram"`
		VMState string `json:"vmState"`
	} `json:"properties"`
	Metadata struct {
		State string `json:"state"`
	} `json:"metadata"`
}

type ServersResponse struct {
	Items []Server `json:"items"`
}

type Volume struct {
	ID         string `json:"id"`
	Properties struct {
		Name string  `json:"name"`
		Size float64 `json:"size"`
		Type string  `json:"type"`
	} `json:"properties"`
	Metadata struct {
		State string `json:"state"`
	} `json:"metadata"`
}

type VolumesResponse struct {
	Items []Volume `json:"items"`
}

type DatacenterStatus struct {
	Datacenter DataCenter
	Servers    []Server
	Volumes    []Volume
	Issues     []string
}

type K8sCluster struct {
	ID         string `json:"id"`
	Properties struct {
		Name       string `json:"name"`
		K8sVersion string `json:"k8sVersion"`
	} `json:"properties"`
	Metadata struct {
		State string `json:"state"`
	} `json:"metadata"`
}

type K8sClustersResponse struct {
	Items []K8sCluster `json:"items"`
}

type K8sNodePool struct {
	ID         string `json:"id"`
	Properties struct {
		Name             string `json:"name"`
		NodeCount        int    `json:"nodeCount"`
		K8sVersion       string `json:"k8sVersion"`
		AvailabilityZone string `json:"availabilityZone"`
	} `json:"properties"`
	Metadata struct {
		State string `json:"state"`
	} `json:"metadata"`
}

type K8sNodePoolsResponse struct {
	Items []K8sNodePool `json:"items"`
}

type K8sClusterStatus struct {
	Cluster   K8sCluster
	NodePools []K8sNodePool
	Issues    []string
}

func (c *Client) ListK8sClusters() ([]K8sCluster, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/k8s?depth=1", nil)
	if err != nil {
		return nil, err
	}

	c.setAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result K8sClustersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

func (c *Client) GetK8sNodePools(clusterID string) ([]K8sNodePool, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/k8s/"+clusterID+"/nodepools?depth=1", nil)
	if err != nil {
		return nil, err
	}

	c.setAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result K8sNodePoolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

func (c *Client) CheckK8sClusters() ([]K8sClusterStatus, error) {
	clusters, err := c.ListK8sClusters()
	if err != nil {
		return nil, err
	}

	var statuses []K8sClusterStatus

	for _, cluster := range clusters {
		status := K8sClusterStatus{
			Cluster: cluster,
		}

		if cluster.Metadata.State != "ACTIVE" {
			status.Issues = append(status.Issues, fmt.Sprintf("Cluster state: %s", cluster.Metadata.State))
		}

		nodePools, err := c.GetK8sNodePools(cluster.ID)
		if err != nil {
			status.Issues = append(status.Issues, fmt.Sprintf("Failed to get node pools: %v", err))
		} else {
			status.NodePools = nodePools
			for _, np := range nodePools {
				if np.Metadata.State != "ACTIVE" {
					status.Issues = append(status.Issues, fmt.Sprintf("Node pool %s state: %s", np.Properties.Name, np.Metadata.State))
				}
			}
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

func (c *Client) setAuth(req *http.Request) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	} else {
		auth := base64.StdEncoding.EncodeToString([]byte(c.Username + ":" + c.Password))
		req.Header.Set("Authorization", "Basic "+auth)
	}
}

func (c *Client) GetServers(datacenterID string) ([]Server, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/datacenters/"+datacenterID+"/servers?depth=1", nil)
	if err != nil {
		return nil, err
	}

	c.setAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result ServersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

func (c *Client) GetVolumes(datacenterID string) ([]Volume, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/datacenters/"+datacenterID+"/volumes?depth=1", nil)
	if err != nil {
		return nil, err
	}

	c.setAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result VolumesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

func (c *Client) CheckDatacenters() ([]DatacenterStatus, error) {
	datacenters, err := c.ListDatacenters()
	if err != nil {
		return nil, err
	}

	var statuses []DatacenterStatus

	for _, dc := range datacenters {
		status := DatacenterStatus{
			Datacenter: dc,
		}

		servers, err := c.GetServers(dc.ID)
		if err != nil {
			status.Issues = append(status.Issues, fmt.Sprintf("Failed to get servers: %v", err))
		} else {
			status.Servers = servers
			for _, srv := range servers {
				if srv.Metadata.State == "BUSY" || srv.Metadata.State == "ERROR" {
					status.Issues = append(status.Issues, fmt.Sprintf("Server %s state: %s", srv.Properties.Name, srv.Metadata.State))
				}
			}
		}

		volumes, err := c.GetVolumes(dc.ID)
		if err != nil {
			status.Issues = append(status.Issues, fmt.Sprintf("Failed to get volumes: %v", err))
		} else {
			status.Volumes = volumes
			for _, vol := range volumes {
				if vol.Metadata.State != "AVAILABLE" {
					status.Issues = append(status.Issues, fmt.Sprintf("Volume %s state: %s", vol.Properties.Name, vol.Metadata.State))
				}
			}
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}
