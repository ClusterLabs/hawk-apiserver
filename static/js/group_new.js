const groupForm = document.getElementById("group-create-form");
const editingGroup = groupForm.dataset.editing === "true";
const currentGroupID = groupForm.dataset.groupId;
let groupDirty = false;
let groupReady = false;
let creatingGroup = false;
let draggedGroupChild = null;

function showGroupFlash(message, type = "danger") {
    const alert = document.createElement("x-alert");
    alert.id = "group-alert";
    alert.setAttribute("type", type);
    alert.setAttribute("message", message);
    alert.setAttribute("visible", "true");
    document.getElementById("group-alert").replaceWith(alert);
}

function updateGroupCreateButton() {
    const disabled = !groupReady || creatingGroup;
    document.getElementById("create-group-button").disabled = disabled;
    groupForm.setAttribute("aria-busy", String(disabled));
    groupForm.querySelectorAll("input, select, x-kvgroup button").forEach(input => { input.disabled = creatingGroup; });
    for (const id of ["rename-group", "delete-group", "revert-group"]) {
        const button = document.getElementById(id);
        if (button) button.disabled = disabled;
    }
}

async function loadGroupChildren(data = null) {
    const list = document.getElementById("group-children");
    let children;
    if (data) children = data.available;
    else {
        const response = await fetch("/api/cib/group/children/fetch", { method: "POST" });
        if (!response.ok) throw new Error(await response.text() || "Unable to load group children.");
        children = await response.json();
    }
    const selected = new Set(data ? data.group.children : []);
    list.replaceChildren();
    for (const id of children) {
        const row = document.createElement("li");
        row.className = "list-group-item";
        const checkbox = document.createElement("input");
        checkbox.type = "checkbox";
        checkbox.name = "group[children][]";
        checkbox.value = id;
        checkbox.checked = selected.has(id);
        checkbox.setAttribute("aria-label", id);
        const link = document.createElement("a");
        link.href = `/cib/live/primitives/${encodeURIComponent(id)}/edit`;
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
                groupDirty = true;
            } else if (event.key === "ArrowDown" && row.nextElementSibling) {
                event.preventDefault();
                list.insertBefore(row.nextElementSibling, row);
                handle.focus();
                groupDirty = true;
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
            groupDirty = true;
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
        const group = new Group(id, list, meta);
        if (editingGroup) {
            const result = await group.update();
            if (await initializeGroupForm()) showGroupFlash(result.message, "success");
        } else await group.create();
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
        let data = null;
        if (editingGroup) {
            const response = await fetch(`/api/cib/group/fetch?id=${encodeURIComponent(currentGroupID)}`, { cache: "no-store" });
            if (!response.ok) throw new Error(await response.text() || "Unable to load group.");
            data = await response.json();
            const previous = document.getElementById("kvgroup-meta_attributes");
            const meta = document.createElement("x-kvgroup");
            for (const attr of previous.attributes) meta.setAttribute(attr.name, attr.value);
            meta.setAttribute("initial-values", JSON.stringify(data.group.meta));
            previous.replaceWith(meta);
        }
        const [hasChildren, metaReady] = await Promise.all([
            loadGroupChildren(data), document.getElementById("kvgroup-meta_attributes").loaded,
        ]);
        if (metaReady !== true) throw new Error("Unable to load group meta attributes.");
        groupReady = hasChildren;
        groupDirty = false;
        updateGroupCreateButton();
        return true;
    } catch (err) {
        groupReady = false;
        updateGroupCreateButton();
        document.getElementById("group-children").setAttribute("aria-busy", "false");
        showGroupFlash(err.message);
        return false;
    }
}

function discardGroupChanges(onConfirm) {
    if (!groupDirty) { onConfirm(); return; }
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

if (editingGroup) {
    groupForm.addEventListener("input", () => { groupDirty = true; });
    groupForm.addEventListener("change", () => { groupDirty = true; });
    groupForm.addEventListener("click", event => { if (event.target.closest("x-kvgroup button")) groupDirty = true; });
    document.getElementById("rename-group").addEventListener("click", () => new RenamePopup(currentGroupID));
    document.getElementById("revert-group").addEventListener("click", () => discardGroupChanges(() => window.location.reload()));
    const back = groupForm.querySelector("a.back");
    back.addEventListener("click", event => { if (groupDirty) { event.preventDefault(); discardGroupChanges(() => { window.location.href = back.href; }); } });
    document.getElementById("delete-group").addEventListener("click", () => new DeletePopup(currentGroupID, async () => {
        creatingGroup = true;
        updateGroupCreateButton();
        try {
            const response = await fetch("/api/cib/resource/delete", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(currentGroupID) });
            if (!response.ok) throw new Error(await response.text() || "Unable to delete group.");
            sessionStorage.setItem("configFlash", "Group deleted successfully");
            window.location.href = "/cib/live/config/edit";
        } catch (error) { showGroupFlash(error.message); creatingGroup = false; updateGroupCreateButton(); }
    }));
    const url = new URL(window.location.href);
    const flash = url.searchParams.get("flash");
    if (flash) {
        if (flash === "created") showGroupFlash("Group created successfully", "success");
        if (flash === "child-created") showGroupFlash("Primitive created successfully", "success");
        if (flash === "renamed") showGroupFlash("Group renamed successfully", "success");
        if (flash === "error") showGroupFlash(url.searchParams.get("msg") || "Unable to rename group.");
        url.searchParams.delete("flash"); url.searchParams.delete("msg");
        history.replaceState({}, "", url);
    }
}
initializeGroupForm();
