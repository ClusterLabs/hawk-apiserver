"use strict";

class TagForm {
    constructor(form) {
        this.form = form;
        this.editing = form.dataset.editing === "true";
        this.id = form.dataset.tagId;
        this.input = form.querySelector("#input-tag-id");
        this.list = form.querySelector("#tag-objects");
        this.button = form.querySelector("#create-tag-button");
        this.revert = form.querySelector("#revert-tag");
        this.ready = false;
        this.saving = false;
        form.addEventListener("submit", event => { event.preventDefault(); this.save(); });
        form.addEventListener("change", () => { this.dirty = true; });
        if (this.editing) {
            this.revert.addEventListener("click", () => this.discard(() => this.load()));
            document.getElementById("rename-tag").addEventListener("click", () => new RenamePopup(this.id));
            document.getElementById("delete-tag").addEventListener("click", () => new DeletePopup(this.id, () => this.delete()));
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
            if (flash === "created") this.notify("success", "Tag created successfully");
            if (flash === "renamed") this.notify("success", "Tag renamed successfully");
            if (flash === "error") this.notify("danger", url.searchParams.get("msg") || "Unable to rename tag.");
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
        document.getElementById("tag-notifications").replaceChildren(alert);
    }

    busy(value) {
        this.button.disabled = value || !this.ready;
        if (this.revert) this.revert.disabled = value || !this.ready;
        this.list.querySelectorAll("input").forEach(input => { input.disabled = value; });
        this.form.setAttribute("aria-busy", String(value));
        for (const name of ["rename-tag", "delete-tag"]) {
            const button = document.getElementById(name);
            if (button) button.disabled = value || !this.ready;
        }
    }

    async load() {
        this.ready = false;
        this.busy(true);
        this.list.setAttribute("aria-busy", "true");
        try {
            const url = this.editing ? `/api/cib/tag/fetch?id=${encodeURIComponent(this.id)}` : "/api/cib/tag/objects/fetch";
            const response = await fetch(url, { method: this.editing ? "GET" : "POST", cache: "no-store" });
            if (!response.ok) throw new Error(await response.text() || "Unable to load tag.");
            const data = await response.json();
            const objects = this.editing ? data.objects : data;
            const selected = new Set(this.editing ? data.refs : []);
            this.list.replaceChildren();
            for (const id of objects) {
                const row = document.createElement("li");
                row.className = "list-group-item";
                const checkbox = document.createElement("input");
                checkbox.type = "checkbox";
                checkbox.name = "tag[refs][]";
                checkbox.value = id;
                checkbox.checked = selected.has(id);
                checkbox.setAttribute("aria-label", id);
                row.append(checkbox, document.createTextNode(id));
                this.list.append(row);
            }
            if (!objects.length) {
                const empty = document.createElement("li");
                empty.className = "list-group-item text-warning";
                empty.textContent = "No objects available";
                this.list.append(empty);
            }
            this.ready = true;
            this.dirty = false;
            return true;
        } catch (error) { this.notify("danger", error.message); return false; }
        finally { this.list.setAttribute("aria-busy", "false"); this.busy(false); }
    }

    async save() {
        if (!this.ready || this.saving) return;
        if (!this.input.value.trim()) { this.notify("danger", "ID is required"); return; }
        if (!this.list.querySelector("input:checked")) { this.notify("danger", "No Tag resources specified"); return; }
        this.saving = true;
        this.busy(true);
        try {
            const tag = new Tag(this.input, this.list);
            if (this.editing) {
                const result = await tag.update();
                if (await this.load()) this.notify("success", result.message);
            } else await tag.create();
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
            if (!response.ok) throw new Error(await response.text() || "Unable to delete tag.");
            sessionStorage.setItem("configFlash", "Tag deleted successfully");
            window.location.href = "/cib/live/config/edit#tags";
        } catch (error) { this.notify("danger", error.message); this.busy(false); }
    }
}

new TagForm(document.getElementById("tag-create-form"));
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/resources/types"]').setAttribute("aria-current", "page");
