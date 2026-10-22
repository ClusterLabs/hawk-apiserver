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

	liveHandler := authMiddleware(api.LiveStatusHandler)
	mux.HandleFunc("/cib/live", liveHandler)
	mux.HandleFunc("/cib/live/", func(w http.ResponseWriter, r *http.Request) {
		/*  "/cib/live/" is a special case. It also matches for example /cib/live/resources/types
		 *  but  /cib/live/resources/types should be handled in Ruby instead */
		if r.URL.Path == "/cib/live/" {
			liveHandler(w, r)
			return
		}
		routehandler.ServeHTTP(w, r) // if not exactly "/cib/live/" --> Ruby fallback
	})
	// Register BOTH /cib/live/foo and /cib/live/foo/ to avoid conflicts with Ruby
	mux.HandleFunc("/cib/live/primitives", authMiddleware(api.ResourceEditHandler))
	mux.HandleFunc("/cib/live/primitives/", authMiddleware(api.ResourceEditHandler))
	mux.HandleFunc("/cib/live/clones", authMiddleware(api.CloneEditHandler))
	mux.HandleFunc("/cib/live/clones/", authMiddleware(api.CloneEditHandler))
	mux.HandleFunc("/cib/live/clones/new", authMiddleware(api.CloneNewHandler))
	mux.HandleFunc("/cib/live/clones/new/", authMiddleware(api.CloneNewHandler))
	mux.HandleFunc("/cib/live/nodes", authMiddleware(api.NodesEditHandler))
	mux.HandleFunc("/cib/live/nodes/", authMiddleware(api.NodesEditHandler))
	dashboardStore := internal.NewDashboardConfigStore("")
	mux.HandleFunc("/api/dashboard/clusters", authMiddleware(api.NewDashboardClustersHandler(dashboardStore)))
	mux.HandleFunc("/dashboard", authMiddleware(api.DashboardHandler))
	mux.HandleFunc("/dashboard/", authMiddleware(api.DashboardHandler))

	// resource = primitive or clone
	mux.HandleFunc("/api/cib/primitive/create", authMiddleware(api.PrimitiveCreateHandler))
	mux.HandleFunc("/api/cib/clone/create", authMiddleware(api.CloneCreateHandler))
	mux.HandleFunc("/api/cib/primitive/update", authMiddleware(api.PrimitiveUpdateHandler)) // Can't find where it was used (let's not remove it though)
	mux.HandleFunc("/api/cib/resource/rename", authMiddleware(api.ResourceRenameHandler))
	mux.HandleFunc("/api/cib/resource/delete", authMiddleware(api.ResourceDeleteHandler))
	mux.HandleFunc("/api/cib/resource/stop", authMiddleware(api.ResourceStopHandler))
	mux.HandleFunc("/api/cib/resource/start", authMiddleware(api.ResourceStartHandler))
	mux.HandleFunc("/api/cib/clone/promote", authMiddleware(api.ClonePromoteHandler))
	mux.HandleFunc("/api/cib/clone/demote", authMiddleware(api.CloneDemoteHandler))
	mux.HandleFunc("/api/cib/resource/maintenance-on", authMiddleware(api.ResourceMaintenanceOnHandler))
	mux.HandleFunc("/api/cib/resource/maintenance-off", authMiddleware(api.ResourceMaintenanceOffHandler))
	mux.HandleFunc("/api/cib/resource/migrate", authMiddleware(api.ResourceMigrateHandler))
	mux.HandleFunc("/api/cib/resource/cleanup", authMiddleware(api.ResourceCleanupHandler))
	mux.HandleFunc("/api/cib/resource/clear", authMiddleware(api.ResourceClearHandler))
	mux.HandleFunc("/api/cib/cluster/details/fetch", authMiddleware(api.FetchClusterDetails))
	mux.HandleFunc("/api/cib/cluster/dashboard/fetch", api.DashboardStatusCORS(authMiddleware(api.FetchDashboardHandler)))
	mux.HandleFunc("/api/cib/primitive/classes/fetch", authMiddleware(api.FetchPrimitiveClasses))
	mux.HandleFunc("/api/cib/primitive/providers/fetch", authMiddleware(api.FetchPrimitiveProviders))
	mux.HandleFunc("/api/cib/primitive/types/fetch", authMiddleware(api.FetchPrimitiveTypes))
	mux.HandleFunc("/api/cib/clone/child-resources/fetch", authMiddleware(api.FetchChildResources))
	mux.HandleFunc("/api/cib/resource/params/fetch", authMiddleware(api.FetchResourceParams))
	mux.HandleFunc("/api/cib/resource/params/submit", authMiddleware(api.SubmitResourceParams))
	mux.HandleFunc("/api/cib/resource/meta-attributes/fetch", authMiddleware(api.FetchResourceMetaAttributes))
	mux.HandleFunc("/api/cib/resource/meta-attributes/submit", authMiddleware(api.SubmitResourceMetaAttributes))
	mux.HandleFunc("/api/cib/resource/operations/fetch", authMiddleware(api.FetchResourceOperations))
	mux.HandleFunc("/api/cib/resource/operations/submit", authMiddleware(api.SubmitResourceOperations))
	mux.HandleFunc("/api/cib/resource/operation/attributes/fetch", authMiddleware(api.FetchResourceOperationAttributes))
	mux.HandleFunc("/api/cib/resource/utilizations/fetch", authMiddleware(api.FetchResourceUtilizations))
	mux.HandleFunc("/api/cib/resource/utilizations/submit", authMiddleware(api.SubmitResourceUtilizations))
	mux.HandleFunc("/api/cib/clone/meta-attributes/fetch", authMiddleware(api.FetchCloneMetaAttributes))
	mux.HandleFunc("/api/cib/clone/meta-attributes/submit", authMiddleware(api.SubmitCloneMetaAttributes))
	mux.HandleFunc("/api/cib/node/maintenance-on", authMiddleware(api.NodeMaintenanceOnHandler))
	mux.HandleFunc("/api/cib/node/maintenance-off", authMiddleware(api.NodeMaintenanceOffHandler))
	mux.HandleFunc("/api/cib/node/standby-on", authMiddleware(api.NodeStandbyOnHandler))
	mux.HandleFunc("/api/cib/node/standby-off", authMiddleware(api.NodeStandbyOffHandler))
	mux.HandleFunc("/api/cib/node/fence", authMiddleware(api.NodeFenceHandler))
	mux.HandleFunc("/api/cib/node/clearstate", authMiddleware(api.NodeClearstateHandler))
	mux.HandleFunc("/api/cib/node/attributes/fetch", authMiddleware(api.FetchNodeAttributes))
	mux.HandleFunc("/api/cib/node/attributes/submit", authMiddleware(api.SubmitNodeAttributes))
	mux.HandleFunc("/api/cib/node/utilizations/fetch", authMiddleware(api.FetchNodeUtilizations))
	mux.HandleFunc("/api/cib/node/utilizations/submit", authMiddleware(api.SubmitNodeUtilizations))
	mux.HandleFunc("/api/cib/node/fence/history/fetch", authMiddleware(api.FetchNodeFencingHistory))

	mux.HandleFunc("/api/crm/status/fetch", authMiddleware(api.FetchCrmStatus))
	mux.HandleFunc("/api/cib/resources/fetch", authMiddleware(api.FetchCibResources))

	mux.Handle("/", routehandler) // routehandler is a fallback

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	//TODO: this function should return errors
	// an https server with a reverse proxy. http is redirected to https
	server.ListenAndServeWithRedirect(fmt.Sprintf("%s:%d", config.Listen, config.Port), mux, config.Cert, config.Key)
}
