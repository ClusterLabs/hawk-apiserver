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

	reportsAPI := api.NewReportsAPI("")
	mux.HandleFunc("/reports", authMiddleware(api.ReportsHandler))
	mux.HandleFunc("/reports/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/reports/" {
			authMiddleware(api.ReportsHandler)(w, r)
			return
		}
		// The transition explorer remains in Rails during the landing page port.
		routehandler.ServeHTTP(w, r)
	})
	mux.HandleFunc("/api/reports", authMiddleware(reportsAPI.ServeHTTP))
	mux.HandleFunc("/api/reports/", authMiddleware(reportsAPI.ServeHTTP))

	liveHandler := authMiddleware(api.LiveStatusHandler)
	mux.HandleFunc("/cib/live", liveHandler)
	mux.HandleFunc("/cib/live/", func(w http.ResponseWriter, r *http.Request) {
		// Unported pages beneath /cib/live/ still use the Ruby fallback.
		if r.URL.Path == "/cib/live/" {
			liveHandler(w, r)
			return
		}
		routehandler.ServeHTTP(w, r) // if not exactly "/cib/live/" --> Ruby fallback
	})
	// Register BOTH /cib/live/foo and /cib/live/foo/ to avoid conflicts with Ruby
	mux.HandleFunc("/cib/live/wizards/cryptctl", authMiddleware(api.CryptctlWizardHandler))
	mux.HandleFunc("/cib/live/wizards/cryptctl/", authMiddleware(api.CryptctlWizardHandler))
	mux.HandleFunc("/cib/live/wizards/sbd", authMiddleware(api.SBDWizardHandler))
	mux.HandleFunc("/cib/live/wizards/sbd/", authMiddleware(api.SBDWizardHandler))
	mux.HandleFunc("/cib/live/wizards/nginx", authMiddleware(api.NginxWizardHandler))
	mux.HandleFunc("/cib/live/wizards/nginx/", authMiddleware(api.NginxWizardHandler))
	mux.HandleFunc("/cib/live/wizards/haproxy", authMiddleware(api.HAProxyWizardHandler))
	mux.HandleFunc("/cib/live/wizards/haproxy/", authMiddleware(api.HAProxyWizardHandler))
	mux.HandleFunc("/cib/live/wizards/apache", authMiddleware(api.ApacheWizardHandler))
	mux.HandleFunc("/cib/live/wizards/apache/", authMiddleware(api.ApacheWizardHandler))
	mux.HandleFunc("/cib/live/wizards/sap-simple-stack-plus", authMiddleware(api.SAPSimpleStackPlusWizardHandler))
	mux.HandleFunc("/cib/live/wizards/sap-simple-stack-plus/", authMiddleware(api.SAPSimpleStackPlusWizardHandler))
	mux.HandleFunc("/cib/live/wizards/sap-simple-stack", authMiddleware(api.SAPSimpleStackWizardHandler))
	mux.HandleFunc("/cib/live/wizards/sap-simple-stack/", authMiddleware(api.SAPSimpleStackWizardHandler))
	mux.HandleFunc("/cib/live/wizards/sap-db", authMiddleware(api.SAPDBWizardHandler))
	mux.HandleFunc("/cib/live/wizards/sap-db/", authMiddleware(api.SAPDBWizardHandler))
	mux.HandleFunc("/cib/live/wizards/sap-ci", authMiddleware(api.SAPCIWizardHandler))
	mux.HandleFunc("/cib/live/wizards/sap-ci/", authMiddleware(api.SAPCIWizardHandler))
	mux.HandleFunc("/cib/live/wizards/sap-as", authMiddleware(api.SAPASWizardHandler))
	mux.HandleFunc("/cib/live/wizards/sap-as/", authMiddleware(api.SAPASWizardHandler))
	mux.HandleFunc("/cib/live/wizards/nfsserver-lvm-drbd", authMiddleware(api.NFSServerLVMDRBDWizardHandler))
	mux.HandleFunc("/cib/live/wizards/nfsserver-lvm-drbd/", authMiddleware(api.NFSServerLVMDRBDWizardHandler))
	mux.HandleFunc("/cib/live/wizards/nfsserver", authMiddleware(api.NFSServerWizardHandler))
	mux.HandleFunc("/cib/live/wizards/nfsserver/", authMiddleware(api.NFSServerWizardHandler))
	mux.HandleFunc("/cib/live/wizards/exportfs", authMiddleware(api.ExportFSWizardHandler))
	mux.HandleFunc("/cib/live/wizards/exportfs/", authMiddleware(api.ExportFSWizardHandler))
	mux.HandleFunc("/cib/live/wizards/raid-lvm", authMiddleware(api.RAIDLVMWizardHandler))
	mux.HandleFunc("/cib/live/wizards/raid-lvm/", authMiddleware(api.RAIDLVMWizardHandler))
	mux.HandleFunc("/cib/live/wizards/lvm-drbd", authMiddleware(api.LVMDRBDWizardHandler))
	mux.HandleFunc("/cib/live/wizards/lvm-drbd/", authMiddleware(api.LVMDRBDWizardHandler))
	mux.HandleFunc("/cib/live/wizards/gfs2", authMiddleware(api.GFS2WizardHandler))
	mux.HandleFunc("/cib/live/wizards/gfs2/", authMiddleware(api.GFS2WizardHandler))
	mux.HandleFunc("/cib/live/wizards/filesystem", authMiddleware(api.FilesystemWizardHandler))
	mux.HandleFunc("/cib/live/wizards/filesystem/", authMiddleware(api.FilesystemWizardHandler))
	mux.HandleFunc("/cib/live/wizards/drbd", authMiddleware(api.DRBDWizardHandler))
	mux.HandleFunc("/cib/live/wizards/drbd/", authMiddleware(api.DRBDWizardHandler))
	mux.HandleFunc("/cib/live/wizards/oracle", authMiddleware(api.OracleWizardHandler))
	mux.HandleFunc("/cib/live/wizards/oracle/", authMiddleware(api.OracleWizardHandler))
	mux.HandleFunc("/cib/live/wizards/db2-hadr", authMiddleware(api.DB2HADRWizardHandler))
	mux.HandleFunc("/cib/live/wizards/db2-hadr/", authMiddleware(api.DB2HADRWizardHandler))
	mux.HandleFunc("/cib/live/wizards/db2", authMiddleware(api.DB2WizardHandler))
	mux.HandleFunc("/cib/live/wizards/db2/", authMiddleware(api.DB2WizardHandler))
	mux.HandleFunc("/cib/live/wizards/database", authMiddleware(api.DatabaseWizardHandler))
	mux.HandleFunc("/cib/live/wizards/database/", authMiddleware(api.DatabaseWizardHandler))
	mux.HandleFunc("/cib/live/wizards/virtual-ip", authMiddleware(api.VirtualIPWizardHandler))
	mux.HandleFunc("/cib/live/wizards/virtual-ip/", authMiddleware(api.VirtualIPWizardHandler))
	mux.HandleFunc("/cib/live/wizards/mailto", authMiddleware(api.MailtoWizardHandler))
	mux.HandleFunc("/cib/live/wizards/mailto/", authMiddleware(api.MailtoWizardHandler))
	mux.HandleFunc("/cib/live/wizards/health", authMiddleware(api.HealthWizardHandler))
	mux.HandleFunc("/cib/live/wizards/health/", authMiddleware(api.HealthWizardHandler))
	mux.HandleFunc("/cib/live/wizards", authMiddleware(api.WizardsHandler))
	mux.HandleFunc("/cib/live/wizards/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cib/live/wizards/" {
			authMiddleware(api.WizardsHandler)(w, r)
			return
		}
		routehandler.ServeHTTP(w, r)
	})
	mux.HandleFunc("/cib/live/commands", authMiddleware(api.CommandsHandler))
	mux.HandleFunc("/cib/live/commands/", authMiddleware(api.CommandsHandler))
	mux.HandleFunc("/cib/live/orders/new", authMiddleware(api.OrderNewHandler))
	mux.HandleFunc("/cib/live/orders/new/", authMiddleware(api.OrderNewHandler))
	mux.HandleFunc("/cib/live/orders/", authMiddleware(api.OrderRubyHandler))
	mux.HandleFunc("/api/cib/order/options/fetch", authMiddleware(api.FetchColocationOptions))
	mux.HandleFunc("/api/cib/order/create", authMiddleware(api.OrderCreateHandler))
	mux.HandleFunc("/cib/live/colocations/new", authMiddleware(api.ColocationNewHandler))
	mux.HandleFunc("/cib/live/colocations/new/", authMiddleware(api.ColocationNewHandler))
	mux.HandleFunc("/cib/live/colocations/", authMiddleware(api.ColocationRubyHandler))
	mux.HandleFunc("/api/cib/colocation/options/fetch", authMiddleware(api.FetchColocationOptions))
	mux.HandleFunc("/api/cib/colocation/fetch", authMiddleware(api.FetchColocation))
	mux.HandleFunc("/api/cib/colocation/update", authMiddleware(api.ColocationUpdateHandler))
	mux.HandleFunc("/api/cib/colocation/create", authMiddleware(api.ColocationCreateHandler))
	mux.HandleFunc("/cib/live/locations/new", authMiddleware(api.LocationNewHandler))
	mux.HandleFunc("/cib/live/locations/new/", authMiddleware(api.LocationNewHandler))
	mux.HandleFunc("/cib/live/locations/", authMiddleware(api.LocationRubyHandler))
	mux.HandleFunc("/api/cib/location/options/fetch", authMiddleware(api.FetchLocationOptions))
	mux.HandleFunc("/api/cib/location/fetch", authMiddleware(api.FetchLocation))
	mux.HandleFunc("/api/cib/location/update", authMiddleware(api.LocationUpdateHandler))
	mux.HandleFunc("/api/cib/location/create", authMiddleware(api.LocationCreateHandler))
	mux.HandleFunc("/cib/live/constraints/types", authMiddleware(api.ConstraintTypesHandler))
	mux.HandleFunc("/cib/live/constraints/types/", authMiddleware(api.ConstraintTypesHandler))
	mux.HandleFunc("/cib/live/resources/types", authMiddleware(api.ResourceTypesHandler))
	mux.HandleFunc("/cib/live/resources/types/", authMiddleware(api.ResourceTypesHandler))
	mux.HandleFunc("/cib/live/primitives/new", authMiddleware(api.PrimitiveNewHandler))
	mux.HandleFunc("/cib/live/primitives/new/", authMiddleware(api.PrimitiveNewHandler))
	mux.HandleFunc("/cib/live/alerts/new", authMiddleware(api.AlertNewHandler))
	mux.HandleFunc("/cib/live/alerts/new/", authMiddleware(api.AlertNewHandler))
	mux.HandleFunc("/cib/live/alerts/", authMiddleware(api.AlertRubyHandler))
	mux.HandleFunc("/api/cib/alert/create", authMiddleware(api.AlertCreateHandler))
	mux.HandleFunc("/api/cib/alert/meta-attributes/fetch", authMiddleware(api.FetchAlertMetaAttributes))
	mux.HandleFunc("/cib/live/tags/new", authMiddleware(api.TagNewHandler))
	mux.HandleFunc("/cib/live/tags/new/", authMiddleware(api.TagNewHandler))
	mux.HandleFunc("/cib/live/tags/", authMiddleware(api.TagRubyHandler))
	mux.HandleFunc("/api/cib/tag/create", authMiddleware(api.TagCreateHandler))
	mux.HandleFunc("/api/cib/tag/objects/fetch", authMiddleware(api.FetchTagObjects))
	mux.HandleFunc("/cib/live/groups/new", authMiddleware(api.GroupNewHandler))
	mux.HandleFunc("/cib/live/groups/new/", authMiddleware(api.GroupNewHandler))
	mux.HandleFunc("/cib/live/groups/", authMiddleware(api.GroupRubyHandler))
	mux.HandleFunc("/cib/live/primitives", authMiddleware(api.ResourceEditHandler))
	mux.HandleFunc("/cib/live/primitives/", authMiddleware(api.ResourceEditHandler))
	mux.HandleFunc("/cib/live/clones", authMiddleware(api.CloneEditHandler))
	mux.HandleFunc("/cib/live/clones/", authMiddleware(api.CloneEditHandler))
	mux.HandleFunc("/cib/live/clones/new", authMiddleware(api.CloneNewHandler))
	mux.HandleFunc("/cib/live/clones/new/", authMiddleware(api.CloneNewHandler))
	mux.HandleFunc("/cib/live/nodes", authMiddleware(api.NodesEditHandler))
	mux.HandleFunc("/cib/live/nodes/", authMiddleware(api.NodesEditHandler))
	mux.HandleFunc("/cib/live/config/edit", authMiddleware(api.ConfigEditHandler))
	mux.HandleFunc("/cib/live/config/edit/", authMiddleware(api.ConfigEditHandler))

	mux.HandleFunc("/cib/live/crm_config/edit", authMiddleware(api.ClusterConfigHandler))
	mux.HandleFunc("/cib/live/crm_config/edit/", authMiddleware(api.ClusterConfigHandler))
	mux.HandleFunc("/api/cib/cluster/config/options", authMiddleware(api.ClusterConfigOptionsHandler))
	mux.HandleFunc("/api/cib/cluster/config/update", authMiddleware(api.ClusterConfigUpdateHandler))

	mux.HandleFunc("/cib/live/users", authMiddleware(api.UsersHandler))
	mux.HandleFunc("/cib/live/users/", authMiddleware(api.UsersHandler))
	mux.HandleFunc("/api/cib/acl/users", authMiddleware(api.UsersListHandler))
	mux.HandleFunc("/api/cib/acl/users/delete", authMiddleware(api.UserDeleteHandler))
	mux.HandleFunc("/api/cib/acl/users/create", authMiddleware(api.UserCreateHandler))
	mux.HandleFunc("/cib/live/roles", authMiddleware(api.RolesHandler))
	mux.HandleFunc("/cib/live/roles/", authMiddleware(api.RolesHandler))
	mux.HandleFunc("/api/cib/acl/roles", authMiddleware(api.RolesListHandler))
	mux.HandleFunc("/api/cib/acl/roles/delete", authMiddleware(api.RoleDeleteHandler))
	mux.HandleFunc("/api/cib/acl/role", authMiddleware(api.RoleEditHandler))

	dashboardStore := internal.NewDashboardConfigStore("")
	mux.HandleFunc("/api/dashboard/clusters", authMiddleware(api.NewDashboardClustersHandler(dashboardStore)))
	mux.HandleFunc("/dashboard", authMiddleware(api.DashboardHandler))
	mux.HandleFunc("/dashboard/", authMiddleware(api.DashboardHandler))

	// resource = primitive or clone
	mux.HandleFunc("/api/cib/primitive/create", authMiddleware(api.PrimitiveCreateHandler))
	mux.HandleFunc("/api/cib/group/create", authMiddleware(api.GroupCreateHandler))
	mux.HandleFunc("/api/cib/group/children/fetch", authMiddleware(api.FetchGroupChildren))
	mux.HandleFunc("/api/cib/group/meta-attributes/fetch", authMiddleware(api.FetchGroupMetaAttributes))
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
