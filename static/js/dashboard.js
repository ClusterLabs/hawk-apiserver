function insertAllNodesRow(tbody, clusterDetails,  nodes) {
    const trAllNodes = document.createElement("tr");

    tbody.appendChild(trAllNodes);
    trAllNodes.className = "no-top-border table-header nodes-table-row";


    const thLeftBorder = document.createElement("th");
    trAllNodes.appendChild(thLeftBorder);
    thLeftBorder.className = "col-sm-2 no-left-border";
    thLeftBorder.colSpan = 2;

    nodes.forEach((node, index) => {
        const thNode = document.createElement("th");
        trAllNodes.appendChild(thNode);
        thNode.className = "no-left-border";
        thNode.dataset.nodeIndex = index;

        const spanStatusBar = document.createElement("span");
        thNode.appendChild(spanStatusBar);
        spanStatusBar.className = "status-bar";
        if(!node.online || node.standby) {
            spanStatusBar.classList.add("status-offline");
        } else {
            spanStatusBar.classList.add("status-success");
        }
        const divNodeName = document.createElement("div");
        thNode.appendChild(divNodeName);

        divNodeName.className = "node-name";
        divNodeName.title = `Node id: ${node?.id}`;
        divNodeName.innerText = node.name;

        const spanNodeStatus = document.createElement("span");
        divNodeName.appendChild(spanNodeStatus);
        spanNodeStatus.className = "table-cluster-name";
        spanNodeStatus.title = `Status:  ${clusterDetails?.status}
Epoch:  ${clusterDetails?.epoch}
Update Origin:  ${clusterDetails?.updateOrigin}
Update User:  ${clusterDetails?.updateUser}
Stack:  ${clusterDetails?.stack}`;
        spanNodeStatus.innerText = `${clusterDetails?.clusterName} `;

        const iInfoCircle = document.createElement("i");
        spanNodeStatus.appendChild(iInfoCircle);
        iInfoCircle.className = "fa fa-info-circle";
        iInfoCircle.ariaHidden = true;

        const spanStatus = document.createElement("span");
        thNode.appendChild(spanStatus);
        spanStatus.className = "status-icon";

        if(node?.isDC) {
            const iHome = document.createElement("i");
            spanStatus.appendChild(iHome);
            iHome.className = "fa fa-home";
            iHome.ariaHidden = true;
            iHome.title = "Designated coordinator";
            thNode.appendChild(document.createTextNode(" "));
        }

        if (node.maintenance) {
            const iWrench = document.createElement("i");
            spanStatus.appendChild(iWrench);
            iWrench.className = "fa fa-wrench";
            iWrench.ariaHidden = true;
            iWrench.title = "Maintenance mode";
            thNode.appendChild(document.createTextNode(" "));
        }

        if (node.eventsFound?.trim()) {
            const iEventsFound = document.createElement("i");
            thNode.appendChild(iEventsFound);
            iEventsFound.className = "fas fa-refresh text-warning";
            iEventsFound.title = node.eventsFound;
        }
    });
}

function insertResourceRow(tbody, resource) {
    const tr = document.createElement("tr");
    tr.dataset.resourceName = resource.name;
    tbody.appendChild(tr);

    const tdLeftBorder = document.createElement("td");
    tr.appendChild(tdLeftBorder);
    tdLeftBorder.className = "no-left-border resource-icon";
    if (resource.maintenance) {
        const spanLeftStatusIcon = document.createElement("span");
        tdLeftBorder.appendChild(spanLeftStatusIcon);
        spanLeftStatusIcon.className = "status-icon";

        const iWrench = document.createElement("i");
        spanLeftStatusIcon.appendChild(iWrench);
        iWrench.className = "fa fa-wrench";
        iWrench.ariaHidden = true;
    }

    const tdResourceName = document.createElement("td");
    tr.appendChild(tdResourceName);
    tdResourceName.className = "text-left no-left-border";
    const spanStatusBar = document.createElement("span");
    tdResourceName.appendChild(spanStatusBar);
    spanStatusBar.className = `status-bar status-${resource.maintenance ? "success" :  (resource.active ? "success" : "offline" )}`;
    const spanResourceName = document.createElement("span");
    tdResourceName.appendChild(spanResourceName);
    spanResourceName.className = "resource-name";

    spanResourceName.innerText = resource.name;

    for(let i = 0; i < resource.roles.length; i++) {
        const tdCircle = document.createElement("td");
        tdCircle.dataset.nodeIndex = i;
        tr.appendChild(tdCircle);
        if (i == 0) {
            tdCircle.className = "no-left-border";
        }

        const tdDiv = document.createElement("div");
        tdCircle.appendChild(tdDiv);
        tdDiv.className = "node-circle";
        if (resource.roles[i] == RESOURCE_STATUS_STARTED) {
            tdDiv.classList.add("status-success");
            tdDiv.title = "Started";
        } else if (resource.roles[i] == RESOURCE_STATUS_PROMOTED) {
            tdDiv.classList.add("status-success");
            tdDiv.title = "Promoted";
            const spanPromoted = document.createElement("span");
            tdCircle.appendChild(spanPromoted);
            spanPromoted.className = "table-label promoted-unpromoted";
            const iAsterisk = document.createElement("i");
            spanPromoted.appendChild(iAsterisk);
            iAsterisk.className = "fa fa-asterisk";
        } else if (resource.roles[i] == RESOURCE_STATUS_UNPROMOTED) {
            tdDiv.classList.add("status-success");
            tdDiv.title = "Started";
        } else if (resource.roles[i] == RESOURCE_STATUS_STOPPED) {
            tdDiv.classList.add("status-danger");
        } else if (resource.roles[i] == RESOURCE_STATUS_DEFAULT){
            tdDiv.classList.add("status-default");
        } else if (resource.roles[i] == RESOURCE_STATUS_OFFLINE){
            tdDiv.classList.add("status-offline");
            tdDiv.title = "Node offline";
        } else {
            tdDiv.classList.add("status-default");
        }
    }
}


const dashboardViews = new Map();
const dashboardActiveTabKey = "dashboardActiveCluster";

function rememberDashboardTab(key) {
    try {
        sessionStorage.setItem(dashboardActiveTabKey, key);
    } catch (_) {
        // Tabs still work when browser storage is unavailable.
    }
}

function selectDashboardTab(key, remember = true) {
    if (!dashboardViews.has(key)) return;
    dashboardViews.forEach((view, viewKey) => {
        const active = viewKey === key;
        view.tab.classList.toggle("active", active);
        view.panel.classList.toggle("active", active);
        view.tab.querySelector('a[role="tab"]').setAttribute("aria-selected", String(active));
    });
    if (remember) rememberDashboardTab(key);
}

function restoreDashboardTab() {
    let key = "local";
    try {
        key = sessionStorage.getItem(dashboardActiveTabKey) || key;
    } catch (_) {}
    if (!dashboardViews.has(key)) key = "local";
    selectDashboardTab(key);
}

function applyResourceFilter(control) {
    control.closest(".tab-pane").dashboardView.setSearch(control.value);
}

function applyNodeFilter(control) {
    control.closest(".tab-pane").dashboardView.setNodeFilter(control.querySelector("input:checked").value);
}

function addDashboardView(key, config, client) {
    if (dashboardViews.has(key)) return dashboardViews.get(key);
    const view = new DashboardClusterView(config, client);
    dashboardViews.set(key, view);
    view.tab.querySelector('a[role="tab"]').addEventListener("click", event => {
        event.preventDefault();
        selectDashboardTab(key);
    });
    const tabs = document.getElementById("dashboard-tabs-placeholder");
    tabs.insertBefore(view.tab, tabs.querySelector(".dashboard-legend"));
    document.getElementById("dashboard-panels").appendChild(view.panel);
    if (dashboardViews.size === 1) {
        selectDashboardTab(key, false);
    }
    view.start();
    return view;
}

function openAddClusterPopup() {
    new AddClusterPopup(config => {
        const key = `remote:${config.name}`;
        const existing = dashboardViews.get(key);
        if (existing) existing.reconfigure(config, new DashboardClusterClient(config));
        else addDashboardView(key, config, new DashboardClusterClient(config));
    });
}

async function removeDashboardCluster(button) {
    const view = button.closest(".tab-pane").dashboardView;
    if (view.config.local || button.disabled) return;
    button.disabled = true;
    try {
        if (!await new RemoveClusterPopup(view.config.name).result) return;
        const response = await fetch("/api/dashboard/clusters", {
            method: "DELETE",
            headers: { "Content-Type": "application/json", Accept: "application/json" },
            body: JSON.stringify({ name: view.config.name }),
        });
        if (response.redirected || response.status === 401 || response.status === 403) {
            throw new Error("Please sign in again to remove this cluster.");
        }
        if (!response.ok) throw new Error("Unable to remove cluster configuration. Please try again.");
        const wasActive = view.tab.classList.contains("active");
        view.destroy();
        dashboardViews.delete(`remote:${view.config.name}`);
        if (wasActive) {
            selectDashboardTab("local");
        }
        const alert = document.createElement("x-alert");
        alert.id = "status-alert";
        alert.setAttribute("type", "success");
        alert.setAttribute("message", "Cluster removed successfully.");
        alert.setAttribute("visible", "true");
        document.getElementById("status-alert").replaceWith(alert);
    } catch (error) {
        view.showError(error);
    } finally {
        button.disabled = false;
    }
}

async function initializeDashboard() {
    addDashboardView("local", { name: "Local cluster", local: true }, new DashboardClusterClient({ local: true }));
    try {
        const response = await fetch("/api/dashboard/clusters");
        if (!response.ok || response.redirected) throw new Error("Unable to load saved clusters.");
        const clusters = await response.json();
        Object.values(clusters).forEach(config => addDashboardView(`remote:${config.name}`, config, new DashboardClusterClient(config)));
        restoreDashboardTab();
    } catch (error) {
        const message = document.createElement("div");
        message.className = "alert alert-danger";
        message.textContent = error.message;
        document.getElementById("dashboard-panels").before(message);
    }
}

window.addEventListener("pagehide", () => dashboardViews.forEach(view => view.stop()));
window.addEventListener("pageshow", event => {
    if (event.persisted) dashboardViews.forEach(view => view.start());
});
initializeDashboard();
