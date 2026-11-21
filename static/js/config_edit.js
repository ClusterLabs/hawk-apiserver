"use strict";

const configBase = statusCIBBase();
const configTables = new Map();
const configAPI = configBase === "/cib/live" ? "/api/cib/config" : `/api/batch/${configBase.split("/")[2]}/config`;

function configNotification(message, danger = false) {
    const alert = document.createElement("x-alert");
    alert.setAttribute("type", danger ? "danger" : "success");
    alert.setAttribute("message", message);
    alert.setAttribute("visible", "true");
    document.getElementById("config-notifications").replaceChildren(alert);
}

const configFlash = sessionStorage.getItem("configFlash");
if (configFlash) {
    sessionStorage.removeItem("configFlash");
    configNotification(configFlash);
}

function openConfigRenameModal(url, kind, row) {
    const label = { resources: "Resource", constraints: "Constraint", tags: "Tag", alerts: "Alert" }[kind];
    new RenamePopup(row.id, url, (result, error) => {
        configNotification(error ? error.message : result.message || "Renamed successfully.", Boolean(error));
        if (!error) refreshConfigTables();
    }, label);
}

function configObjectRoutes(kind, row) {
    const id = encodeURIComponent(row.id);
    if (kind === "nodes") return { edit: `${configBase}/nodes/${id}/edit` };
    let collection = kind;
    if (kind === "resources") {
        collection = { primitive: "primitives", group: "groups", clone: "clones", master: "masters", tag: "tags", template: "templates" }[row.object_type] || "resources";
    } else if (kind === "constraints") {
        collection = { location: "locations", colocation: "colocations", order: "orders", ticket: "tickets" }[row.object_type] || "constraints";
    }
    const base = `${configBase}/${collection}/${id}`;
    const renameCollection = kind === "constraints" ? "constraints" : "resources";
    return { edit: row.editable === false ? null : `${base}/edit`, remove: base, rename: `${configBase}/${renameCollection}/${id}/edit_name?source=edit` };
}

function configOperations(kind, row) {
    const routes = configObjectRoutes(kind, row);
    const group = configElement("div", undefined, "btn-group");
    group.setAttribute("role", "group");
    for (const [operation, title, icon] of [["edit", "Edit", "pencil"], ["rename", "Rename", "font"], ["remove", "Delete", "trash"]]) {
        if (!routes[operation]) continue;
        const link = configElement("a", undefined, `${operation === "remove" ? "delete" : operation} btn btn-default btn-xs`);
        link.href = routes[operation];
        link.title = title;
        link.setAttribute("aria-label", `${title} ${row.id || row.name}`);
        link.append(configElement("i", undefined, `fa fa-${icon}`));
        if (operation === "rename") link.addEventListener("click", event => { event.preventDefault(); openConfigRenameModal(`${configAPI}/rename?kind=${kind}`, kind, row); });
        if (operation === "remove") link.addEventListener("click", async event => {
            event.preventDefault();
            if (link.getAttribute("aria-disabled") === "true") return;
            new DeletePopup(row.id, async () => {
                link.setAttribute("aria-disabled", "true");
                try {
                    const result = await configJSON(`${configAPI}/delete?kind=${kind}`, {
                        method: "POST",
                        headers: { "Content-Type": "application/json" },
                        body: JSON.stringify({ id: row.id }),
                    });
                    configNotification(result.message || "Deleted successfully.");
                    await refreshConfigTables();
                } catch (error) { configNotification(error.message, true); }
                finally { link.removeAttribute("aria-disabled"); }
            });
        });
        group.append(link);
    }
    return group;
}

function constraintResourceList(row) {
    const values = [];
    function visit(object) {
        if (typeof object === "string") values.push(object);
        else if (Array.isArray(object)) object.forEach(visit);
        else if (object && typeof object === "object") {
            if (object.resources) visit(object.resources);
            if (object.resource) visit(object.resource);
        }
    }
    visit(row);
    return [...values.slice(0, 8), ...(values.length > 8 ? ["..."] : [])].join(", ");
}

function configResourceType(row) {
    const types = { group: "Group", master: "Multi-state", clone: "Clone", tag: "Tag", bundle: "Bundle" };
    if (types[row.object_type]) return types[row.object_type];
    const agent = row.template ? `@${row.template}` : [row.clazz, row.provider, row.type].filter(Boolean).join(":");
    if (!agent) return row.sort_type || row.type || "";
    const link = configElement("a", row.object_type === "template" ? `Template (${agent})` : agent);
    link.href = `${configBase}/agents/${encodeURIComponent(agent)}`;
    link.addEventListener("click", event => {
        event.preventDefault();
        const metadataRow = row.template ? configTables.get("resources")?.rows.find(item => item.id === row.template) : row;
        const resourceAgent = metadataRow ? [metadataRow.clazz, metadataRow.provider, metadataRow.type].filter(Boolean).join(":") : agent;
        new RAInfoPopup(window.hawkBatch ? "" : row.id, resourceAgent || agent);
    });
    return link;
}

const configColumns = {
    resources: [["id", "Name"], ["sort_type", "Type", configResourceType], [null, "Children", row => row.child || (row.children || row.refs || []).join(", ")]],
    constraints: [["id", "Name"], ["object_type", "Type", row => ({ location: "Location", colocation: "Colocation", order: "Order", ticket: "Ticket" }[row.object_type] || row.object_type)], [null, "Resources", constraintResourceList]],
    nodes: [["name", "Name"], ["id", "ID"]],
    tags: [["id", "Name"], [null, "Objects", row => (row.refs || []).join(", ")]],
    alerts: [["id", "Name"]],
    fencing: [[null, "Target", row => {
        if (row.type === "pattern") return configElement("code", row.target);
        return row.type === "node" ? row.target : `${row.target}=${row.value}`;
    }], [null, "Index", row => row.index], [null, "Devices", row => (row.devices || []).join(",")]],
};

function selectConfigTab(name) {
    if (!configTables.has(name)) name = "resources";
    document.querySelectorAll("[data-config-tab]").forEach(link => {
        const selected = link.dataset.configTab === name;
        link.setAttribute("aria-selected", String(selected));
        link.parentElement.classList.toggle("active", selected);
    });
    for (const [kind, table] of configTables) {
        table.table.parentElement.hidden = kind !== name;
        table.table.parentElement.classList.toggle("active", kind === name);
    }
}

function refreshConfigTables() {
    return Promise.all(Array.from(configTables.values(), table => table.load()));
}

for (const table of document.querySelectorAll("[data-config-table]")) {
    const kind = table.dataset.configTable;
    const controller = new ConfigTable(table, {
        columns: configColumns[kind],
        operations: row => configOperations(kind, row),
        url: `${configBase}/${kind}.json${kind === "nodes" ? "?remote=false" : ""}`,
    });
    configTables.set(controller.kind, controller);
}
document.querySelectorAll("[data-config-tab]").forEach(link => link.addEventListener("click", event => {
    event.preventDefault();
    window.location.hash = link.dataset.configTab;
    selectConfigTab(link.dataset.configTab);
}));
window.addEventListener("hashchange", () => selectConfigTab(window.location.hash.slice(1)));
selectConfigTab(window.location.hash.slice(1));
refreshConfigTables();

// Use the same monitor-driven refresh loop as the live status page.
if (window.hawkBatch) window.hawkBatch.start();
else pollClusterStatus(refreshConfigTables);
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector(`a[href="${configBase}/config/edit"]`).setAttribute("aria-current", "page");

// BatchPage calls the same refresh hook as the status page after simulations
// and while polling the shadow CIB.
function updateClusterStatus() {
    return refreshConfigTables();
}
