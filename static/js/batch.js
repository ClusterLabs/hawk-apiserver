function batchElement(tag, className = "", text = "") {
    const element = document.createElement(tag);
    element.className = className;
    element.textContent = text;
    return element;
}

class BatchPopup extends Popup {
    constructor(title, confirm = null, label = "OK") {
        super();
        const header = batchElement("div", "modal-header");
        const close = batchElement("button", "close", "×");
        close.type = "button";
        close.setAttribute("aria-label", "Close");
        close.onclick = () => this.close();
        this.header = header;
        this.title = batchElement("h3", "modal-title", title);
        header.append(close, this.title);
        this._appendModalContentChild(header);
        header.closest(".modal").classList.add("batch-dialog");
        this.body = batchElement("div", "modal-body");
        this._appendModalContentChild(this.body);
        const footer = batchElement("div", "modal-footer");
        const cancel = batchElement("button", "btn btn-default cancel", confirm ? "Cancel" : "Close");
        cancel.type = "button";
        cancel.onclick = () => this.close();
        this.footer = footer;
        this.cancel = cancel;
        footer.append(cancel);
        if (confirm) {
            this.ok = batchElement("button", "btn btn-primary commit", label);
            this.ok.type = "button";
            this.ok.onclick = async () => {
                if (this.ok.disabled) return;
                this.ok.disabled = true;
                try { await confirm(this); }
                catch (error) { this.error(error); }
                finally { this.ok.disabled = false; }
            };
            footer.append(this.ok);
        }
        this._appendModalContentChild(footer);
    }
    error(error) {
        this.body.querySelector(".batch-error")?.remove();
        this.body.prepend(batchElement("div", "alert alert-danger batch-error", error.message));
    }
}

class BatchShowPopup extends BatchPopup {
    constructor() {
        super("Batch Mode");
        this.header.closest(".modal").classList.add("batch-show-dialog");
        this.header.querySelector(".close").remove();
        this.generation = 0;
        this.main = batchElement("div", "container-fluid batch-view");
        const toolbar = batchElement("div", "btn-toolbar");
        toolbar.setAttribute("role", "toolbar");
        this.dropdown(toolbar, "Simulator", "rocket", [
            ["Summary", () => this.showReport("info", "Summary")],
            ["CIB (in)", () => this.showReport("in", "CIB (in)")],
            ["CIB (out)", () => this.showReport("out", "CIB (out)")],
            ["Transition Graph", () => this.showReport("graph", "Transition Graph")],
            ["Transition", () => this.showReport("graph-xml", "Transition (XML)")],
        ]);
        const choices = [];
        if (hawkBatch.state?.nodes?.length) choices.push(["Node Event", () => this.inject("node")]);
        if (hawkBatch.state?.resources?.length) choices.push(["Resource Event", () => this.inject("resource")]);
        if (hawkBatch.state?.tickets?.length) choices.push(["Ticket Event", () => this.inject("ticket")]);
        this.dropdown(toolbar, "Inject", "bolt", choices);
        this.row(toolbar);
        this.row(batchElement("hr"));
        this.events = batchElement("ul", "list-group batch-events");
        this.row(this.events, "col-sm-9");
        this.drawEvents();
        this.diffBox = batchElement("div", "batch-report batch-diff-data");
        this.row(this.diffBox);
        this.reportBox = batchElement("div", "batch-report batch-report-view");
        this.body.append(this.main, this.reportBox);
        this.showMain();
    }
    row(content, columnClass = "col-sm-12") {
        const row = batchElement("div", "row");
        const column = batchElement("div", columnClass);
        column.append(content); row.append(column); this.main.append(row);
    }
    dropdown(toolbar, title, icon, entries) {
        const group = batchElement("div", "btn-group");
        const dropdown = batchElement("div", "dropdown");
        const button = batchElement("button", "btn btn-default btn-block dropdown-toggle");
        button.type = "button"; button.dataset.toggle = "dropdown";
        button.setAttribute("aria-haspopup", "true"); button.setAttribute("aria-expanded", "false");
        button.append(batchElement("i", `fa fa-${icon}`), document.createTextNode(` ${title}`));
        const menu = batchElement("ul", "dropdown-menu");
        for (const [label, action] of entries) {
            const item = batchElement("li");
            const link = batchElement("a", "", label); link.href = "#";
            link.onclick = event => { event.preventDefault(); action(); };
            item.append(link); menu.append(item);
        }
        dropdown.append(button, menu); group.append(dropdown); toolbar.append(group);
    }
    setTitle(title, showCIB = false) {
        this.title.replaceChildren(batchElement("i", "fa fa-cog"), document.createTextNode(` ${title}`));
        if (showCIB) {
            const user = batchElement("div", "pull-right text-muted");
            user.append(batchElement("small", "", window.batchConfig.id)); this.title.append(user);
        }
    }
    showMain() {
        this.setTitle("Batch Mode", true);
        this.main.hidden = false; this.reportBox.hidden = true;
        this.cancel.className = "btn btn-default cancel"; this.cancel.textContent = "Close";
        this.cancel.onclick = () => this.close();
        this.load("diff", this.diffBox);
    }
    showReport(file, title) {
        this.setTitle(title);
        this.main.hidden = true; this.reportBox.hidden = false;
        this.cancel.className = "btn btn-primary cancel"; this.cancel.textContent = "OK";
        this.cancel.onclick = () => this.showMain();
        this.load(file, this.reportBox);
    }
    inject(kind) { this.close(); new BatchEventPopup(kind); }
    drawEvents() {
        this.events.replaceChildren();
        hawkBatch.events.forEach((event, index) => {
            const item = batchElement("li", "list-group-item", event);
            const remove = batchElement("button", "btn btn-link pull-right", "×");
            remove.type = "button"; remove.setAttribute("aria-label", "Remove event");
            remove.onclick = async () => {
                try {
                    hawkBatch.events.splice(index, 1);
                    await hawkBatch.simulate();
                    this.drawEvents(); this.load("diff", this.diffBox);
                } catch (error) { this.error(error); }
            };
            item.append(remove); this.events.append(item);
        });
    }
    async load(file, target) {
        const generation = ++this.generation;
        target.replaceChildren(batchElement("div", "text-center text-muted", "Calculating list of changes..."));
        try {
            if (file === "graph") {
                const image = batchElement("img"); image.alt = "Simulated transition graph";
                image.src = `${hawkBatch.api}/result?file=graph&t=${Date.now()}`;
                image.onerror = () => { target.textContent = "Unable to load the transition graph."; };
                target.replaceChildren(image);
            } else {
                const text = await hawkBatch.report(file);
                if (generation !== this.generation) return;
                if (file === "diff" && !text.trim()) {
                    target.replaceChildren(batchElement("div", "text-center text-muted", "No configuration changes."));
                } else {
                    const pre = batchElement("pre");
                    const code = batchElement("code", file === "diff" ? "crmsh" : ["in", "out", "graph-xml"].includes(file) ? "xml" : "", text);
                    pre.append(code); target.replaceChildren(pre);
                    if (code.className && window.hljs) hljs.highlightBlock(code);
                }
            }
        } catch (error) { if (generation === this.generation) this.error(error); }
    }
}

class BatchEventPopup extends BatchPopup {
    constructor(kind) {
        super(`Inject ${kind[0].toUpperCase() + kind.slice(1)} Event`, async popup => {
            if (!this.form.reportValidity()) return;
            const value = name => this.form.elements.namedItem(name).value;
            let event;
            if (kind === "node") event = `node ${value("node")} ${value("state")}`;
            if (kind === "resource") event = `op ${value("op")}:${value("interval")} ${value("resource")} ${value("result")} ${value("node")}`;
            if (kind === "ticket") event = `ticket ${value("ticket")} ${value("action")}`;
            hawkBatch.events.push(event);
            try { await hawkBatch.simulate(); }
            catch (error) { hawkBatch.events.pop(); throw error; }
            popup.close(); new BatchShowPopup();
        });
        this.cancel.onclick = () => { this.close(); new BatchShowPopup(); };
        this.form = batchElement("form", "form-horizontal");
        const nodes = (hawkBatch.state?.nodes || []).map(node => node.Uname);
        const resources = (hawkBatch.state?.resources || []).filter(resource => resource.ObjectType !== "group").map(resource => resource.ID);
        if (kind === "node") { this.field("node", "Node", nodes); this.field("state", "State", ["online", "offline", "unclean"]); }
        if (kind === "resource") {
            this.field("resource", "Resource", resources);
            this.field("op", "Operation", ["monitor", "start", "stop", "promote", "demote", "notify", "migrate_to", "migrate_from"]);
            this.field("interval", "Interval (ms)", null);
            this.field("node", "Node", nodes);
            this.field("result", "Result", ["success", "generic error", "incorrect arguments", "unimplemented action", "insufficient permissions", "installation error", "configuration error", "not running", "promoted", "failed (promoted)"].map((name, index) => [String(index), `${name} (${index})`]));
        }
        if (kind === "ticket") { this.field("ticket", "Ticket", hawkBatch.state?.tickets || []); this.field("action", "Action", ["grant", "revoke", "standby", "activate"]); }
        this.body.append(this.form);
    }
    field(name, title, choices) {
        const row = batchElement("div", "form-group");
        const label = batchElement("label", "col-sm-4 control-label", title);
        label.htmlFor = `batch-event-${name}`;
        const column = batchElement("div", "col-sm-8");
        const input = batchElement(choices ? "select" : "input", "form-control");
        input.id = label.htmlFor; input.name = name; input.required = true;
        if (choices) for (const choice of choices) input.add(Array.isArray(choice) ? new Option(choice[1], choice[0]) : new Option(choice, choice));
        else { input.type = "number"; input.min = "0"; input.value = "0"; }
        column.append(input); row.append(label, column); this.form.append(row);
    }
}

class BatchPage {
    constructor(config) {
        this.api = config.api; this.base = config.base;
        this.state = null; this.busy = false;
        try { this.events = JSON.parse(decodeURIComponent(document.cookie.split(";").map(cookie => cookie.trim()).find(cookie => cookie.startsWith("hawk-batch-events="))?.slice("hawk-batch-events=".length) || "[]")); }
        catch { this.events = []; }
        if (!Array.isArray(this.events)) this.events = [];
    }
    async request(path, data) {
        const response = await fetch(`${this.api}/${path}`, { cache: "no-store", method: data === undefined ? "GET" : "POST", headers: data === undefined ? {} : { "Content-Type": "application/json" }, body: data === undefined ? undefined : JSON.stringify(data) });
        if (response.redirected || !response.headers.get("content-type")?.includes("application/json")) throw new Error("Unable to load batch data. Check that your session is still authenticated.");
        const result = await response.json();
        if (!response.ok) throw new Error(result.error || `Batch request failed (${response.status})`);
        return result;
    }
    notice(message, type = "danger") {
        const alert = document.createElement("x-alert");
        alert.setAttribute("type", type); alert.setAttribute("message", message);
        document.getElementById("batch-notifications").replaceChildren(alert);
    }
    async simulate() {
        document.cookie = `hawk-batch-events=${encodeURIComponent(JSON.stringify(this.events))}; path=/; SameSite=Lax`;
        const state = await this.request("simulate", { injections: this.events });
        await this.render(state); return state;
    }
    async report(file) {
        const response = await fetch(`${this.api}/result?file=${encodeURIComponent(file)}`, { cache: "no-store" });
        if (!response.ok || response.redirected) throw new Error("Unable to load simulation results.");
        return response.text();
    }
    async render(state) {
        this.state = state;
        const updates = [updateClusterStatusIndicator()];
        if (typeof updateClusterStatus === "function") updates.push(updateClusterStatus());
        await Promise.all(updates);
    }
    response(value, status = 200) {
        return new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } });
    }
    async fetch(url, options = {}) {
        const data = options.body ? JSON.parse(options.body) : null;
        if (!this.state) throw new Error("Batch data is not loaded yet.");
        if (url === "/api/cib/cluster/details/fetch") return this.response(this.state.details);
        if (url === "/api/crm/status/fetch") return this.response(this.state.crm);
        if (url === "/api/cib/resources/fetch") return this.response(this.state.resources);
        if (url === "/api/cib/node/fence/history/fetch") return this.response("");
        if (url === "/api/cib/node/attributes/fetch" || url === "/api/cib/node/utilizations/fetch") {
            const node = this.state.nodes.find(node => node.Uname === data.CibObject);
            return this.response(node?.[url.includes("utilizations") ? "Utilizations" : "Attributes"] || []);
        }
        // Agent descriptions are independent of the selected CIB. Suppress the
        // live instance lookup; the shared popup displays metadata, not values.
        if (url === "/api/cib/resource/operations/fetch" || url === "/api/cib/resource/params/fetch") {
            return fetch(`${url}?cib_id=${encodeURIComponent(window.batchConfig.id)}`, { ...options, body: JSON.stringify({ ...data, resourceID: "" }) });
        }
        const match = url.match(/^\/api\/cib\/(resource|clone|node)\/(start|stop|promote|demote|maintenance-on|maintenance-off|standby-on|standby-off|clear|migrate|cleanup|fence|clearstate)$/);
        if (!match) throw new Error("This operation is not available in batch mode.");
        const kind = match[1] === "node" ? "node" : "resource";
        let action = match[2];
        const id = typeof data === "string" ? data : data.ResourceID;
        const node = typeof data === "object" ? data.Destination || "" : "";
        try {
            if (action === "fence") this.events.push(`node ${id} unclean`);
            else if (action === "clearstate") this.events = this.events.filter(event => !event.startsWith(`node ${id} `));
            else if (action === "cleanup") this.events = this.events.filter(event => !(event.startsWith("op ") && event.split(/\s+/)[2] === id && (!node || event.split(/\s+/)[4] === node)));
            else {
                if (kind === "node") action = { "maintenance-on": "maintenance", "maintenance-off": "ready", "standby-on": "standby", "standby-off": "online" }[action] || action;
                await this.request("action", { kind, id, action, node });
            }
            await this.simulate();
            return this.response({ message: "The batch configuration has been updated." });
        } catch (error) {
            this.notice(error.message);
            return new Response(error.message, { status: 422 });
        }
    }
    async discard() {
        try {
            const result = await this.request("discard", {});
            window.location.href = result.redirect;
        } catch (error) { this.notice(error.message); }
    }
    async apply() {
        let epoch = "";
        const popup = new BatchPopup("Apply Configuration — hacluster", async dialog => {
            const result = await this.request("apply", { epoch });
            window.location.href = result.redirect;
        }, "Apply");
        popup.ok.disabled = true;
        const code = batchElement("pre", "", "Calculating list of changes..."); popup.body.append(code);
        try { const review = await this.request("review"); epoch = review.epoch; code.textContent = review.diff || "No configuration changes."; popup.ok.disabled = false; }
        catch (error) { popup.error(error); }
    }
    async start() {
        document.getElementById("btn-show-batch").onclick = () => new BatchShowPopup();
        document.getElementById("btn-apply-batch").onclick = () => this.apply();
        document.getElementById("btn-discard-batch").onclick = event => {
            event.preventDefault();
            new ConfirmActionPopup(null, null, "Any pending changes to the configuration will be lost - do you wish to proceed?", null, () => this.discard());
        };
        document.getElementById("batch-help").onclick = event => { event.preventDefault(); const popup = new BatchPopup("Batch Mode"); popup.body.textContent = "Configuration changes are made in a shadow copy of the CIB. Show lets you inspect differences and simulate node, resource, and ticket events. Discard returns to the live cluster. Apply commits the staged configuration. Wizards cannot be applied in batch mode."; };
        try { await this.simulate(); }
        catch (error) { this.notice(error.message); try { await this.render(await this.request("state")); } catch (error) { this.notice(error.message); } }
        const poll = async () => {
            if (!this.busy) {
                this.busy = true;
                try { const state = await this.request("state"); if (this.state && (this.state.details.epoch !== state.details.epoch || this.state.liveEpoch !== state.liveEpoch)) await this.simulate(); else await this.render(state); }
                catch (error) { this.notice(error.message); }
                finally { this.busy = false; }
            }
            this.timer = setTimeout(poll, 5000);
        };
        this.timer = setTimeout(poll, 5000);
        window.addEventListener("pagehide", () => clearTimeout(this.timer), { once: true });
    }
}
window.hawkBatch = new BatchPage(window.batchConfig);
