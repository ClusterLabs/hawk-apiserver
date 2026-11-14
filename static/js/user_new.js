"use strict";

class ACLTargetCreator {
    constructor(form) {
        this.form = form;
        this.fields = form.querySelector("#user-fields");
        this.roles = form.querySelector("#user_roles");
        this.warning = document.getElementById("roles-acl-warning");
        this.warning.querySelector(".close").addEventListener("click", () => { this.warning.hidden = true; });
        form.addEventListener("submit", event => { event.preventDefault(); this.create(); });
        this.load();
    }

    notify(message) {
        const alert = document.createElement("x-alert");
        alert.setAttribute("type", "danger");
        alert.setAttribute("message", message);
        document.getElementById("user-notifications").replaceChildren(alert);
    }

    busy(value) {
        this.fields.disabled = value;
        this.form.setAttribute("aria-busy", String(value));
    }

    async load() {
        try {
            const data = await configJSON("/api/cib/acl/users");
            if (!data.supported) throw new Error("ACL is not supported by this cluster.");
            this.warning.hidden = data.enabled;
            this.roles.replaceChildren(...data.roles.map(role => new Option(role.id, role.id)));
            this.fields.hidden = false;
            this.busy(false);
        } catch (error) { this.notify(error.message); }
        finally { document.getElementById("user-loading").hidden = true; }
    }

    async create() {
        if (this.fields.disabled || !this.form.reportValidity()) return;
        const target = {
            id: this.form.querySelector("#user_id").value,
            roles: Array.from(this.roles.selectedOptions, option => option.value),
        };
        this.busy(true);
        try {
            const data = await configJSON("/api/cib/acl/users/create", {
                method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(target),
            });
            sessionStorage.setItem("usersFlash", data.message);
            window.location.href = "/cib/live/users";
        } catch (error) { this.notify(error.message); this.busy(false); }
    }
}

new ACLTargetCreator(document.querySelector(".user-editor"));
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/users"]').setAttribute("aria-current", "page");
