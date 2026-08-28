package api

import (
	"encoding/json"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/ClusterLabs/hawk-apiserver/internal"
)

type dashboardStore interface {
	Load() (map[string]internal.DashboardCluster, error)
	Save(internal.DashboardCluster) error
	Remove(string) error
}

// NewDashboardClustersHandler exposes saved remote clusters. The caller must
// wrap this handler with authentication, as with the other configuration APIs.
func NewDashboardClustersHandler(store dashboardStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		respond := func(status int, value any) {
			w.WriteHeader(status)
			if err := json.NewEncoder(w).Encode(value); err != nil {
				log.Printf("dashboard response: %v", err)
			}
		}
		storageError := func(err error) {
			log.Printf("dashboard configuration: %v", err)
			respond(http.StatusInternalServerError, map[string]string{"error": "Unable to access dashboard configuration"})
		}
		switch r.Method {
		case http.MethodGet:
			clusters, err := store.Load()
			if err != nil {
				storageError(err)
				return
			}
			respond(http.StatusOK, clusters)
		case http.MethodPost, http.MethodDelete:
			mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mediaType != "application/json" {
				respond(http.StatusUnsupportedMediaType, map[string]string{"error": "Content-Type must be application/json"})
				return
			}
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
			var cluster internal.DashboardCluster
			if err := decoder.Decode(&cluster); err != nil {
				respond(http.StatusBadRequest, map[string]string{"error": "Invalid cluster JSON"})
				return
			}
			var extra any
			if err := decoder.Decode(&extra); err != io.EOF {
				respond(http.StatusBadRequest, map[string]string{"error": "Expected a single JSON object"})
				return
			}
			if r.Method == http.MethodPost {
				if errors := cluster.Validate(); len(errors) > 0 {
					respond(http.StatusUnprocessableEntity, errors)
					return
				}
				if err := store.Save(cluster); err != nil {
					storageError(err)
					return
				}
				respond(http.StatusOK, cluster)
			} else {
				if strings.TrimSpace(cluster.Name) == "" {
					respond(http.StatusUnprocessableEntity, map[string][]string{"name": {"can't be blank"}})
					return
				}
				if err := store.Remove(cluster.Name); err != nil {
					storageError(err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		default:
			w.Header().Set("Allow", "GET, POST, DELETE")
			respond(http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		}
	}
}

// DashboardStatusCORS follows Hawk's existing credentialed CORS policy for
// read-only status requests. Configuration mutation endpoints do not use it.
func DashboardStatusCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			allowed := os.Getenv("HAWK_ACCESS_CONTROL_ALLOW_ORIGIN")
			if allowed == "" {
				allowed = origin
			}
			w.Header().Set("Access-Control-Allow-Origin", allowed)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST, OPTIONS")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		next(w, r)
	}
}
