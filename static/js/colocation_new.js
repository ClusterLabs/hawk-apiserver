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
        await colocation.create();
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
        resources = new ConstraintResources(document.getElementById("colocation-resources"), await response.json());
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
form.addEventListener("change", () => colocationChanged = true);
form.addEventListener("input", () => colocationChanged = true);
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
    colocationChanged = true;
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
