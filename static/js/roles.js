"use strict";

function showRolesNotification(message, danger = false) {
    const alert = document.createElement("x-alert");
    alert.setAttribute("type", danger ? "danger" : "success");
    alert.setAttribute("message", message);
    alert.setAttribute("visible", "true");
    document.getElementById("roles-notifications").replaceChildren(alert);
}

function roleOperations(role) {
    const group = configElement("div", undefined, "btn-group");
    group.setAttribute("role", "group");
    const edit = configElement("a", undefined, "edit btn btn-default btn-xs");
    edit.href = `/cib/live/roles/${encodeURIComponent(role.id)}/edit`;
    edit.title = "Edit";
    edit.setAttribute("aria-label", `Edit ${role.id}`);
    edit.append(configElement("i", undefined, "fa fa-pencil"));
    const remove = configElement("a", undefined, "delete btn btn-default btn-xs");
    remove.href = `/cib/live/roles/${encodeURIComponent(role.id)}`;
    remove.title = "Delete";
    remove.setAttribute("aria-label", `Delete ${role.id}`);
    remove.append(configElement("i", undefined, "fa fa-trash"));
    let deleting = false;
    remove.addEventListener("click", event => {
        event.preventDefault();
        if (deleting) return;
        new DeletePopup(role.id, async () => {
            deleting = true;
            try {
                const result = await configJSON("/api/cib/acl/roles/delete", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ id: role.id }) });
                showRolesNotification(result.message);
                await rolesTable.load();
            } catch (error) { showRolesNotification(error.message, true); }
            finally { deleting = false; }
        });
    });
    group.append(edit, remove);
    return group;
}

let rolesWarningDismissed = false;
document.querySelector("#roles-acl-warning .close").addEventListener("click", () => {
    rolesWarningDismissed = true;
    document.getElementById("roles-acl-warning").hidden = true;
});

const rolesTable = new ConfigTable(document.querySelector("table.roles"), {
    columns: [["id", "Role ID"]],
    operations: roleOperations,
    pageSize: 50,
    url: "/api/cib/acl/roles",
    rows: data => {
        if (!data.supported) throw new Error("ACL is not supported by the cluster");
        document.getElementById("roles-acl-warning").hidden = data.enabled || rolesWarningDismissed;
        return data.roles;
    },
});
rolesTable.load();
pollClusterStatus(() => rolesTable.load());
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/roles"]').setAttribute("aria-current", "page");
