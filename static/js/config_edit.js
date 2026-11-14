"use strict";

const configBase = "/cib/live";
const configTables = new Map();
let configAuthenticityToken;

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

// Rename forms continue to use the existing Rails endpoint.
async function openConfigRenameModal(url) {
    try {
        const response = await fetch(url, { headers: { "X-Requested-With": "XMLHttpRequest", Accept: "text/html" }, cache: "no-store" });
        if (!response.ok || response.status === 204) throw new Error("Unable to open this operation.");
        const parsed = new DOMParser().parseFromString(await response.text(), "text/html");
        parsed.querySelectorAll("script").forEach(script => script.remove());
        const content = configElement("div");
        content.append(...parsed.body.childNodes);
        const popup = new Popup();
        popup._appendModalContentChild(content);
        content.querySelectorAll('[data-dismiss="modal"]').forEach(button => button.addEventListener("click", () => popup.close()));
        const form = content.querySelector("form");
        if (!form) { popup.close(); throw new Error("Invalid rename form."); }
        configAuthenticityToken = form.querySelector('[name="authenticity_token"]')?.value || configAuthenticityToken;
        const input = form.querySelector('[name="to"]');
        input.required = true;
        input.focus();
        form.addEventListener("submit", async event => {
            event.preventDefault();
            const submit = form.querySelector('[type="submit"]');
            if (submit.disabled || !form.reportValidity()) return;
            submit.disabled = true;
            try {
                const result = await configJSON(form.action, { method: "POST", body: new URLSearchParams(new FormData(form)) });
                popup.close();
                configNotification(result.message || "Renamed successfully.");
                await refreshConfigTables();
            } catch (error) {
                let notice = content.querySelector(".config-modal-error");
                if (!notice) { notice = configElement("div", "", "alert alert-danger config-modal-error"); form.prepend(notice); }
                notice.textContent = error.message;
                submit.disabled = false;
            }
        });
    } catch (error) { configNotification(error.message, true); }
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
    return { edit: `${base}/edit`, remove: base, rename: `${configBase}/${renameCollection}/${id}/edit_name?source=edit` };
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
        if (operation === "rename") link.addEventListener("click", event => { event.preventDefault(); openConfigRenameModal(link.href); });
        if (operation === "remove") link.addEventListener("click", async event => {
            event.preventDefault();
            if (link.getAttribute("aria-disabled") === "true") return;
            new DeletePopup(row.id, async () => {
                link.setAttribute("aria-disabled", "true");
                try {
                    // Reuse Rails' session token rather than bypassing its CSRF checks.
                    if (!configAuthenticityToken) {
                        const response = await fetch(routes.rename, { headers: { "X-Requested-With": "XMLHttpRequest", Accept: "text/html" }, cache: "no-store" });
                        if (!response.ok) throw new Error("Unable to prepare deletion.");
                        const document = new DOMParser().parseFromString(await response.text(), "text/html");
                        configAuthenticityToken = document.querySelector('[name="authenticity_token"]')?.value;
                        if (!configAuthenticityToken) throw new Error("Unable to retrieve the configuration session token. Reload the page and retry.");
                    }
                    const result = await configJSON(link.href, { method: "POST", body: new URLSearchParams({ _method: "delete", authenticity_token: configAuthenticityToken }) });
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
        new RAInfoPopup(row.id, resourceAgent || agent);
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
pollClusterStatus(refreshConfigTables);
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/config/edit"]').setAttribute("aria-current", "page");
