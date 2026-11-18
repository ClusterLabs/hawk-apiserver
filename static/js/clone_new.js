/* This const is a glue code. The thing is that e.g. the {{ .ResourceID }}
 * is not processed by Go if it's outside in the partial template.
 */
const { ResourceID, ResourceAgent } = window.resourceData;

console.log("clone_new.js loaded");

// remove the ?flash=...
// #TODO: later, when there is only Go and no RoR it's better to stop using ?flash
const url = new URL(window.location.href);
if (url.searchParams.has("flash")) {
    url.searchParams.delete("flash");
    url.searchParams.delete("msg");
    history.replaceState({}, "", url);
}

function clickCreateButton() {
    // sanity checks
    const cloneNameInput = document.getElementById("input-clone-id");
    const cloneName = (cloneNameInput?.value || "").trim();

    if (!cloneName) {
        showFlash("danger", "Please enter a Clone ID before creating a clone.");
        return;
    }

    const childResourceXSelect = document.getElementById("child-resource-xselect");
    const childResourceName = (childResourceXSelect?.value || "").trim();

    if (!childResourceName) {
        showFlash("danger", "Please select a child resource before creating a clone.");
        return;
    }

    const metaAttributes = document.getElementById("kvgroup-meta_attributes");
    const result = metaAttributes.submitSanityCheck();
    if (!result.ok) {
        showFlash("danger", result.message);
        return;
    }

    const clone = new Clone(
        document.getElementById("input-clone-id"),
        document.getElementById("child-resource-xselect"),
        metaAttributes
    );

    clone.create().catch(err => {
        console.error("Create error:", err);
        showFlash("danger", `There was a problem creating the clone:\n${err.message}`);
    });
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
 * input-clone-id is only left w/o it's own listener. */
const input = document.getElementById("input-clone-id");
input.addEventListener("mouseenter", () => {
    fieldShortdesc.textContent = "Clone ID";
    fieldLongdesc.textContent = "Unique identifier for the clone resource. May not contain spaces.";
});

// Keep the shared sidebar and batch controls in the selected CIB context.
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector(`a[href="${statusCIBBase()}/resources/types"]`)?.setAttribute("aria-current", "page");
if (window.hawkBatch) window.hawkBatch.start();
