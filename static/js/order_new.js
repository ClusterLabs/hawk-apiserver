let resources = null;
let creatingOrder = false;
let orderChanged = false;

function showFlash(type, message) {
    const alert = document.getElementById("order-error");
    alert.className = `alert alert-${type}`;
    alert.textContent = message.trim();
    alert.hidden = false;
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
        await order.create();
    } catch (err) {
        showFlash("danger", err.message);
    } finally {
        creatingOrder = false;
        button.disabled = !resources;
    }
}

async function initializeOrderForm() {
    try {
        const response = await fetch("/api/cib/order/options/fetch", { method: "POST" });
        if (!response.ok) throw new Error(await response.text() || "Unable to load cluster options.");
        const available = await response.json();
        resources = new ConstraintResources(document.getElementById("order-resources"), available, [], "order");
        document.getElementById("create-order-button").disabled = false;
    } catch (err) {
        showFlash("danger", err.message);
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
    if (orderChanged && !window.confirm("Any changes will be lost - do you wish to proceed?")) event.preventDefault();
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
initializeOrderForm();
