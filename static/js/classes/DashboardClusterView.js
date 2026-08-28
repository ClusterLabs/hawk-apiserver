// One view owns the DOM, data and refresh lifecycle for one cluster.
class DashboardClusterView {
    static nextID = 0;
    nodes = [];
    resources = [];
    clusterDetails = null;
    searchText = "";
    nodeFilter = "show_all";
    epoch = "";
    controller = null;
    credentials = null;

    constructor(config, client) {
        this.config = config;
        this.client = client;
        const id = `dashboard-cluster-${DashboardClusterView.nextID++}`;
        this.panel = document.getElementById("dashboard-cluster-template").content.firstElementChild.cloneNode(true);
        this.panel.id = id;
        this.panel.dashboardView = this;
        const removeControl = this.panel.querySelector(".cluster-remove");
        if (config.local) removeControl.remove();
        else removeControl.hidden = false;
        this.tbody = this.panel.querySelector("tbody");
        this.errors = this.panel.querySelector(".cluster-errors");
        this.clusterPanel = this.panel.querySelector(".status-table > .panel");
        this.alarm = this.panel.querySelector(".cluster-status-alarm");
        this.alarm.id = `${id}-alarm`;
        this.filter = this.panel.querySelector(".filter-dropdown");
        this.filter.querySelectorAll("input").forEach(input => input.name = `${id}-filter`);
        const toggle = this.panel.querySelector(".cluster-filter button");
        toggle.id = `${id}-filter-toggle`;
        this.filter.setAttribute("aria-labelledby", toggle.id);
        this.tab = document.createElement("li");
        this.tab.setAttribute("role", "presentation");
        const link = document.createElement("a");
        link.href = `#${id}`;
        link.id = `${id}-tab`;
        link.setAttribute("role", "tab");
        link.setAttribute("aria-selected", "false");
        link.setAttribute("aria-controls", id);
        this.panel.setAttribute("aria-labelledby", link.id);
        this.title = document.createElement("span");
        this.title.textContent = `${config.name} `;
        this.indicator = document.createElement("i");
        link.append(this.title, this.indicator);
        this.tab.appendChild(link);
        const clusterLink = this.panel.querySelector(".cluster-url");
        clusterLink.href = this.client.baseURL;
        clusterLink.parentElement.hidden = !!this.client.configurationError;
        clusterLink.rel = "noopener";
    }

    setSearch(value) {
        this.searchText = value;
        this.applyFilters();
    }

    setNodeFilter(value) {
        this.nodeFilter = value;
        this.applyFilters();
    }

    matchesNode(node, filter = this.nodeFilter) {
        if (!node) return false;
        if (filter === "offline") return node.online === false;
        if (["online", "maintenance", "standby"].includes(filter)) return Boolean(node[filter]);
        return true;
    }

    applyFilters() {
        const search = this.searchText.trim().toLowerCase();
        this.tbody.querySelectorAll("[data-resource-name]").forEach(row => {
            row.classList.toggle("hidden", !row.dataset.resourceName.toLowerCase().includes(search));
        });
        this.tbody.querySelectorAll("[data-node-index]").forEach(cell => {
            cell.classList.toggle("hidden", !this.matchesNode(this.nodes[cell.dataset.nodeIndex]));
        });
    }

    render(data) {
        this.clusterDetails = data.clusterDetails;
        this.nodes = data.nodes || [];
        this.resources = data.resources || [];
        this.epoch = this.clusterDetails.epoch || "";
        if (this.config.local) this.title.textContent = `${this.clusterDetails.clusterName || this.config.name} `;
        this.indicator.className = getClusterStatusIconClasses(this.clusterDetails.status);
        const status = this.clusterDetails.status;
        this.clusterPanel.className = "panel " +
            (this.clusterDetails.maintenance || status === "nofencing" ? "panel-warning" :
                status === "online" ? "panel-default" : "panel-danger");
        this.panel.querySelector(".cluster-url").href = this.client.baseURL;
        setClusterStatusBar(this.alarm.id, this.clusterDetails.summary, this.clusterDetails.status);
        if (this.clusterDetails.maintenance) {
            this.alarm.className = "alert alert-warning";
        }
        if (this.clusterDetails.maintenance || (status === "online" && this.nodes.some(node => node.maintenance))) {
            this.indicator.className = "fa fa-wrench";
        }
        this.tbody.replaceChildren();
        insertAllNodesRow(this.tbody, this.clusterDetails, this.nodes);
        this.resources.forEach(resource => insertResourceRow(this.tbody, resource));
        ["offline", "maintenance", "standby"].forEach(filter => {
            const input = this.filter.querySelector(`input[value="${filter}"]`);
            input.disabled = !this.nodes.some(node => this.matchesNode(node, filter));
            input.closest("label").classList.toggle("line-through-filter", input.disabled);
            if (input.disabled && this.nodeFilter === filter) this.nodeFilter = "show_all";
        });
        this.filter.querySelector(`input[value="${this.nodeFilter}"]`).checked = true;
        this.applyFilters();
        this.errors.replaceChildren();
    }

    showError(error) {
        this.errors.className = "cluster-errors alert alert-danger";
        this.errors.textContent = error.message;
        this.indicator.className = "fa fa-exclamation-triangle text-danger";
    }

    async refresh(signal) {
        const data = await this.client.fetchStatus(signal);
        if (signal.aborted) return;
        this.render(data);
        this.errors.className = "cluster-errors";
    }

    async waitForLogin(signal) {
        this.errors.replaceChildren();
        this.errors.className = "cluster-errors";
        this.tbody.replaceChildren();
        this.alarm.className = "hidden";
        const form = document.createElement("form");
        form.className = "form-horizontal";
        const fields = {};
        for (const name of ["host", "username", "password"]) {
            const group = document.createElement("div");
            group.className = "form-group";
            const label = document.createElement("label");
            label.className = "col-sm-3 control-label";
            label.textContent = { host: "Hostname of node in cluster", username: "Username", password: "Password" }[name];
            const wrapper = document.createElement("div");
            wrapper.className = "col-sm-4 dashboard-login";
            const input = document.createElement("input");
            input.id = `${this.panel.id}-login-${name}`;
            label.htmlFor = input.id;
            input.type = name === "password" ? "password" : "text";
            input.name = name;
            input.className = "form-control";
            input.autocomplete = name === "password" ? "current-password" : "username";
            if (name === "host") {
                input.readOnly = true;
                input.value = new URL(this.client.baseURL).hostname;
            } else {
                input.placeholder = label.textContent;
                if (name === "username") input.value = "hacluster";
            }
            fields[name] = input;
            wrapper.appendChild(input);
            group.append(label, wrapper);
            form.appendChild(group);
        }
        const controls = document.createElement("div");
        controls.className = "form-group";
        const wrapper = document.createElement("div");
        wrapper.className = "col-sm-offset-3 col-sm-4";
        const button = document.createElement("button");
        button.type = "submit";
        button.className = "btn btn-primary";
        button.textContent = "Connect";
        wrapper.appendChild(button);
        controls.appendChild(wrapper);
        form.appendChild(controls);
        this.errors.appendChild(form);
        await new Promise(resolve => {
            const finish = () => {
                signal.removeEventListener("abort", finish);
                fields.password.value = "";
                form.remove();
                resolve();
            };
            if (signal.aborted) { finish(); return; }
            signal.addEventListener("abort", finish, { once: true });
            form.onsubmit = event => {
                event.preventDefault();
                if (signal.aborted) return;
                this.credentials = { username: fields.username.value.trim() || "hacluster", password: fields.password.value };
                Object.values(fields).forEach(input => input.disabled = true);
                finish();
            };
        });
    }

    showRetry(error) {
        this.showError(new Error(`${error.message} Retrying every 5 seconds...`));
        if (this.config.local) return;
        const cancel = document.createElement("button");
        cancel.type = "button";
        cancel.className = "btn btn-default";
        cancel.textContent = "Cancel";
        cancel.onclick = () => {
            this.stop();
            this.start(true);
        };
        this.errors.appendChild(cancel);
    }

    async start(promptLogin = false) {
        if (this.controller) return;
        if (this.client.configurationError) { this.showError(this.client.configurationError); return; }
        const controller = new AbortController();
        this.controller = controller;
        const { signal } = controller;
        let needsRefresh = true;
        let needsLogin = false;
        let operation = "status";
        try {
            if (promptLogin) {
                await this.waitForLogin(signal);
                needsLogin = true;
            }
            while (!signal.aborted) {
                try {
                    if (needsLogin) {
                        operation = "login";
                        this.indicator.className = "fa fa-refresh fa-spin";
                        await this.client.login(this.credentials.username, this.credentials.password, signal);
                        if (signal.aborted) break;
                        needsLogin = false;
                    }
                    if (needsRefresh) {
                        operation = "status";
                        await this.refresh(signal);
                    }
                    if (signal.aborted) break;
                    operation = "monitor";
                    const epoch = await this.client.monitor(this.epoch, signal);
                    needsRefresh = epoch !== this.epoch;
                    // Avoid a tight loop if a server answers unchanged epochs immediately.
                    await this.wait(1000, signal);
                } catch (error) {
                    if (signal.aborted) break;
                    needsRefresh = true;
                    if (!this.config.local && !this.credentials && operation === "status") {
                        await this.waitForLogin(signal);
                        needsLogin = true;
                    } else {
                        this.showRetry(error);
                        if (!this.config.local && this.credentials && operation !== "monitor") {
                            const changedHost = this.client.tryNextHost();
                            if (changedHost) this.epoch = "";
                            needsLogin = operation === "login" || changedHost;
                        }
                        await this.wait(5000, signal);
                    }
                }
            }
        } finally {
            if (this.controller === controller) this.controller = null;
        }
    }

    wait(ms, signal) {
        return new Promise(resolve => {
            const finish = () => {
                clearTimeout(timer);
                signal.removeEventListener("abort", finish);
                resolve();
            };
            const timer = setTimeout(finish, ms);
            if (signal.aborted) finish();
            else signal.addEventListener("abort", finish, { once: true });
        });
    }

    reconfigure(config, client) {
        this.stop();
        this.config = config;
        this.client = client;
        this.epoch = "";
        this.nodes = [];
        this.resources = [];
        this.clusterDetails = null;
        this.tbody.replaceChildren();
        this.alarm.className = "hidden";
        this.title.textContent = `${config.name} `;
        this.panel.querySelector(".cluster-url").href = client.baseURL;
        this.panel.querySelector(".cluster-url").parentElement.hidden = !!client.configurationError;
        this.start();
    }

    stop() {
        this.credentials = null;
        this.controller?.abort();
        this.controller = null;
    }

    destroy() {
        this.stop();
        this.tab.remove();
        this.panel.remove();
    }
}
