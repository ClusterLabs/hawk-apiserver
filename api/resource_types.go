package api

import (
	"context"
	"net/http"
	"os/exec"
	"time"
)

type resourceType struct {
	Name string
	Path string
	Icon string
	Help string
}

func resourceTypes(ctx context.Context, crm string) []resourceType {
	types := []resourceType{
		{
			Name: "Primitive",
			Path: "primitives",
			Icon: "file",
			Help: "Primitive resources are the most basic type of resource. Define an ID and " +
				"select the resource agent class, provider and type, then configure its " +
				"parameters. For example, an IP address agent can manage a floating IP address.",
		},
		{
			Name: "Group",
			Path: "groups",
			Icon: "users",
			Help: "Groups contain resources that run together, start in order and stop in reverse " +
				"order. Define an ID and select the member resources. For example, group an IP " +
				"address, file system and Web server so they fail over together.",
		},
		{
			Name: "Clone",
			Path: "clones",
			Icon: "copy",
			Help: "Clones run instances of a resource on multiple cluster nodes. Define an ID and " +
				"select a primitive or group as the child resource. Instances may behave " +
				"identically or use node-specific configuration.",
		},
		/* Multi-state is deprecated
		{
			Name: "Multi-state",
			Path: "masters",
			Icon: "superscript",
			Help: "Promotable resources are specialized clones whose instances can operate in two " +
				"modes. A subset of instances can be promoted. Define an ID and select the " +
				"child resource. DRBD is an example of a promotable resource.",
		},
		*/
		{
			Name: "Tag",
			Path: "tags",
			Icon: "tag",
			Help: "Tags refer to multiple resources without adding colocation or ordering " +
				"relationships. Define an ID and select the objects to reference. A tag can be " +
				"used to start or stop related resources with a single command.",
		},
	}
	// Match Rails' Util.has_feature?(:rsc_template), with a bounded probe.
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if exec.CommandContext(ctx, crm, "configure", "help", "rsc_template").Run() == nil {
		types = append(types, resourceType{
			Name: "Template",
			Path: "templates",
			Icon: "file-lines",
			Help: "Resource templates provide reusable configuration for similar resources. " +
				"Primitives referencing a template inherit its operations, parameters, meta " +
				"attributes and utilization attributes. Values defined on the primitive " +
				"override inherited values.",
		})
	}
	return append(types, resourceType{
		Name: "Alert",
		Path: "alerts",
		Icon: "bullhorn",
		Help: "Alerts run external agents when cluster events occur, such as node failures or " +
			"resources starting and stopping. Configure the agent and its recipients, " +
			"parameters and meta attributes. Recipients can be addresses, file names or " +
			"other values understood by the agent.",
	})
}

func ResourceTypesHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/cib/live/resources/types" && r.URL.Path != "/cib/live/resources/types/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data := map[string]any{
		"Title":         "Add Resource",
		"ResourceTypes": resourceTypes(r.Context(), "/usr/sbin/crm"),
	}
	// This flash is only for backward compatibility with Ruby redirects.
	if r.URL.Query().Get("flash") == "deleted" {
		data["AlertType"] = "success"
		data["AlertMessage"] = "Resource deleted successfully"
	}
	renderTemplate(w, "resource_types", data)
}
