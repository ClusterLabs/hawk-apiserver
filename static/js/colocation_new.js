const colocationID = document.getElementById("colocation-create-form").dataset.colocationId;
let resources = null;
let creatingColocation = false;
let colocationChanged = false;

function showFlash(type, message) {
    const alert = document.getElementById("colocation-error");
    alert.className = `alert alert-${type}`;
    alert.textContent = message.trim();
    alert.hidden = false;
}

async function clickCreateButton(event) {
    event.preventDefault();
    if (!resources || creatingColocation) return;
    const button = document.getElementById("create-colocation-button");
    creatingColocation = true;
    button.disabled = true;
    try {
        const colocation = new Colocation(
            document.getElementById("colocation_id"),
            document.getElementById("colocation_score"),
            document.getElementById("colocation_node_attr"),
            resources
        );
        await colocation.create(Boolean(colocationID));
    } catch (err) {
        showFlash("danger", err.message);
    } finally {
        creatingColocation = false;
        button.disabled = !resources;
    }
}

async function initializeColocationForm() {
    try {
        const response = await fetch("/api/cib/colocation/options/fetch", { method: "POST" });
        if (!response.ok) throw new Error(await response.text() || "Unable to load cluster options.");
        const available = await response.json();
        let entries = [];
        if (colocationID) {
            const response = await fetch("/api/cib/colocation/fetch?id=" + encodeURIComponent(colocationID), { method: "POST" });
            if (!response.ok) throw new Error(await response.text());
            const colocation = await response.json();
            document.getElementById("colocation_id").value = colocation.id;
            document.getElementById("colocation_score").value = colocation.score;
            document.getElementById("colocation_node_attr").value = colocation.node_attr;
            entries = colocation.resources;
        }
        resources = new ConstraintResources(document.getElementById("colocation-resources"), available, entries);
        const url = new URL(window.location.href);
        const flash = url.searchParams.get("flash");
        if (flash === "created" || flash === "updated") {
            showFlash("success", `Constraint ${flash} successfully`);
            url.searchParams.delete("flash");
            history.replaceState({}, "", url);
        }
        document.getElementById("create-colocation-button").disabled = false;
    } catch (err) {
        showFlash("danger", err.message);
    }
}

function showColocationHelp(event) {
    const help = {
        id: ["Constraint ID", "Unique identifier for the constraint. May not contain spaces."],
        score: ["Score", "The score determines the location relationship between the resources."],
        "node-attribute": ["Node Attribute", "Colocate resources by the given node attribute instead of by node name (the default)."],
    };
    const field = event.target.closest("[data-help]");
    if (!field) return;
    const [title, description] = help[field.dataset.help];
    document.getElementById("field-shortdesc").textContent = title;
    document.getElementById("field-longdesc").textContent = description;
}

const form = document.getElementById("colocation-create-form");
form.addEventListener("submit", clickCreateButton);
form.addEventListener("mouseover", showColocationHelp);
form.addEventListener("focusin", showColocationHelp);
function markColocationChanged() {
    colocationChanged = true;
    const revert = document.getElementById("revert-colocation-button");
    if (revert) revert.hidden = false;
}
form.addEventListener("change", markColocationChanged);
form.addEventListener("input", markColocationChanged);
document.getElementById("revert-colocation-button")?.addEventListener("click", () => window.location.reload());
document.getElementById("colocation-back").addEventListener("click", event => {
    if (colocationChanged && !window.confirm("Any changes will be lost - do you wish to proceed?")) event.preventDefault();
});
const presets = document.getElementById("colocation-score-presets");
const presetButton = document.getElementById("colocation_score_btn");
presetButton.addEventListener("click", () => {
    presets.hidden = !presets.hidden;
    presetButton.setAttribute("aria-expanded", String(!presets.hidden));
});
document.querySelectorAll("[data-score]").forEach(preset => preset.addEventListener("click", event => {
    event.preventDefault();
    document.getElementById("colocation_score").value = preset.dataset.score;
    markColocationChanged();
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
initializeColocationForm();
