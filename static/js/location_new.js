const locationID = document.getElementById("location-create-form").dataset.locationId;
let locationReady = false;
let creatingLocation = false;
const rules = new LocationRules(document.getElementById("location-rules"));

function showFlash(type, message) {
    const alert = document.getElementById("location-error");
    alert.className = `alert alert-${type}`;
    alert.textContent = message.trim();
    alert.hidden = false;
}

function updateFromSimple() {
    const score = document.getElementById("simple-score").value;
    const node = document.getElementById("simple-node").value;
    if (!score) return;
    rules.setValues([{ score, role: "", operator: "and", expressions: node ? [{ value: node, attribute: "#uname", operation: "eq", kind: "uname" }] : [] }]);
}

function clickCreateButton(event) {
    event.preventDefault();
    if (!locationReady || creatingLocation) return;
    const location = new Location(document.getElementById("input-location-id"), document.getElementById("location_resources"), document.getElementById("location_discovery"), rules);
    const button = document.getElementById("create-location-button");
    creatingLocation = true;
    button.disabled = true;
    location.create(Boolean(locationID)).catch(err => showFlash("danger", err.message)).finally(() => {
        creatingLocation = false;
        button.disabled = !locationReady;
    });
}

async function initializeLocationForm() {
    try {
        const response = await fetch("/api/cib/location/options/fetch", { method: "POST" });
        if (!response.ok) throw new Error(await response.text() || "Unable to load cluster options.");
        const options = await response.json();
        for (const id of options.resources) document.getElementById("location_resources").add(new Option(id, id));
        for (const name of options.nodes) document.getElementById("simple-node").add(new Option(name, name));
        if (locationID) {
            const response = await fetch("/api/cib/location/fetch?id=" + encodeURIComponent(locationID), { method: "POST" });
            if (!response.ok) throw new Error(await response.text());
            const location = await response.json();
            document.getElementById("input-location-id").value = location.id;
            for (const option of document.getElementById("location_resources").options) option.selected = location.resources.includes(option.value);
            document.getElementById("location_discovery").value = location.discovery;
            rules.setValues(location.rules);
            const rule = location.rules[0];
            const expression = rule?.expressions?.[0];
            const simple = location.rules.length === 1 && rule.expressions.length === 1 && !rule.role && expression.attribute === "#uname" && expression.operation === "eq" && expression.value;
            if (simple) {
                document.getElementById("simple-score").value = rule.score;
                const nodes = document.getElementById("simple-node");
                if (!Array.from(nodes.options).some(option => option.value === expression.value)) nodes.add(new Option(expression.value, expression.value));
                nodes.value = expression.value;
            }
            if (!simple || location.discovery) document.querySelector('[data-location-tab="locadvanced"]').click();
        }
        const flash = new URL(window.location.href).searchParams.get("flash");
        if (flash === "created" || flash === "updated") {
            showFlash("success", `Constraint ${flash} successfully`);
            const url = new URL(window.location.href);
            url.searchParams.delete("flash");
            history.replaceState({}, "", url);
        }
        locationReady = true;
        document.getElementById("create-location-button").disabled = false;
    } catch (err) {
        showFlash("danger", err.message);
    }
}

document.getElementById("location-create-form").addEventListener("submit", clickCreateButton);
for (const id of ["simple-score", "simple-node"]) document.getElementById(id).addEventListener("change", updateFromSimple);
document.querySelectorAll("[data-score]").forEach(button => button.addEventListener("click", event => {
    event.preventDefault();
    document.getElementById("simple-score").value = button.dataset.score;
    document.getElementById("score-presets").hidden = true;
    document.getElementById("simple-score-btn").setAttribute("aria-expanded", "false");
    updateFromSimple();
}));
document.getElementById("simple-score-btn").addEventListener("click", () => {
    const menu = document.getElementById("score-presets");
    menu.hidden = !menu.hidden;
    document.getElementById("simple-score-btn").setAttribute("aria-expanded", String(!menu.hidden));
});
document.querySelectorAll("[data-location-tab]").forEach(tab => tab.addEventListener("click", event => {
    event.preventDefault();
    document.querySelectorAll("[data-location-tab]").forEach(item => {
        const active = item === tab;
        item.parentElement.classList.toggle("active", active);
        item.setAttribute("aria-selected", String(active));
        document.getElementById(item.dataset.locationTab).hidden = !active;
    });
}));
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/constraints/types"]').setAttribute("aria-current", "page");
initializeLocationForm();

const locationHelp = {
    "Constraint ID": "Unique identifier for the constraint. May not contain spaces.",
    Resources: "Resources to apply the constraint to.",
    Score: 'Positive values indicate the resources should run on this node. Negative values indicate the resources should not run on this node. Values of +/- INFINITY change "should"/"should not" to "must"/"must not".',
    Node: "Name of a node in the cluster.",
    "Resource Discovery": "Controls resource discovery for the specified resource on nodes covered by the constraint. always: Always perform resource discovery (default). never: Never perform resource discovery for the specified resource on this node. This option should generally be used with a -INFINITY score. exclusive: Only perform resource discovery for the specified resource on this node.",
    Role: "Limits the rule to apply only when the resource is in the specified role.",
    "Boolean Operator": "How to combine the result of multiple expression objects. Allowed values are and and or.",
    Expression: "Each rule can contain a number of expressions. The results of the expressions are combined based on the rule's boolean operator.",
};

const locationDefaults = {
    Score: "INFINITY",
    "Resource Discovery": "always",
    Role: "started",
    "Boolean Operator": "and",
};

function showLocationHelp(event) {
    const row = event.target.closest(".kvsection-row, .location-expression");
    if (!row) return;
    const name = row.classList.contains("location-expression") ? "Expression" : row.querySelector("label")?.textContent.trim();
    if (!locationHelp[name]) return;
    document.getElementById("field-shortdesc").textContent = name;
    document.getElementById("field-longdesc").textContent = locationHelp[name];
    const defaultValue = locationDefaults[name];
    document.getElementById("field-default").hidden = !defaultValue;
    document.getElementById("field-default-value").textContent = defaultValue || "";
}

document.getElementById("location-create-form").addEventListener("mouseover", showLocationHelp);
document.getElementById("location-create-form").addEventListener("focusin", showLocationHelp);

document.getElementById("revert-location-button")?.addEventListener("click", () => window.location.reload());
