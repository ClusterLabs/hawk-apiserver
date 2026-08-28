class DashboardAuthenticationError extends Error {}
class DashboardProtocolError extends Error {}

// Connections go directly to the selected Hawk server, using its session cookies.
class DashboardClusterClient {
    constructor(config) {
        this.config = config;
        this.protocol = config.local ? "go" : null;
        this.hosts = config.local ? [] : [config.host];
        this.hostIndex = 0;
        try {
            this.baseURL = config.local ? window.location.origin : DashboardClusterClient.clusterURL(config);
        } catch (error) {
            this.configurationError = error;
            this.baseURL = "";
        }
    }

    static clusterURL(config) {
        let host = config.host.trim();
        if (host.includes(":") && !host.startsWith("[")) host = `[${host}]`;
        const url = new URL(`${config.https === false ? "http" : "https"}://${host}:${config.port || 7630}`);
        if (url.username || url.password || url.pathname !== "/" || url.search || url.hash) {
            throw new Error("Cluster hostname must contain only a hostname or IP address.");
        }
        return url.origin;
    }

    async request(path, options = {}, signal, timeout = 30000) {
        const controller = new AbortController();
        const abort = () => controller.abort();
        if (signal?.aborted) controller.abort();
        else signal?.addEventListener("abort", abort, { once: true });
        const timer = setTimeout(abort, timeout);
        try {
            const response = await fetch(this.baseURL + path, {
                ...options, credentials: "include", signal: controller.signal,
                headers: { Accept: "application/json", ...options.headers },
            });
            if ([401, 403].includes(response.status) ||
                (response.redirected && /\/login(?:[/.?]|$)/.test(new URL(response.url).pathname))) {
                throw new DashboardAuthenticationError("Sign in to this cluster.");
            }
            if ([404, 405].includes(response.status)) throw new DashboardProtocolError("Endpoint unavailable");
            const type = response.headers.get("content-type") || "";
            if (response.status === 503 && path === "/api/cib/cluster/dashboard/fetch" && !type.includes("application/json")) {
                return { clusterDetails: { status: "offline", summary: await response.text(), epoch: "" }, nodes: [], resources: [] };
            }
            if (!type.includes("application/json")) {
                if (!response.ok) throw new Error(`Cluster request failed (${response.status}).`);
                throw new DashboardProtocolError("The cluster returned HTML instead of JSON.");
            }
            const data = await response.json();
            if (!response.ok && !(response.status === 503 && data.clusterDetails)) {
                throw new Error(typeof data.error === "string" ? data.error : `Cluster request failed (${response.status}).`);
            }
            return data;
        } catch (error) {
            if (signal?.aborted) throw error;
            if (controller.signal.aborted || error instanceof TypeError) {
                const failure = new Error(controller.signal.aborted ? "Connection to cluster timed out." :
                    "Cannot reach cluster. Check its address, TLS certificate, and browser cross-origin access.");
                failure.connectionFailure = true;
                throw failure;
            }
            throw error;
        } finally {
            clearTimeout(timer);
            signal?.removeEventListener("abort", abort);
        }
    }

    async fetchStatus(signal) {
        if (this.configurationError) throw this.configurationError;
        if (this.protocol !== "legacy") {
            try {
                const data = await this.request("/api/cib/cluster/dashboard/fetch", { method: "POST" }, signal);
                if (!data?.clusterDetails) throw new DashboardProtocolError("Unrecognized dashboard response");
                this.protocol = "go";
                this.rememberHosts(data.nodes || []);
                return data;
            } catch (error) {
                // Old installations may not expose the Go endpoint (including its CORS headers).
                // Probe their documented status endpoint once; retain the selected protocol.
                if (this.protocol || signal?.aborted || error instanceof DashboardAuthenticationError) throw error;
            }
        }
        const data = await this.request("/cib/live?format=json", {}, signal);
        const normalized = DashboardClusterClient.normalizeLegacy(data);
        this.protocol = "legacy";
        this.rememberHosts(data.nodes.filter(node => node.remote === false && !node.host));
        return normalized;
    }

    rememberHosts(nodes) {
        if (this.config.local) return;
        for (const node of nodes) {
            const host = node.name || node.uname;
            if (host && !this.hosts.includes(host)) this.hosts.push(host);
        }
    }

    tryNextHost() {
        if (this.config.local || this.hosts.length < 2) return false;
        this.hostIndex = (this.hostIndex + 1) % this.hosts.length;
        this.baseURL = DashboardClusterClient.clusterURL({ ...this.config, host: this.hosts[this.hostIndex] });
        this.protocol = null;
        return true;
    }

    async monitor(epoch, signal) {
        const endpoint = this.protocol === "legacy" ? "/monitor.json" : "/monitor";
        const data = await this.request(`${endpoint}?${epoch || ""}`, {}, signal, 90000);
        if (typeof data?.epoch !== "string") throw new Error("Invalid cluster monitor response.");
        return data.epoch;
    }

    async login(username, password, signal) {
        return this.request("/login.json", {
            method: "POST",
            body: new URLSearchParams({ "session[username]": username, "session[password]": password }),
        }, signal);
    }

    static normalizeLegacy(data) {
        if (!data?.meta || !Array.isArray(data.nodes) || !Array.isArray(data.resources)) {
            throw new DashboardProtocolError("Unrecognized legacy cluster status response.");
        }
        const statuses = { ok: "online", errors: "unclean", nostonith: "nofencing", offline: "offline", maintenance: "online" };
        const nodes = data.nodes.map(node => ({
            id: node.id, name: node.name || node.uname,
            online: ["online", "standby", "pending"].includes(node.state),
            standby: node.standby || node.state === "standby", maintenance: !!node.maintenance,
            isDC: (node.name || node.uname) === data.meta.dc, eventsFound: node.fence_history || "",
        }));
        const errors = (data.errors || [])
            .map(error => typeof error === "string" ? error : error?.msg)
            .filter(message => typeof message === "string" && message.length > 0);
        return {
            clusterDetails: {
                clusterName: data.crm_config?.cluster_name || "Cluster",
                status: statuses[data.meta.status] || "unclean", epoch: data.meta.epoch || "",
                summary: errors.join("\n") || (data.meta.status === "maintenance" ? "Cluster is in maintenance mode." : data.meta.status),
                updateOrigin: data.meta.update_origin, updateUser: data.meta.update_user, stack: data.meta.stack,
                maintenance: data.meta.status === "maintenance",
            },
            nodes,
            resources: data.resources.map(resource => ({
                name: resource.id, maintenance: !!resource.maintenance,
                active: !["stopped", "unknown"].includes(resource.state),
                roles: nodes.map(node => {
                    if (resource.maintenance) return "";
                    if (resource.state === "stopped") return "stopped";
                    if (!node.online) return "offline";
                    const role = resource.running_on?.[node.name];
                    return { started: "started", master: "promoted", promoted: "promoted", slave: "unpromoted", unpromoted: "unpromoted" }[role] || "";
                }),
            })),
        };
    }
}
