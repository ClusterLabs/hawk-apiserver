"use strict";

class ACLOverview {
    constructor({ kind, column, rows, flash }) {
        this.kind = kind;
        this.base = statusCIBBase();
        this.api = window.batchConfig ? `${window.batchConfig.api}/acl` : "/api/cib/acl";
        this.notifications = document.getElementById("roles-notifications");
        this.warning = document.getElementById("roles-acl-warning");
        this.warningDismissed = false;
        this.warning.querySelector(".close").addEventListener("click", () => {
            this.warningDismissed = true;
            this.warning.hidden = true;
        });
        this.table = new ConfigTable(document.querySelector(`table.${kind}`), {
            columns: [["id", column]],
            operations: record => this.operations(record),
            pageSize: 50,
            url: `${this.api}/${kind}`,
            rows: data => {
                if (!data.supported) throw new Error("ACL is not supported by the cluster");
                this.warning.hidden = data.enabled || this.warningDismissed;
                return data[rows];
            },
        });
        const message = sessionStorage.getItem(flash);
        if (message) {
            sessionStorage.removeItem(flash);
            this.notify(message);
        }
        this.table.load();
        if (window.hawkBatch) {
            window.updateClusterStatus = () => this.table.load();
            window.hawkBatch.start();
        } else pollClusterStatus(() => this.table.load());
        document.getElementById("monitoringMenu").classList.remove("in");
        document.getElementById("configurationMenu").classList.add("in");
        document.querySelector(`a[href="${this.base}/${kind}"]`).setAttribute("aria-current", "page");
    }

    notify(message, danger = false) {
        const alert = document.createElement("x-alert");
        alert.setAttribute("type", danger ? "danger" : "success");
        alert.setAttribute("message", message);
        alert.setAttribute("visible", "true");
        this.notifications.replaceChildren(alert);
    }

    operations(record) {
        const group = configElement("div", undefined, "btn-group");
        group.setAttribute("role", "group");
        const path = `${this.base}/${this.kind}/${encodeURIComponent(record.id)}`;
        const edit = configElement("a", undefined, "edit btn btn-default btn-xs");
        edit.href = `${path}/edit`;
        edit.title = "Edit";
        edit.setAttribute("aria-label", `Edit ${record.id}`);
        edit.append(configElement("i", undefined, "fa fa-pencil"));
        const remove = configElement("a", undefined, "delete btn btn-default btn-xs");
        remove.href = path;
        remove.title = "Delete";
        remove.setAttribute("aria-label", `Delete ${record.id}`);
        remove.append(configElement("i", undefined, "fa fa-trash"));
        let deleting = false;
        remove.addEventListener("click", event => {
            event.preventDefault();
            if (deleting) return;
            new DeletePopup(record.id, async () => {
                deleting = true;
                try {
                    const result = await configJSON(`${this.api}/${this.kind}/delete`, {
                        method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ id: record.id }),
                    });
                    this.notify(result.message);
                    await this.table.load();
                } catch (error) { this.notify(error.message, true); }
                finally { deleting = false; }
            });
        });
        group.append(edit, remove);
        return group;
    }
}
