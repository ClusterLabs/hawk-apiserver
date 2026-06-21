package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/ClusterLabs/hawk-apiserver/api"
	"github.com/ClusterLabs/hawk-apiserver/internal"
	"github.com/ClusterLabs/hawk-apiserver/server"
	logrus "github.com/sirupsen/logrus"
)

const hawkRubySock = "/usr/share/hawk/tmp/hawk.sock"

type rubyAuthResp struct {
	OK   bool   `json:"ok"`
	User string `json:"user"`
}

/*
hawk-apiserver doesn't do auth right now (but will do it in later versions)
instead, it asks the hawk (RoR) if it's authenticated
each time inside /cib/live/primitives/{primitive-id}/edit page.
Thought it's stressfull, but it happens only in one page right now (as of 18.12.2025)
and in future, Go should do all the auth routines. (#TODO)
*/
func authViaRuby(r *http.Request) (ok bool, user string, err error) {
	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := net.Dialer{Timeout: 2 * time.Second}
			return d.DialContext(ctx, "unix", hawkRubySock)
		},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   3 * time.Second,
	}

	// URL host is dummy; DialContext ignores it and uses the unix socket.
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, "http://localhost/internal/auth", nil)
	if err != nil {
		return false, "", err
	}

	// Forward cookies from the *incoming* browser request to Ruby.
	if ck := r.Header.Get("Cookie"); ck != "" {
		req.Header.Set("Cookie", ck)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	var out rubyAuthResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, "", err
	}

	return out.OK, out.User, nil
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ok, user, err := authViaRuby(r)
		if err != nil {
			log.Printf("[auth] ruby auth error: %v", err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if !ok {
			log.Printf("[auth] ruby auth: forbidden")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// (Optional) log who Ruby thinks it is
		_ = user

		next.ServeHTTP(w, r)
	}
}

// the released version of the binary. Generated via makefile or buildsystem
var version = "was not built correctly"

func main() {
	// initialize configuration, which handle the different routes
	config := internal.InitConfig(version)
	routehandler := internal.NewRouteHandler(&config)
	api.Routehandler = routehandler

	// binds to pacemaker-go and serve CIB async info
	routehandler.Cib.Start()

	logrus.Infof("Listening to https://%s:%d\n", config.Listen, config.Port)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/cib/primitive/create", authMiddleware(api.PrimitiveCreateHandler))
	mux.HandleFunc("/api/cib/primitive/update", authMiddleware(api.PrimitiveUpdateHandler)) // Can't find where it was used (let's not remove it though)
	mux.HandleFunc("/api/cib/primitive/rename", authMiddleware(api.PrimitiveRenameHandler))
	mux.HandleFunc("/api/cib/primitive/delete", authMiddleware(api.PrimitiveDeleteHandler))
	mux.HandleFunc("/api/data-interface/fetch-cluster-details", authMiddleware(api.FetchClusterDetails))
	mux.HandleFunc("/api/data-interface/fetch-resource-classes", authMiddleware(api.FetchResourceClasses))
	mux.HandleFunc("/api/data-interface/fetch-resource-providers", authMiddleware(api.FetchResourceProviders))
	mux.HandleFunc("/api/data-interface/fetch-resource-types", authMiddleware(api.FetchResourceTypes))
	mux.HandleFunc("/api/data-interface/fetch-resource-params", authMiddleware(api.FetchResourceParams))
	mux.HandleFunc("/api/data-interface/submit-resource-params", authMiddleware(api.SubmitResourceParams))
	mux.HandleFunc("/api/data-interface/fetch-resource-meta-attributes", authMiddleware(api.FetchResourceMetaAttributes))
	mux.HandleFunc("/api/data-interface/submit-resource-meta-attributes", authMiddleware(api.SubmitResourceMetaAttributes))
	mux.HandleFunc("/api/data-interface/fetch-resource-operations", authMiddleware(api.FetchResourceOperations))
	mux.HandleFunc("/api/data-interface/submit-resource-operations", authMiddleware(api.SubmitResourceOperations))
	mux.HandleFunc("/api/data-interface/resource-operation/fetch-attributes", authMiddleware(api.FetchResourceOperationAttributes))

	// Register BOTH /cib/live/primitives and /cib/live/primitives/ to avoid a conflicts with Ruby
	mux.HandleFunc("/cib/live/primitives", authMiddleware(api.ResourceEditHandler))
	mux.HandleFunc("/cib/live/primitives/", authMiddleware(api.ResourceEditHandler))

	mux.Handle("/", routehandler) // routehandler is a fallback

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	//TODO: this function should return errors
	// an https server with a reverse proxy. http is redirected to https
	server.ListenAndServeWithRedirect(fmt.Sprintf("%s:%d", config.Listen, config.Port), mux, config.Cert, config.Key)
}
