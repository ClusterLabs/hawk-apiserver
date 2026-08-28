package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestDashboardClusterDefaultsAndValidation(t *testing.T) {
	for _, test := range []struct {
		data  string
		https bool
		port  int
		valid bool
	}{
		{`{"name":"west","host":"node1"}`, true, 7630, true},
		{`{"name":"west","host":"node1","https":false,"port":443}`, false, 443, true},
		{`{"name":"west","host":"node1","https":null,"port":null}`, true, 7630, true},
		{`{"name":"west","host":"node1","port":0}`, true, 0, false},
		{`{"name":"west","host":"node1","port":65536}`, true, 65536, false},
		{`{"name":" ","host":"\t","interval":0}`, true, 7630, false},
	} {
		var c DashboardCluster
		if err := json.Unmarshal([]byte(test.data), &c); err != nil {
			t.Fatal(err)
		}
		if c.HTTPS != test.https || c.Port != test.port || (len(c.Validate()) == 0) != test.valid {
			t.Fatalf("unexpected configuration: %+v", c)
		}
	}
}

func TestDashboardConfigPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dashboard.js")
	store := NewDashboardConfigStore(path)
	copies := 0
	store.copyCluster = func(p string) error { copies++; return fmt.Errorf("replication unavailable") }
	if clusters, err := store.Load(); err != nil || len(clusters) != 0 {
		t.Fatalf("missing file: %v %v", clusters, err)
	}
	c := NewDashboardCluster("west", "node1")
	if err := store.Save(c); err != nil {
		t.Fatal(err)
	}
	c.Host = "node2"
	c.HTTPS = false
	if err := store.Save(c); err != nil {
		t.Fatal(err)
	}
	clusters, err := store.Load()
	if err != nil || len(clusters) != 1 || clusters["west"] != c {
		t.Fatalf("round trip: %v %v", clusters, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0660 {
		t.Fatalf("permissions: %v", info.Mode())
	}
	if err := store.Remove("west"); err != nil {
		t.Fatal(err)
	}
	if err := store.Remove("missing"); err != nil {
		t.Fatal(err)
	}
	clusters, err = store.Load()
	if err != nil || len(clusters) != 0 || copies != 3 {
		t.Fatalf("remove: %v %v copies=%d", clusters, err, copies)
	}
}

func TestDashboardConfigEmptyFile(t *testing.T) {
	for _, data := range []string{"", " \n\t\r"} {
		path := filepath.Join(t.TempDir(), "dashboard.js")
		if err := os.WriteFile(path, []byte(data), 0660); err != nil {
			t.Fatal(err)
		}
		store := NewDashboardConfigStore(path)
		store.copyCluster = nil
		clusters, err := store.Load()
		if err != nil || len(clusters) != 0 {
			t.Fatalf("empty file: %v %v", clusters, err)
		}
		cluster := NewDashboardCluster("west", "node1")
		if err := store.Save(cluster); err != nil {
			t.Fatal(err)
		}
		clusters, err = store.Load()
		if err != nil || len(clusters) != 1 || clusters["west"] != cluster {
			t.Fatalf("save to empty file: %v %v", clusters, err)
		}
	}
}

func TestDashboardConfigRejectsCorruptFile(t *testing.T) {
	for _, data := range []string{`{`, `null`, `{"west":{"name":"east","host":"node"}}`} {
		path := filepath.Join(t.TempDir(), "dashboard.js")
		if err := os.WriteFile(path, []byte(data), 0660); err != nil {
			t.Fatal(err)
		}
		store := NewDashboardConfigStore(path)
		store.copyCluster = nil
		if err := store.Save(NewDashboardCluster("new", "node")); err == nil {
			t.Fatal("expected error")
		}
		after, err := os.ReadFile(path)
		if err != nil || string(after) != data {
			t.Fatalf("corrupt file changed: %s %v", after, err)
		}
	}
}

func TestDashboardConfigConcurrentUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dashboard.js")
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			store := NewDashboardConfigStore(path)
			store.copyCluster = nil
			if err := store.Save(NewDashboardCluster(fmt.Sprintf("cluster%d", i), "node")); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	clusters, err := NewDashboardConfigStore(path).Load()
	if err != nil || len(clusters) != 12 {
		t.Fatalf("lost updates: %v %v", clusters, err)
	}
}
