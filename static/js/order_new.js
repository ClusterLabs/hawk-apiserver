const orderForm = document.getElementById("order-create-form");
const editingOrder = orderForm.dataset.editing === "true";
const currentOrderID = orderForm.dataset.orderId;
let resources = null;
let creatingOrder = false;
let orderChanged = false;

function showFlash(type, message) {
    const alert = document.createElement("x-alert");
    alert.setAttribute("type", type);
    alert.setAttribute("message", message.trim());
    document.getElementById("order-notifications").replaceChildren(alert);
}

async function clickCreateButton(event) {
    event.preventDefault();
    if (!resources || creatingOrder) return;
    const button = document.getElementById("create-order-button");
    creatingOrder = true;
    button.disabled = true;
    try {
        const order = new Order(
            document.getElementById("order_id"),
            document.getElementById("order_score"),
            document.getElementById("order_score_symmetrical"),
            resources
        );
        if (editingOrder) {
            const result = await order.update();
            if (await initializeOrderForm()) showFlash("success", result.message);
        } else await order.create();
    } catch (err) {
        showFlash("danger", err.message);
    } finally {
        creatingOrder = false;
        button.disabled = !resources;
    }
}

async function initializeOrderForm() {
    try {
        const response = await fetch(editingOrder ? `/api/cib/order/fetch?id=${encodeURIComponent(currentOrderID)}` : "/api/cib/order/options/fetch", { method: editingOrder ? "GET" : "POST", cache: "no-store" });
        if (!response.ok) throw new Error(await response.text() || "Unable to load cluster options.");
        const data = await response.json();
        const available = editingOrder ? data.available : data;
        if (editingOrder) {
            document.getElementById("order_score").value = data.order.score;
            document.getElementById("order_score_symmetrical").value = String(data.order.symmetrical);
        }
        // Linked controls represent parallel sets. Split sequential sets into
        // individual ordered entries so their semantics survive an Apply.
        const entries = editingOrder ? data.order.resources.flatMap(set =>
            set.sequential ? set.resources.map(id => ({ ...set, resources: [id] })) : [set]) : [];
        resources = new ConstraintResources(document.getElementById("order-resources"), available, entries, "order");
        orderChanged = false;
        for (const id of ["rename-order", "delete-order", "revert-order"]) {
            const button = document.getElementById(id);
            if (button) button.disabled = false;
        }
        document.getElementById("create-order-button").disabled = false;
        return true;
    } catch (err) {
        showFlash("danger", err.message);
        return false;
    }
}

function showOrderHelp(event) {
    const help = {
        id: ["Constraint ID", "Unique identifier for the constraint. May not contain spaces."],
    };
    const field = event.target.closest("[data-help]");
    if (!field) return;
    if (!help[field.dataset.help]) return;
    const [title, description] = help[field.dataset.help];
    document.getElementById("field-shortdesc").textContent = title;
    document.getElementById("field-longdesc").textContent = description;
}

const form = document.getElementById("order-create-form");
form.addEventListener("submit", clickCreateButton);
form.addEventListener("mouseover", showOrderHelp);
form.addEventListener("focusin", showOrderHelp);
function markOrderChanged() {
    orderChanged = true;
}
form.addEventListener("change", markOrderChanged);
form.addEventListener("input", markOrderChanged);
document.getElementById("order-back").addEventListener("click", event => {
    if (!orderChanged) return;
    if (editingOrder) { event.preventDefault(); const href = event.currentTarget.href; discardOrderChanges(() => { window.location.href = href; }); }
    else if (!window.confirm("Any changes will be lost - do you wish to proceed?")) event.preventDefault();
});
const presets = document.getElementById("order-score-presets");
const presetButton = document.getElementById("order_score_btn");
presetButton.addEventListener("click", () => {
    presets.hidden = !presets.hidden;
    presetButton.setAttribute("aria-expanded", String(!presets.hidden));
});
document.querySelectorAll("[data-score]").forEach(preset => preset.addEventListener("click", event => {
    event.preventDefault();
    document.getElementById("order_score").value = preset.dataset.score;
    markOrderChanged();
    presets.hidden = true;
    presetButton.setAttribute("aria-expanded", "false");
}));
document.addEventListener("click", event => {
    if (!event.target.closest(".colocation-score")) {
        presets.hidden = true;
        presetButton.setAttribute("aria-expanded", "false");
    }
});
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/constraints/types"]').setAttribute("aria-current", "page");
function discardOrderChanges(onConfirm) {
    if (!orderChanged) { onConfirm(); return; }
    const popup = new Popup();
    const body = configElement("div", "Any changes will be lost - do you wish to proceed?", "modal-body");
    const footer = configElement("div", undefined, "modal-footer");
    const cancel = configElement("button", "Cancel", "btn btn-default");
    const ok = configElement("button", "OK", "btn btn-primary");
    cancel.onclick = () => popup.close();
    ok.onclick = () => { popup.close(); onConfirm(); };
    footer.append(cancel, ok); popup._appendModalContentChild(body); popup._appendModalContentChild(footer);
}
if (editingOrder) {
    document.getElementById("rename-order").addEventListener("click", () => new RenamePopup(currentOrderID));
    document.getElementById("revert-order").addEventListener("click", () => discardOrderChanges(() => window.location.reload()));
    document.getElementById("delete-order").addEventListener("click", () => new DeletePopup(currentOrderID, async () => {
        try {
            const response = await fetch("/api/cib/resource/delete", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(currentOrderID) });
            if (!response.ok) throw new Error(await response.text() || "Unable to delete constraint.");
            sessionStorage.setItem("configFlash", "Constraint deleted successfully");
            window.location.href = "/cib/live/config/edit#constraints";
        } catch (error) { showFlash("danger", error.message); }
    }));
    const url = new URL(window.location.href); const flash = url.searchParams.get("flash");
    if (flash) {
        if (flash === "created") showFlash("success", "Constraint created successfully");
        if (flash === "renamed") showFlash("success", "Constraint renamed successfully");
        if (flash === "error") showFlash("danger", url.searchParams.get("msg") || "Unable to rename constraint.");
        url.searchParams.delete("flash"); url.searchParams.delete("msg"); history.replaceState({}, "", url);
    }
}
initializeOrderForm();
