"use strict";

class AlertForm {
    constructor(form) {
        this.form = form;
        this.editing = form.dataset.editing === "true";
        this.id = form.dataset.alertId;
        this.fields = form.querySelector("#alert-form-fields");
        this.button = form.querySelector("#create-alert-button");
        this.recipients = [];
        this.recipientCount = 0;
        this.ready = false;
        this.saving = false;
        form.addEventListener("submit", event => { event.preventDefault(); this.save(); });
        form.addEventListener("input", () => { this.dirty = true; });
        form.addEventListener("change", () => { this.dirty = true; });
        form.addEventListener("click", event => {
            const path = event.composedPath();
            if (path.some(node => node.matches?.("x-inputs-kvgroup, x-kvgroup")) &&
                path.some(node => node.matches?.("button"))) this.dirty = true;
        });
        document.getElementById("add-recipient-button").addEventListener("click", () => {
            let id;
            do { id = `alerts-recipient-temporary-id-${++this.recipientCount}`; }
            while (this.recipients.some(recipient => recipient.id === id));
            this.editRecipient({ id, value: "", params: {}, meta: {} }, false);
        });
        if (this.editing) {
            document.getElementById("revert-alert").addEventListener("click", () => this.discard(() => this.load()));
            document.getElementById("rename-alert").addEventListener("click", () => new RenamePopup(this.id));
            document.getElementById("delete-alert").addEventListener("click", () => new DeletePopup(this.id, () => this.delete()));
            const back = form.querySelector("a.back");
            back.addEventListener("click", event => {
                if (!this.dirty) return;
                event.preventDefault();
                this.discard(() => { window.location.href = back.href; });
            });
        }
        const url = new URL(window.location.href);
        const flash = url.searchParams.get("flash");
        if (this.editing && flash) {
            if (flash === "created") this.notify("success", "Alert created successfully");
            if (flash === "renamed") this.notify("success", "Alert renamed successfully");
            if (flash === "error") this.notify("danger", url.searchParams.get("msg") || "Unable to rename alert.");
            url.searchParams.delete("flash");
            url.searchParams.delete("msg");
            history.replaceState({}, "", url);
        }
        this.load();
    }

    notify(type, message) {
        const alert = document.createElement("x-alert");
        alert.setAttribute("type", type);
        alert.setAttribute("message", message.trim());
        document.getElementById("alert-notifications").replaceChildren(alert);
    }

    busy(value) {
        this.fields.disabled = value;
        this.button.disabled = value || !this.ready;
        this.form.setAttribute("aria-busy", String(value));
        for (const group of this.fields.querySelectorAll("x-inputs-kvgroup, x-kvgroup")) {
            group.shadowRoot.querySelectorAll("input, select, button").forEach(input => { input.disabled = value; });
        }
        for (const id of ["rename-alert", "delete-alert", "revert-alert"]) {
            const button = document.getElementById(id);
            if (button) button.disabled = value || !this.ready;
        }
    }

    async load() {
        this.ready = false;
        this.busy(true);
        try {
            let data = { recipients: [], params: {}, meta: {} };
            if (this.editing) {
                const response = await fetch(`/api/cib/alert/fetch?id=${encodeURIComponent(this.id)}`, { cache: "no-store" });
                if (!response.ok) throw new Error(await response.text() || "Unable to load alert.");
                data = await response.json();
                document.getElementById("input-alert-path").value = data.path;
            }
            this.recipients = data.recipients;
            this.renderRecipients();
            const container = document.getElementById("alert-params");
            container.replaceChildren();
            this.params = new AlertAttributes(container, "alert[params]", data.params);
            const previous = document.getElementById("kvgroup-meta_attributes");
            this.meta = previous;
            if (this.editing) {
                this.meta = document.createElement("x-kvgroup");
                for (const attr of previous.attributes) this.meta.setAttribute(attr.name, attr.value);
                this.meta.setAttribute("initial-values", JSON.stringify(data.meta));
                previous.replaceWith(this.meta);
            }
            if ((await Promise.all([this.meta.loaded, this.params.loaded])).some(ready => ready !== true)) {
                throw new Error("Unable to load alert attributes.");
            }
            this.ready = true;
            this.fields.hidden = false;
            this.dirty = false;
            return true;
        } catch (error) { this.notify("danger", error.message); return false; }
        finally { document.getElementById("alert-loading").hidden = true; this.busy(false); }
    }

    editRecipient(recipient, editing) {
        new AlertRecipientPopup(recipient, editing, updated => {
            const index = this.recipients.findIndex(item => item.id === updated.id);
            if (index < 0) this.recipients.push(updated);
            else this.recipients[index] = updated;
            this.dirty = true;
            this.renderRecipients();
        });
    }

    renderRecipients() {
        const list = document.getElementById("alert-recipients");
        list.replaceChildren();
        for (const recipient of this.recipients) {
            const row = document.createElement("div");
            row.className = "kvsection-row";
            const label = document.createElement("label");
            label.textContent = "To";
            const control = document.createElement("div");
            control.className = "alert-control input-group";
            const input = document.createElement("input");
            input.className = "form-control";
            input.value = recipient.value;
            input.readOnly = true;
            input.name = `alert[recipients][${recipient.id}][value]`;
            input.setAttribute("aria-label", "To");
            const edit = document.createElement("button");
            edit.type = "button";
            edit.className = "btn btn-default";
            edit.title = "Edit Recipient";
            edit.innerHTML = '<i class="fa fa-pencil" aria-hidden="true"></i>';
            edit.addEventListener("click", () => this.editRecipient(recipient, true));
            const remove = document.createElement("button");
            remove.type = "button";
            remove.className = "btn btn-default";
            remove.title = "Remove Recipient";
            remove.innerHTML = '<i class="fa fa-minus" aria-hidden="true"></i>';
            remove.addEventListener("click", () => {
                this.recipients = this.recipients.filter(item => item.id !== recipient.id);
                this.dirty = true;
                this.renderRecipients();
            });
            control.append(input, edit, remove);
            row.append(label, control);
            list.append(row);
        }
    }

    async save() {
        if (!this.ready || this.saving) return;
        const sanity = this.meta.submitSanityCheck();
        if (!sanity.ok) { this.notify("danger", sanity.message); return; }
        this.saving = true;
        this.busy(true);
        try {
            const alert = new Alert(document.getElementById("input-alert-id"), document.getElementById("input-alert-path"), this.recipients, this.params, this.meta);
            if (this.editing) {
                const result = await alert.update();
                if (await this.load()) this.notify("success", result.message);
            } else await alert.create();
        } catch (error) { this.notify("danger", error.message); }
        finally { this.saving = false; this.busy(false); }
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
            const response = await fetch("/api/cib/resource/delete", {
                method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(this.id),
            });
            if (!response.ok) throw new Error(await response.text() || "Unable to delete alert.");
            sessionStorage.setItem("configFlash", "Alert deleted successfully");
            window.location.href = "/cib/live/config/edit#alerts";
        } catch (error) { this.notify("danger", error.message); this.busy(false); }
    }
}

new AlertForm(document.getElementById("alert-create-form"));
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/resources/types"]').setAttribute("aria-current", "page");
