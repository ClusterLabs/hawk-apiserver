const { ResourceID, ResourceAgent, CreateMode = false } = window.resourceData;
let createMode = CreateMode;
let currentResourceAgent = ResourceAgent;
let agentReady = !CreateMode;
let selectionVersion = 0;
let classInitialized = false;
let creatingPrimitive = false;

const url = new URL(window.location.href);
if (url.searchParams.has("flash")) {
    url.searchParams.delete("flash");
    url.searchParams.delete("msg");
    history.replaceState({}, "", url);
}

function selectedResourceAgent() {
    const className = document.getElementById("class-xselect").selectedOption()?.getName();
    const provider = document.getElementById("provider-xselect").selectedOption()?.getName();
    const type = document.getElementById("type-xselect").selectedOption()?.getName();
    return className && type ? [className, provider, type].filter(Boolean).join(":") : "";
}

async function setAgentInfo(agent = currentResourceAgent) {
    const response = await fetch('/api/cib/resource/params/fetch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ResourceID: createMode ? "" : ResourceID, ResourceAgent: agent })
    });
    if (!response.ok) throw new Error(await response.text() || "Unable to load resource agent metadata.");
    const content = await response.json();
    if (agent !== currentResourceAgent) return;
    document.getElementById("agent-shortdesc").textContent = (content.Shortdesc || "").trim();
    const description = document.getElementById("agent-longdesc");
    description.replaceChildren();
    for (const line of (content.Longdesc || "").trim().split(/\n\n+/)) {
        const paragraph = document.createElement("p");
        paragraph.textContent = line;
        description.append(paragraph);
    }
}

function showAgentInfo() {
    if (currentResourceAgent) new RAInfoPopup(createMode ? "" : ResourceID, currentResourceAgent);
}

function primitiveGroups() {
    return ["kvgroup-instance_attributes", "kvgroup-meta_attributes", "kvgroup-operations", "kvgroup-resource_utilizations"]
        .map(id => document.getElementById(id));
}

async function clickApplyCreateButton() {
    if (creatingPrimitive) return;
    if (createMode) {
        const id = document.getElementById("input-primitive-id").value.trim();
        if (!/^[A-Za-z_][A-Za-z0-9_.:-]*$/.test(id)) {
            showFlash("danger", "Enter a Resource ID starting with a letter or underscore, using letters, numbers, underscores, dots, colons or hyphens.");
            return;
        }
        if (!agentReady || !selectedResourceAgent()) {
            showFlash("danger", "Select a resource agent and wait for its fields to load.");
            return;
        }
    }
    const groups = primitiveGroups();
    if (groups.some(group => !group)) return;
    for (const group of groups) {
        const result = group.submitSanityCheck();
        if (!result.ok) { showFlash("danger", result.message); return; }
    }
    creatingPrimitive = true;
    const button = document.getElementById("apply-create-button");
    button.disabled = true;
    try {
        if (createMode) {
            const primitive = new Primitive(
                document.getElementById("input-primitive-id"),
                document.getElementById("class-xselect"),
                document.getElementById("provider-xselect"),
                document.getElementById("type-xselect"),
                groups[0], groups[1], groups[2], groups[3]
            );
            await primitive.create();
        } else {
            await Promise.all(groups.map(group => group.submit()));
            const location = new URL(window.location.href);
            location.searchParams.set("flash", "updated");
            window.location.href = location.toString();
        }
    } catch (err) {
        showFlash("danger", `There was a problem ${createMode ? "creating" : "updating"} the primitive:\n${err.message}`);
    } finally {
        creatingPrimitive = false;
        updateCreateButtonState();
    }
}

async function toggleUpdateCreateMode(enabled) {
    createMode = enabled;
    const input = document.getElementById("input-primitive-id");
    input.readOnly = !enabled;
    if (enabled) input.value += "-1";
    document.getElementById("apply-create-button").textContent = enabled ? "Create" : "Apply";
    document.getElementById("copy-rename-delete-container")?.classList.toggle("hidden", enabled);
    for (const id of ["class-xselect", "provider-xselect", "type-xselect"]) {
        const select = document.getElementById(id);
        if (enabled) select.enableEdit(); else select.disableEdit();
    }
    if (enabled) agentReady = false;
    updateCreateButtonState();
    if (enabled) {
        const groups = primitiveGroups();
        const readiness = await Promise.all(groups.map(group => group?.loaded));
        // Changing the agent while loading replaces these groups.
        if (!createMode || groups.some((group, index) => group !== primitiveGroups()[index])) return;
        agentReady = readiness.every(ready => ready === true);
        updateCreateButtonState();
    }
}

async function changeClass() {
    classInitialized = true;
    const version = ++selectionVersion;
    const classSelect = document.getElementById("class-xselect");
    const provider = document.getElementById("provider-xselect");
    const className = classSelect.selectedOption()?.getName();
    if (createMode) { agentReady = false; updateCreateButtonState(); }
    if (!className) { if (createMode) await changeType(); return; }
    try {
        await provider.reload({ Class: className });
        if (version !== selectionVersion) return;
        const options = provider.getHTML().options;
        provider.visible = options.length > 0;
        if (!provider.getOption(provider.value)) provider.setValue(options[0]?.value || "");
        await changeProvider();
    } catch (err) { showFlash("danger", `Unable to load resource agents: ${err.message}`); }
}

async function changeProvider() {
    const version = ++selectionVersion;
    const className = document.getElementById("class-xselect").selectedOption()?.getName();
    const provider = document.getElementById("provider-xselect").selectedOption()?.getName();
    const type = document.getElementById("type-xselect");
    const previousType = type.value;
    const preserveCopy = !CreateMode && currentResourceAgent === ResourceAgent &&
        [className, provider, previousType].filter(Boolean).join(":") === ResourceAgent;
    if (createMode) { agentReady = false; updateCreateButtonState(); }
    if (!className) { if (createMode) await changeType(); return; }
    try {
        await type.reload({ Class: className, Provider: provider || "" });
        if (version !== selectionVersion) return;
        if (createMode) {
            type.appendBlankOption();
            if (preserveCopy && type.getOption(previousType)) {
                type.setValue(previousType);
                const readiness = await Promise.all(primitiveGroups().map(group => group.loaded));
                if (version !== selectionVersion) return;
                agentReady = readiness.every(ready => ready === true);
                updateCreateButtonState();
                return;
            }
            type.setValue("");
            await changeType();
        }
    } catch (err) { showFlash("danger", `Unable to load resource agent types: ${err.message}`); }
}

async function changeType() {
    if (!createMode) return;
    const version = ++selectionVersion;
    agentReady = false;
    currentResourceAgent = selectedResourceAgent();
    window.resourceData.ResourceAgent = currentResourceAgent;
    const fields = document.getElementById("primitive-fields");
    fields.replaceChildren();
    document.getElementById("agent-shortdesc").textContent = "";
    document.getElementById("agent-longdesc").replaceChildren();
    document.getElementById("agent-info-button").disabled = !currentResourceAgent;
    updateCreateButtonState();
    if (!currentResourceAgent) return;
    const fragment = document.getElementById("primitive-fields-template").content.cloneNode(true);
    fragment.querySelectorAll("[resource-id]").forEach(group => group.setAttribute("resource-id", ""));
    fragment.querySelectorAll("[cib-object]").forEach(group => group.setAttribute("cib-object", ""));
    fragment.querySelectorAll("[resource-agent]").forEach(group => group.setAttribute("resource-agent", currentResourceAgent));
    fragment.querySelectorAll("x-kvgroup").forEach(group => group.setAttribute("create-mode", "true"));
    fields.append(fragment);
    try {
        await setAgentInfo();
        const readiness = await Promise.all(primitiveGroups().map(group => group.loaded));
        if (version !== selectionVersion) return;
        if (readiness.some(ready => ready !== true)) throw new Error("Unable to load resource fields. Select the agent again to retry.");
        agentReady = true;
    } catch (err) { if (version === selectionVersion) showFlash("danger", err.message); }
    updateCreateButtonState();
}

function updateCreateButtonState() {
    const button = document.getElementById("apply-create-button");
    const ready = !createMode || (agentReady && !!selectedResourceAgent());
    button.disabled = creatingPrimitive || !ready;
    button.classList.add("btn-success");
}

// TODO: showFlash is too dirty. Remove it later.
// Use the XAlert directly w/o recreating.
function showFlash(type, message) {
    // reuse existing alert if present, otherwise create one at top of .edit-left
    let alertEl = document.querySelector(".edit-left x-alert[data-flash='js']");
    if (!alertEl) {
        alertEl = document.createElement("x-alert");
        alertEl.dataset.flash = "js";

        const left = document.querySelector(".edit-left");
        if (left) left.prepend(alertEl);
    }

    // Why replace? XAlert renders in connectedCallback()
    // and doesn’t watch attribute changes (#TODO, maybe)
    const newAlert = document.createElement("x-alert");
    newAlert.dataset.flash = "js";
    newAlert.setAttribute("type", type);
    newAlert.setAttribute("message", message);
    newAlert.setAttribute("visible", "true");

    alertEl.replaceWith(newAlert);

    console.trace("showFlash called");
}

const fieldShortdesc = document.getElementById('field-shortdesc');
const fieldLongdesc = document.getElementById('field-longdesc');

/* custom elements x-select and x-operations-kvgroup
 * already have their own listeners.
 * input-primitive-id is only left w/o it's own listener. */
const input = document.getElementById("input-primitive-id");
input.addEventListener("mouseenter", () => {
    fieldShortdesc.textContent = "Resource ID";
    fieldLongdesc.textContent = "Unique identifier for the resource. May not contain spaces.";
});

document.getElementById("class-xselect").addEventListener("change", changeClass);
document.getElementById("provider-xselect").addEventListener("change", changeProvider);
document.getElementById("type-xselect").addEventListener("change", changeType);
Promise.resolve(document.getElementById("class-xselect").loaded).then(ready => {
    if (!ready) { showFlash("danger", "Unable to load resource agent classes."); return; }
    if (!classInitialized) changeClass();
});
if (!CreateMode) setAgentInfo().catch(err => showFlash("danger", err.message));

if (CreateMode) {
    document.getElementById("monitoringMenu").classList.remove("in");
    document.getElementById("configurationMenu").classList.add("in");
    document.querySelector('a[href="/cib/live/resources/types"]').setAttribute("aria-current", "page");
}
