package ionos

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	PostgreSQLAPIURL = "https://api.ionos.com/databases/postgresql"
	MongoDBAPIURL    = "https://api.ionos.com/databases/mongodb"
	MariaDBAPIURL    = "https://api.ionos.com/databases/mariadb"
	InMemoryDBAPIURL = "https://api.ionos.com/databases/in-memory-db"
)

type PostgreSQLCluster struct {
	ID         string `json:"id"`
	Properties struct {
		DisplayName     string `json:"displayName"`
		PostgresVersion string `json:"postgresVersion"`
		Location        string `json:"location"`
		Instances       int    `json:"instances"`
	} `json:"properties"`
	Metadata struct {
		State string `json:"state"`
	} `json:"metadata"`
}

type PostgreSQLClustersResponse struct {
	Items []PostgreSQLCluster `json:"items"`
}

type MongoDBCluster struct {
	ID         string `json:"id"`
	Properties struct {
		DisplayName    string `json:"displayName"`
		MongoDBVersion string `json:"mongoDBVersion"`
		Location       string `json:"location"`
		Instances      int    `json:"instances"`
		Edition        string `json:"edition"`
	} `json:"properties"`
	Metadata struct {
		State string `json:"state"`
	} `json:"metadata"`
}

type MongoDBClustersResponse struct {
	Items []MongoDBCluster `json:"items"`
}

type MariaDBCluster struct {
	ID         string `json:"id"`
	Properties struct {
		DisplayName    string `json:"displayName"`
		MariaDBVersion string `json:"mariadbVersion"`
		Location       string `json:"location"`
		Instances      int    `json:"instances"`
	} `json:"properties"`
	Metadata struct {
		State string `json:"state"`
	} `json:"metadata"`
}

type MariaDBClustersResponse struct {
	Items []MariaDBCluster `json:"items"`
}

type InMemoryDBInstance struct {
	ID         string `json:"id"`
	Properties struct {
		DisplayName string `json:"displayName"`
		Version     string `json:"version"`
		Location    string `json:"location"`
		Replicas    int    `json:"replicas"`
	} `json:"properties"`
	Metadata struct {
		State string `json:"state"`
	} `json:"metadata"`
}

type InMemoryDBInstancesResponse struct {
	Items []InMemoryDBInstance `json:"items"`
}

type DBaaSStatus struct {
	PostgreSQL []PostgreSQLCluster
	MongoDB    []MongoDBCluster
	MariaDB    []MariaDBCluster
	InMemoryDB []InMemoryDBInstance
	Issues     []string
}

func (c *Client) makeDBaaSRequest(url string, result interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	c.setAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 404 {
		return nil
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return err
	}

	return nil
}

func (c *Client) ListPostgreSQLClusters() ([]PostgreSQLCluster, error) {
	var response PostgreSQLClustersResponse
	err := c.makeDBaaSRequest(c.PostgreSQLBaseURL+"/clusters", &response)
	if err != nil {
		return nil, err
	}
	return response.Items, nil
}

func (c *Client) ListMongoDBClusters() ([]MongoDBCluster, error) {
	var response MongoDBClustersResponse
	err := c.makeDBaaSRequest(c.MongoDBBaseURL+"/clusters", &response)
	if err != nil {
		return nil, err
	}
	return response.Items, nil
}

func (c *Client) ListMariaDBClusters() ([]MariaDBCluster, error) {
	var response MariaDBClustersResponse
	err := c.makeDBaaSRequest(c.MariaDBBaseURL+"/clusters", &response)
	if err != nil {
		return nil, err
	}
	return response.Items, nil
}

func (c *Client) ListInMemoryDBInstances() ([]InMemoryDBInstance, error) {
	var response InMemoryDBInstancesResponse
	err := c.makeDBaaSRequest(c.InMemoryDBBaseURL+"/instances", &response)
	if err != nil {
		return nil, err
	}
	return response.Items, nil
}

func (c *Client) CheckDBaaS() DBaaSStatus {
	status := DBaaSStatus{}

	pgClusters, err := c.ListPostgreSQLClusters()
	if err != nil {
		status.Issues = append(status.Issues, fmt.Sprintf("Failed to get PostgreSQL clusters: %v", err))
	} else {
		status.PostgreSQL = pgClusters
		for _, cluster := range pgClusters {
			if cluster.Metadata.State != "AVAILABLE" && cluster.Metadata.State != "ACTIVE" {
				status.Issues = append(status.Issues,
					fmt.Sprintf("PostgreSQL cluster %s state: %s", cluster.Properties.DisplayName, cluster.Metadata.State))
			}
		}
	}

	mongoClusters, err := c.ListMongoDBClusters()
	if err != nil {
		status.Issues = append(status.Issues, fmt.Sprintf("Failed to get MongoDB clusters: %v", err))
	} else {
		status.MongoDB = mongoClusters
		for _, cluster := range mongoClusters {
			if cluster.Metadata.State != "AVAILABLE" && cluster.Metadata.State != "ACTIVE" {
				status.Issues = append(status.Issues,
					fmt.Sprintf("MongoDB cluster %s state: %s", cluster.Properties.DisplayName, cluster.Metadata.State))
			}
		}
	}

	mariadbClusters, err := c.ListMariaDBClusters()
	if err != nil {
		status.Issues = append(status.Issues, fmt.Sprintf("Failed to get MariaDB clusters: %v", err))
	} else {
		status.MariaDB = mariadbClusters
		for _, cluster := range mariadbClusters {
			if cluster.Metadata.State != "AVAILABLE" && cluster.Metadata.State != "ACTIVE" {
				status.Issues = append(status.Issues,
					fmt.Sprintf("MariaDB cluster %s state: %s", cluster.Properties.DisplayName, cluster.Metadata.State))
			}
		}
	}

	inMemoryInstances, err := c.ListInMemoryDBInstances()
	if err != nil {
		status.Issues = append(status.Issues, fmt.Sprintf("Failed to get In-Memory DB instances: %v", err))
	} else {
		status.InMemoryDB = inMemoryInstances
		for _, instance := range inMemoryInstances {
			if instance.Metadata.State != "AVAILABLE" && instance.Metadata.State != "ACTIVE" {
				status.Issues = append(status.Issues,
					fmt.Sprintf("In-Memory DB instance %s state: %s", instance.Properties.DisplayName, instance.Metadata.State))
			}
		}
	}

	return status
}
