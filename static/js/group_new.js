let groupReady = false;
let creatingGroup = false;
let draggedGroupChild = null;

function showGroupFlash(message) {
    const alert = document.createElement("x-alert");
    alert.id = "group-alert";
    alert.setAttribute("type", "danger");
    alert.setAttribute("message", message);
    alert.setAttribute("visible", "true");
    document.getElementById("group-alert").replaceWith(alert);
}

function updateGroupCreateButton() {
    document.getElementById("create-group-button").disabled = !groupReady || creatingGroup;
}

async function loadGroupChildren() {
    const list = document.getElementById("group-children");
    const response = await fetch("/api/cib/group/children/fetch", { method: "POST" });
    if (!response.ok) throw new Error(await response.text() || "Unable to load group children.");
    const children = await response.json();
    list.replaceChildren();
    for (const id of children) {
        const row = document.createElement("li");
        row.className = "list-group-item";
        const checkbox = document.createElement("input");
        checkbox.type = "checkbox";
        checkbox.name = "group[children][]";
        checkbox.value = id;
        checkbox.setAttribute("aria-label", id);
        const link = document.createElement("a");
        link.href = `/cib/live/resources/${encodeURIComponent(id)}/edit`;
        link.textContent = id;
        const handle = document.createElement("i");
        handle.className = "fas fa-bars group-child-handle";
        handle.tabIndex = 0;
        handle.setAttribute("role", "button");
        handle.setAttribute("aria-label", `Reorder ${id}`);
        handle.addEventListener("pointerdown", () => { row.draggable = true; });
        handle.addEventListener("pointerup", () => { row.draggable = false; });
        handle.addEventListener("keydown", event => {
            if (creatingGroup) return;
            if (event.key === "ArrowUp" && row.previousElementSibling) {
                event.preventDefault();
                list.insertBefore(row, row.previousElementSibling);
                handle.focus();
            } else if (event.key === "ArrowDown" && row.nextElementSibling) {
                event.preventDefault();
                list.insertBefore(row.nextElementSibling, row);
                handle.focus();
            }
        });
        row.addEventListener("dragstart", event => {
            if (creatingGroup || !row.draggable) { event.preventDefault(); return; }
            draggedGroupChild = row;
            row.classList.add("dragging");
            event.dataTransfer.setData("text/plain", id);
            event.dataTransfer.effectAllowed = "move";
        });
        row.addEventListener("dragover", event => {
            if (!draggedGroupChild || draggedGroupChild === row) return;
            event.preventDefault();
            const after = event.clientY > row.getBoundingClientRect().top + row.offsetHeight / 2;
            list.insertBefore(draggedGroupChild, after ? row.nextSibling : row);
        });
        row.addEventListener("drop", event => event.preventDefault());
        row.addEventListener("dragend", () => {
            row.classList.remove("dragging");
            row.draggable = false;
            draggedGroupChild = null;
        });
        row.append(checkbox, link, handle);
        list.append(row);
    }
    if (!children.length) {
        const empty = document.createElement("li");
        empty.className = "list-group-item text-warning";
        empty.textContent = "No children available";
        list.append(empty);
    }
    list.setAttribute("aria-busy", "false");
    return children.length > 0;
}

async function clickCreateGroupButton(event) {
    event.preventDefault();
    if (!groupReady || creatingGroup) return;
    const id = document.getElementById("input-group-id");
    if (!/^[a-zA-Z0-9_.-]+$/.test(id.value.trim())) {
        showGroupFlash(id.value.trim() ? "Invalid ID" : "ID is required");
        return;
    }
    const list = document.getElementById("group-children");
    if (!list.querySelector('input[type="checkbox"]:checked')) {
        showGroupFlash("Group cannot be empty, expected at least one child");
        return;
    }
    const meta = document.getElementById("kvgroup-meta_attributes");
    const sanity = meta.submitSanityCheck();
    if (!sanity.ok) { showGroupFlash(sanity.message); return; }
    creatingGroup = true;
    updateGroupCreateButton();
    try {
        await new Group(id, list, meta).create();
    } catch (err) {
        showGroupFlash(err.message);
    } finally {
        creatingGroup = false;
        updateGroupCreateButton();
    }
}

document.getElementById("group-create-form").addEventListener("submit", clickCreateGroupButton);
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/resources/types"]').setAttribute("aria-current", "page");

async function initializeGroupForm() {
    try {
        const [hasChildren, metaReady] = await Promise.all([
            loadGroupChildren(), document.getElementById("kvgroup-meta_attributes").loaded,
        ]);
        if (metaReady !== true) throw new Error("Unable to load group meta attributes.");
        groupReady = hasChildren;
        updateGroupCreateButton();
    } catch (err) {
        document.getElementById("group-children").setAttribute("aria-busy", "false");
        showGroupFlash(err.message);
    }
}

initializeGroupForm();
