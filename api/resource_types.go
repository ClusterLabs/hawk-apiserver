package api

import (
	"html/template"
	"net/http"
)

type resourceType struct {
	Name string
	Path string
	Icon string
	Help template.HTML
}

// Help text is copied verbatim from Ruby Hawk config/help/en-US/resources/*.md.
// HTML is trusted static content so paragraphs, examples, and subheadings match Ruby.
func resourceTypes() []resourceType {
	return []resourceType{
		{
			Name: "Primitive",
			Path: "primitives",
			Icon: "file",
			Help: `<p>Primitive resources are the most basic type of resource.</p>
<p>To create a primitive, define an ID and specify some parameters like class, (provider), and type.</p>
<p>An example of a primitive resource is <code>ocf:heartbeat:IPaddr</code>, which can be used to configure a floating IP address for your cluster.</p>`,
		},
		{
			Name: "Group",
			Path: "groups",
			Icon: "users",
			Help: `<p>Resource groups contain a set of resources that need to be located together, be started sequentially and stopped in the reverse order.</p>
<p>To create a group, define an ID and select the resources that will be members of the group.</p>
<p>An example for use of a group is a Web server that requires an IP address and a file system. Configure IP address and file system as separate resources and combine them into a cluster resource group. In case of a software or hardware malfunction, the group will fail over to another cluster node, similar to an individual cluster resource.</p>`,
		},
		{
			Name: "Clone",
			Path: "clones",
			Icon: "copy",
			Help: `<p>Clones are resources that can run simultaneously on multiple nodes in your cluster.</p>
<p>To create a clone, define an ID and select the child resource that you want to use as clone. Any regular resources or resource groups can be cloned. Instances of cloned resources may behave identically. However, they may also be configured differently, depending on which node they are hosted.</p>
<p>An example of a resource that can be configured as clone is <code>ocf:pacemaker:controld</code> for cluster file systems like OCFS2.</p>`,
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
			Help: `<p>Tags are a way to refer to multiple resources at once, without creating any colocation or ordering relationship between them. You can use tags for grouping conceptually related resources. All resources belonging to a tag can then be started or stopped with a single command.</p>
<p>To create a tag, define an ID and select the objects that you want to refer to with the tag.</p>
<p>For example, if you have several resources related to a database, you can add all related resources to a tag named <code>database</code>.</p>`,
		},
		{
			Name: "Alert",
			Path: "alerts",
			Icon: "bullhorn",
			Help: `<p>Alerts may be configured to take some external action when a cluster event occurs (node failure, resource starting or stopping, etc.).</p>
<p>As with resource agents, the cluster calls an external program (an alert agent) to handle alerts. The cluster passes information about the event to the agent via environment variables. Agents can do anything desired with this information (send an e-mail, log to a file, update a monitoring system, etc.).</p>
<p>Multiple alert agents may be configured; the cluster will call all of them for each event.</p>
<p>Alert agents will be called only on cluster nodes. They will be called for events involving Pacemaker Remote nodes, but they will never be called on those nodes.</p>
<h4>Recipients</h4>
<p>Usually alerts are directed towards a recipient. Thus each alert may be additionally configured with one or more recipients. The cluster will call the agent separately for each recipient.</p>
<p>The recipient may be anything the alert agent can recognize — an IP address, an e-mail address, a file name, whatever the particular agent supports.</p>
<h4>Instance Attributes</h4>
<p>Agent-specific configuration values may be configured as instance attributes. These will be passed to the agent as additional environment variables. The number, names and allowed values of these instance attributes are completely up to the particular agent.</p>
<h4>Meta Attributes</h4>
<p>Meta-attributes can be configured for alert agents to affect how Pacemaker calls them.</p>`,
		},
	}
}

func ResourceTypesHandler(w http.ResponseWriter, r *http.Request) {
	base := "/cib/live"
	switch r.URL.Path {
	case "/cib/live/resources/types", "/cib/live/resources/types/":
	case batchBase + "/resources/types", batchBase + "/resources/types/":
		base = batchBase
	default:
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
		"ResourceTypes": resourceTypes(),
		"CIBBase":       base,
	}
	if base == batchBase {
		data["BatchMode"] = true
		data["BatchConfig"] = map[string]string{"base": batchBase, "api": batchAPI, "id": batchID}
	}
	// This flash is only for backward compatibility with Ruby redirects.
	if r.URL.Query().Get("flash") == "deleted" {
		data["AlertType"] = "success"
		data["AlertMessage"] = "Resource deleted successfully"
	}
	renderTemplate(w, "resource_types", data)
}
