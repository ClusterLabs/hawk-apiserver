let tagReady = false;
let creatingTag = false;

function showFlash(type, message) {
    const alert = document.getElementById("tag-alert");
    alert.className = `alert alert-${type}`;
    alert.textContent = message.trim();
    alert.hidden = false;
}

function clickCreateButton(event) {
    event.preventDefault();
    if (!tagReady || creatingTag) return;
    const id = document.getElementById("input-tag-id");
    if (!id.value.trim()) {
        showFlash("danger", "ID is required");
        return;
    }
    const objects = document.getElementById("tag-objects");
    if (!objects.querySelector('input:checked')) {
        showFlash("danger", "No Tag resources specified");
        return;
    }
    const button = document.getElementById("create-tag-button");
    creatingTag = true;
    button.disabled = true;
    new Tag(id, objects).create().catch(err => {
        showFlash("danger", err.message);
    }).finally(() => {
        creatingTag = false;
        button.disabled = !tagReady;
    });
}

async function initializeTagForm() {
    const list = document.getElementById("tag-objects");
    try {
        const response = await fetch("/api/cib/tag/objects/fetch", { method: "POST" });
        if (!response.ok) throw new Error(await response.text() || "Unable to load objects.");
        const objects = await response.json();
        list.replaceChildren();
        for (const id of objects) {
            const row = document.createElement("li");
            row.className = "list-group-item";
            const checkbox = document.createElement("input");
            checkbox.type = "checkbox";
            checkbox.name = "tag[refs][]";
            checkbox.value = id;
            checkbox.setAttribute("aria-label", id);
            row.append(checkbox, document.createTextNode(id));
            list.append(row);
        }
        if (!objects.length) {
            const empty = document.createElement("li");
            empty.className = "list-group-item text-warning";
            empty.textContent = "No objects available";
            list.append(empty);
        }
        tagReady = true;
        document.getElementById("create-tag-button").disabled = false;
    } catch (err) {
        showFlash("danger", err.message);
    } finally {
        list.setAttribute("aria-busy", "false");
    }
}

document.getElementById("tag-create-form").addEventListener("submit", clickCreateButton);
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/resources/types"]').setAttribute("aria-current", "page");
initializeTagForm();
