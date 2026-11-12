"use strict";

const configBase = "/cib/live";
const configTables = new Map();
let configAuthenticityToken;

function configElement(tag, text, className) {
    const element = document.createElement(tag);
    if (text !== undefined) element.textContent = text;
    if (className) element.className = className;
    return element;
}

function configNotification(message, danger = false) {
    const alert = document.createElement("x-alert");
    alert.setAttribute("type", danger ? "danger" : "success");
    alert.setAttribute("message", message);
    alert.setAttribute("visible", "true");
    document.getElementById("config-notifications").replaceChildren(alert);
}

async function configJSON(url, options = {}) {
    const response = await fetch(url, { cache: "no-store", ...options, headers: { Accept: "application/json", "X-Requested-With": "XMLHttpRequest", ...options.headers } });
    let data;
    try { data = await response.json(); } catch { throw new Error(`Request failed (${response.status}).`); }
    if (!response.ok || data?.error || data?.success === false) throw new Error(data.error || data.message || `Request failed (${response.status}).`);
    return data;
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

class ConfigTable {
    constructor(table) {
        this.table = table;
        this.kind = table.dataset.configTable;
        this.rows = [];
        this.query = "";
        this.page = 1;
        this.size = 25;
        this.sort = this.kind === "nodes" ? "name" : "id";
        this.descending = false;
        this.sequence = 0;
        this.error = table.parentElement.querySelector(".config-table-error");
        this.columns = [...configColumns[this.kind]];
        if (this.kind !== "fencing") {
            this.columns.push([null, "Operations", row => configOperations(this.kind, row)]);
            this.createControls();
        }
        this.head = table.createTHead();
        this.body = table.createTBody();
        this.render();
    }

    createControls() {
        const toolbar = configElement("div", undefined, "config-table-toolbar fixed-table-toolbar");
        const search = configElement("input", undefined, "form-control");
        search.type = "search";
        search.placeholder = "Search";
        search.setAttribute("aria-label", `Search ${this.kind}`);
        search.addEventListener("input", () => { this.query = search.value.toLowerCase(); this.page = 1; this.render(); });
        const refresh = configElement("button", undefined, "btn btn-default");
        refresh.type = "button";
        refresh.title = "Refresh";
        refresh.setAttribute("aria-label", `Refresh ${this.kind}`);
        refresh.append(configElement("i", undefined, "fa fa-refresh"));
        refresh.addEventListener("click", () => this.load());
        toolbar.append(search, refresh);
        this.table.before(toolbar);
        this.pagination = configElement("div", undefined, "config-table-pagination fixed-table-pagination");
        this.table.after(this.pagination);
    }

    async load() {
        const sequence = ++this.sequence;
        this.table.setAttribute("aria-busy", "true");
        try {
            const query = this.kind === "nodes" ? "?remote=false" : "";
            const data = await configJSON(`${configBase}/${this.kind}.json${query}`);
            if (!Array.isArray(data)) throw new Error("Invalid table response.");
            if (sequence !== this.sequence) return;
            this.rows = data;
            this.error.hidden = true;
            this.render();
        } catch (error) {
            if (sequence !== this.sequence) return;
            this.error.textContent = `Unable to load ${this.kind}. ${error.message}`;
            this.error.hidden = false;
        } finally {
            if (sequence === this.sequence) this.table.setAttribute("aria-busy", "false");
        }
    }

    cellValue(column, row) {
        return column[2] ? column[2](row) : row[column[0]] ?? "";
    }

    render() {
        const header = this.head.insertRow();
        this.head.replaceChildren(header);
        for (const column of this.columns) {
            const cell = configElement("th");
            cell.scope = "col";
            if (this.kind !== "fencing") {
                if (column[1] === "Name") cell.className = "col-sm-4";
                else if (column[1] === "Type" || column[1] === "Operations") cell.className = "col-sm-2";
            }
            if (column[0]) {
                const button = configElement("button", column[1]);
                button.type = "button";
                button.append(configElement("span", undefined, `fa fa-${this.sort === column[0] ? (this.descending ? "sort-desc" : "sort-asc") : "sort"} config-sort`));
                cell.setAttribute("aria-sort", this.sort === column[0] ? (this.descending ? "descending" : "ascending") : "none");
                button.addEventListener("click", () => {
                    this.descending = this.sort === column[0] ? !this.descending : false;
                    this.sort = column[0];
                    this.render();
                });
                cell.append(button);
            } else cell.textContent = column[1];
            header.append(cell);
        }
        let rows = this.rows;
        if (this.kind !== "fencing") {
            rows = rows.filter(row => this.columns.slice(0, -1).some(column => {
                const value = this.cellValue(column, row);
                return String(value instanceof Node ? value.textContent : value).toLowerCase().includes(this.query);
            })).slice().sort((a, b) => String(a[this.sort] ?? "").localeCompare(String(b[this.sort] ?? "")) * (this.descending ? -1 : 1));
        }
        const total = rows.length;
        const pages = Math.max(1, Math.ceil(total / this.size));
        this.page = Math.min(this.page, pages);
        if (this.pagination) rows = rows.slice((this.page - 1) * this.size, this.page * this.size);
        this.body.replaceChildren();
        for (const row of rows) {
            const tr = this.body.insertRow();
            for (const column of this.columns) {
                const td = tr.insertCell();
                const value = this.cellValue(column, row);
                if (value instanceof Node) td.append(value);
                else td.textContent = value;
                if (column[1] === "Operations") td.className = "config-operations";
            }
        }
        if (!rows.length) {
            const td = this.body.insertRow().insertCell();
            td.colSpan = this.columns.length;
            td.className = "text-center";
            td.textContent = "No matching records found";
        }
        if (this.pagination) this.renderPagination(total, pages);
    }

    renderPagination(total, pages) {
        const info = configElement("div", `Showing ${total ? (this.page - 1) * this.size + 1 : 0} to ${Math.min(this.page * this.size, total)} of ${total} rows `);
        const sizes = configElement("select", undefined, "form-control input-sm");
        sizes.setAttribute("aria-label", `Rows per page for ${this.kind}`);
        for (const size of [10, 25, 50, 100, 200]) {
            const option = configElement("option", String(size));
            option.value = size;
            option.selected = size === this.size;
            sizes.append(option);
        }
        sizes.addEventListener("change", () => { this.size = Number(sizes.value); this.page = 1; this.render(); });
        info.append(sizes, " records per page");
        const nav = configElement("ul", undefined, "pagination pagination-sm");
        const first = Math.max(1, Math.min(this.page - 2, pages - 4));
        const entries = [["«", 1], ["‹", this.page - 1], ...Array.from({ length: Math.min(5, pages) }, (_, n) => [String(first + n), first + n]), ["›", this.page + 1], ["»", pages]];
        for (const [label, page] of entries) {
            const current = label === String(page) && page === this.page;
            const disabled = page < 1 || page > pages || ((label === "«" || label === "»") && page === this.page);
            const li = configElement("li", undefined, current ? "active" : "");
            const link = configElement("a", label);
            link.href = "#";
            if (current) link.setAttribute("aria-current", "page");
            if (disabled) { li.classList.add("disabled"); link.setAttribute("aria-disabled", "true"); }
            link.addEventListener("click", event => { event.preventDefault(); if (!disabled) { this.page = page; this.render(); } });
            li.append(link);
            nav.append(li);
        }
        this.pagination.replaceChildren(info, nav);
    }
}

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
    const controller = new ConfigTable(table);
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
