package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const DefaultDashboardConfigPath = "/var/lib/hawk/tmp/dashboard.js"

// DashboardCluster describes a saved remote cluster. The local cluster is implicit.
type DashboardCluster struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	HTTPS    bool   `json:"https"`
	Port     int    `json:"port"`
	Interval int    `json:"interval"`
}

// NewDashboardCluster supplies the defaults used by Hawk.
func NewDashboardCluster(name, host string) DashboardCluster {
	return DashboardCluster{Name: name, Host: host, HTTPS: true, Port: 7630, Interval: 30}
}

// UnmarshalJSON defaults omitted or null settings without replacing explicit false or zero.
func (c *DashboardCluster) UnmarshalJSON(data []byte) error {
	type cluster DashboardCluster
	value := cluster(NewDashboardCluster("", ""))
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*c = DashboardCluster(value)
	return nil
}

// Validate returns field errors suitable for displaying beside the add-cluster form.
func (c DashboardCluster) Validate() map[string][]string {
	errors := make(map[string][]string)
	if strings.TrimSpace(c.Name) == "" {
		errors["name"] = []string{"can't be blank"}
	}
	if strings.TrimSpace(c.Host) == "" {
		errors["host"] = []string{"can't be blank"}
	}
	if c.Port < 1 || c.Port > 65535 {
		errors["port"] = []string{"must be between 1 and 65535"}
	}
	if c.Interval < 1 {
		errors["interval"] = []string{"must be greater than 0"}
	}
	return errors
}

// Serialize updates across store instances in this process. External writers,
// including the Ruby application, do not participate in this lock.
var dashboardConfigMu sync.Mutex

// DashboardConfigStore reads and updates Hawk's name-keyed JSON configuration.
// Construct once and reuse; an empty path selects DefaultDashboardConfigPath.
type DashboardConfigStore struct {
	path        string
	copyCluster func(string) error
}

func NewDashboardConfigStore(path string) *DashboardConfigStore {
	if path == "" {
		path = DefaultDashboardConfigPath
	}
	return &DashboardConfigStore{path: path, copyCluster: func(path string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		output, err := exec.CommandContext(ctx, "/usr/sbin/crm", "cluster", "copy", path).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
		}
		return nil
	}}
}

func (s *DashboardConfigStore) Load() (map[string]DashboardCluster, error) {
	dashboardConfigMu.Lock()
	defer dashboardConfigMu.Unlock()
	return s.load()
}

func (s *DashboardConfigStore) load() (map[string]DashboardCluster, error) {
	clusters := make(map[string]DashboardCluster)
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return clusters, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read dashboard configuration: %w", err)
	}
	// An existing but empty file has no saved clusters yet.
	if strings.TrimSpace(string(data)) == "" {
		return clusters, nil
	}
	if err := json.Unmarshal(data, &clusters); err != nil {
		return nil, fmt.Errorf("parse dashboard configuration: %w", err)
	}
	if clusters == nil {
		return nil, fmt.Errorf("dashboard configuration must be a JSON object")
	}
	for name, cluster := range clusters {
		if name != cluster.Name {
			return nil, fmt.Errorf("dashboard cluster key %q does not match name %q", name, cluster.Name)
		}
		if errors := cluster.Validate(); len(errors) > 0 {
			return nil, fmt.Errorf("invalid dashboard cluster %q: %v", name, errors)
		}
	}
	return clusters, nil
}

// Save adds or replaces a cluster with the same name, matching Hawk's behavior.
func (s *DashboardConfigStore) Save(cluster DashboardCluster) error {
	if errors := cluster.Validate(); len(errors) > 0 {
		return fmt.Errorf("invalid dashboard cluster: %v", errors)
	}
	dashboardConfigMu.Lock()
	defer dashboardConfigMu.Unlock()
	clusters, err := s.load()
	if err != nil {
		return err
	}
	clusters[cluster.Name] = cluster
	return s.write(clusters)
}

// Remove is a no-op if the named cluster is not saved.
func (s *DashboardConfigStore) Remove(name string) error {
	dashboardConfigMu.Lock()
	defer dashboardConfigMu.Unlock()
	clusters, err := s.load()
	if err != nil {
		return err
	}
	if _, exists := clusters[name]; !exists {
		return nil
	}
	delete(clusters, name)
	return s.write(clusters)
}

func (s *DashboardConfigStore) write(clusters map[string]DashboardCluster) error {
	data, err := json.MarshalIndent(clusters, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(s.path), ".dashboard-*.json")
	if err != nil {
		return fmt.Errorf("create dashboard configuration: %w", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err := file.Chmod(0660); err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), s.path); err != nil {
		return err
	}
	// A replication failure must not report an already committed local save as failed.
	if s.copyCluster != nil {
		if err := s.copyCluster(s.path); err != nil {
			log.WithError(err).Warn("Could not copy dashboard configuration to cluster nodes")
		}
	}
	return nil
}
