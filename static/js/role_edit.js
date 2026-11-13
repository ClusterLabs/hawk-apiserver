"use strict";

class RoleEditor {
    constructor(form) {
        this.form = form;
        this.id = form.dataset.roleId;
        this.rules = form.querySelector("#rules");
        this.fields = form.querySelector("#role-fields");
        this.remove = document.getElementById("delete-role");
        this.warning = document.getElementById("roles-acl-warning");
        this.warning.querySelector(".close").addEventListener("click", () => { this.warning.hidden = true; });
        form.addEventListener("submit", event => { event.preventDefault(); this.save(); });
        form.querySelector('[name="revert"]').addEventListener("click", () => {
            if (this.dirty) this.confirmDiscard(() => this.load()); else this.load();
        });
        this.remove.addEventListener("click", () => new DeletePopup(this.id, () => this.delete()));
        form.addEventListener("input", () => { this.dirty = true; });
        form.querySelector("a.back").addEventListener("click", event => {
            if (!this.dirty) return;
            event.preventDefault();
            this.confirmDiscard(() => { window.location.href = "/cib/live/roles"; });
        });
        this.load();
    }

    confirmDiscard(onConfirm) {
        const popup = new Popup();
        const body = configElement("div", "Any changes will be lost - do you wish to proceed?", "modal-body");
        const footer = configElement("div", undefined, "modal-footer");
        const cancel = configElement("button", "Cancel", "btn btn-default");
        const confirm = configElement("button", "OK", "btn btn-primary");
        cancel.onclick = () => popup.close();
        confirm.onclick = () => { popup.close(); onConfirm(); };
        footer.append(cancel, confirm);
        popup._appendModalContentChild(body);
        popup._appendModalContentChild(footer);
    }

    notify(message, danger = false) {
        const alert = document.createElement("x-alert");
        alert.setAttribute("type", danger ? "danger" : "success");
        alert.setAttribute("message", message);
        document.getElementById("role-notifications").replaceChildren(alert);
    }

    busy(value) {
        this.fields.disabled = value;
        this.remove.disabled = value;
        this.form.setAttribute("aria-busy", String(value));
    }

    async load() {
        this.busy(true);
        try {
            const data = await configJSON(`/api/cib/acl/role?id=${encodeURIComponent(this.id)}`);
            this.legacy = data.legacy;
            this.warning.hidden = data.enabled;
            this.rules.replaceChildren();
            for (const rule of data.role.rules) this.addRule(rule);
            this.dirty = false;
            this.fields.hidden = false;
            this.busy(false);
        } catch (error) { this.notify(error.message, true); }
        finally { document.getElementById("role-loading").hidden = true; }
    }

    addRule(rule = {}, after = null) {
        const fieldset = configElement("fieldset");
        const legend = configElement("legend", "Rule");
        const controls = configElement("div", undefined, "pull-right");
        for (const [action, icon, title] of [
            ["add", "plus", "Add a rule below"],
            ["remove", "minus", "Remove this rule"],
            ["up", "arrow-up", "Move rule up"],
            ["down", "arrow-down", "Move rule down"],
        ]) {
            const button = configElement("button", undefined, `rule ${action}`);
            button.type = "button";
            button.title = title;
            button.setAttribute("aria-label", title);
            button.append(configElement("i", undefined, `fa fa-${icon}`));
            button.addEventListener("click", () => {
                if (action === "add") this.addRule({}, fieldset);
                if (action === "remove") fieldset.remove();
                if (action === "up" && fieldset.previousElementSibling) fieldset.previousElementSibling.before(fieldset);
                if (action === "down" && fieldset.nextElementSibling) fieldset.nextElementSibling.after(fieldset);
                this.reindex();
                this.dirty = true;
            });
            controls.append(button);
        }
        legend.append(controls);
        fieldset.append(legend);
        const fields = [["right", "Right"], ["xpath", "XPath"], ["tag", "Object Type"], ["ref", "Reference"]];
        if (this.legacy) fields.push(["attribute", "Attribute"]);
        // Modern schemas hide the attribute field, but retain existing values.
        fieldset.dataset.attribute = rule.attribute || "";
        for (const [key, label] of fields) {
            const group = configElement("div", undefined, "form-group");
            const caption = configElement("label", label, "col-sm-5 control-label");
            const column = configElement("div", undefined, "col-sm-7");
            const input = configElement(key === "right" ? "select" : "input", undefined, "form-control");
            input.dataset.field = key;
            if (key === "right") {
                for (const value of ["read", "write", "deny"]) input.append(new Option(value[0].toUpperCase() + value.slice(1), value));
                input.required = true;
            } else input.type = "text";
            input.value = rule[key] || "";
            column.append(input);
            group.append(caption, column);
            fieldset.append(group);
        }
        if (after) after.after(fieldset); else this.rules.append(fieldset);
        this.reindex();
    }

    reindex() {
        Array.from(this.rules.children).forEach((fieldset, index) => {
            fieldset.querySelectorAll("[data-field]").forEach(input => {
                input.name = `role[rules_attributes][${index}][${input.dataset.field}]`;
                input.id = `role_rules_${index}_${input.dataset.field}`;
                input.closest(".form-group").querySelector("label").htmlFor = input.id;
            });
        });
    }

    async save() {
        if (this.fields.disabled) return;
        const rules = Array.from(this.rules.children, fieldset => {
            const rule = { attribute: fieldset.dataset.attribute };
            fieldset.querySelectorAll("[data-field]").forEach(input => { rule[input.dataset.field] = input.value; });
            return rule;
        });
        this.busy(true);
        try {
            const data = await configJSON("/api/cib/acl/role", {
                method: "POST", headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ id: this.id, rules }),
            });
            this.dirty = false;
            this.notify(data.message);
        } catch (error) { this.notify(error.message, true); }
        finally { this.busy(false); }
    }

    async delete() {
        this.busy(true);
        try {
            const data = await configJSON("/api/cib/acl/roles/delete", {
                method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ id: this.id }),
            });
            sessionStorage.setItem("rolesFlash", data.message);
            window.location.href = "/cib/live/roles";
        } catch (error) { this.notify(error.message, true); this.busy(false); }
    }
}

new RoleEditor(document.querySelector(".role-editor"));
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/roles"]').setAttribute("aria-current", "page");
