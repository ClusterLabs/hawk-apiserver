"use strict";

class ACLTargetForm {
    constructor(form) {
        this.form = form;
        this.editing = form.dataset.editing === "true";
        this.id = form.dataset.targetId;
        this.fields = form.querySelector("#user-fields");
        this.roles = form.querySelector("#user_roles");
        this.remove = document.getElementById("delete-user");
        this.warning = document.getElementById("roles-acl-warning");
        this.warning.querySelector(".close").addEventListener("click", () => { this.warning.hidden = true; });
        form.addEventListener("submit", event => { event.preventDefault(); this.save(); });
        form.addEventListener("input", () => { this.dirty = true; });
        form.addEventListener("change", () => { this.dirty = true; });
        if (this.editing) {
            form.querySelector('[name="revert"]').addEventListener("click", () => this.discard(() => this.load()));
            this.remove.addEventListener("click", () => new DeletePopup(this.id, () => this.delete()));
            const back = form.querySelector("a.back");
            back.addEventListener("click", event => {
                if (!this.dirty) return;
                event.preventDefault();
                this.discard(() => { window.location.href = back.href; });
            });
        }
        this.load();
    }

    notify(message, danger = true) {
        const alert = document.createElement("x-alert");
        alert.setAttribute("type", danger ? "danger" : "success");
        alert.setAttribute("message", message);
        document.getElementById("user-notifications").replaceChildren(alert);
    }

    busy(value) {
        this.fields.disabled = value;
        if (this.remove) this.remove.disabled = value;
        this.form.setAttribute("aria-busy", String(value));
    }

    async load() {
        this.busy(true);
        try {
            const data = await configJSON(this.editing ? `/api/cib/acl/user?id=${encodeURIComponent(this.id)}` : "/api/cib/acl/users");
            if (!this.editing && !data.supported) throw new Error("ACL is not supported by this cluster.");
            this.warning.hidden = data.enabled;
            const selected = new Set(this.editing ? data.target.roles : []);
            this.roles.replaceChildren(...data.roles.map(role => new Option(role.id, role.id, false, selected.has(role.id))));
            this.fields.hidden = false;
            this.dirty = false;
            this.busy(false);
            return true;
        } catch (error) { this.notify(error.message); return false; }
        finally { document.getElementById("user-loading").hidden = true; }
    }

    async save() {
        if (this.fields.disabled || !this.form.reportValidity()) return;
        const target = {
            id: this.editing ? this.id : this.form.querySelector("#user_id").value,
            roles: Array.from(this.roles.selectedOptions, option => option.value),
        };
        this.busy(true);
        try {
            const data = await configJSON(this.editing ? "/api/cib/acl/user" : "/api/cib/acl/users/create", {
                method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(target),
            });
            if (this.editing) {
                if (await this.load()) this.notify(data.message, false);
            } else {
                sessionStorage.setItem("usersFlash", data.message);
                window.location.href = "/cib/live/users";
            }
        } catch (error) { this.notify(error.message); this.busy(false); }
    }

    discard(onConfirm) {
        if (!this.dirty) { onConfirm(); return; }
        const popup = new Popup();
        const body = configElement("div", "Any changes will be lost - do you wish to proceed?", "modal-body");
        const footer = configElement("div", undefined, "modal-footer");
        const cancel = configElement("button", "Cancel", "btn btn-default");
        const ok = configElement("button", "OK", "btn btn-primary");
        cancel.onclick = () => popup.close();
        ok.onclick = () => { popup.close(); onConfirm(); };
        footer.append(cancel, ok);
        popup._appendModalContentChild(body);
        popup._appendModalContentChild(footer);
    }

    async delete() {
        this.busy(true);
        try {
            const data = await configJSON("/api/cib/acl/users/delete", {
                method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ id: this.id }),
            });
            sessionStorage.setItem("usersFlash", data.message);
            window.location.href = "/cib/live/users";
        } catch (error) { this.notify(error.message); this.busy(false); }
    }
}

new ACLTargetForm(document.querySelector(".user-editor"));
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/users"]').setAttribute("aria-current", "page");
