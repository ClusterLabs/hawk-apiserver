/* This const is a glue code. The thing is that e.g. the {{ .ResourceID }}
 * is not processed by Go if it's outside in the partial template.
 */
const { ResourceID, ResourceAgent } = window.resourceData;

console.log("primitive_edit.js loaded");

// remove the ?flash=...
// #TODO: later, when there is only Go and no RoR it's better to stop using ?flash
const url = new URL(window.location.href);
if (url.searchParams.has("flash")) {
  url.searchParams.delete("flash");
  url.searchParams.delete("msg");
  history.replaceState({}, "", url);
}

function setAgentInfo()
{
  const agentShortdesc = document.getElementById('agent-shortdesc');
  const agentLongdesc = document.getElementById('agent-longdesc');

  // copy-paste from #buildParametersTable()
  fetch('/api/data-interface/fetch-resource-params', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ResourceID, ResourceAgent})
  })
    .then(async res => {
      if (!res.ok) throw new Error(await res.text() || "Unknown error");
      return res.json();
    })
    .then(content => {
      agentShortdesc.textContent = content.Shortdesc.trim();
      const contentOptions = content.Options || [];

      // Split Longdesc into paragraphs by empty lines or single newline
      const paragraphs = content.Longdesc.trim().split(/\n\n+/);

      paragraphs.forEach(line => {
        const p = document.createElement("p");
        p.textContent = line;           // safe, no HTML injection
        agentLongdesc.appendChild(p);
      });
    })
    .catch(err => console.error("Failed to init field-shortdesc and field-longdesc", err));
}

setAgentInfo();

function clickApplyCreateButton(applyButton) {
  const isCreateMode = applyButton.textContent === "Create";

  // sanity check
  if (isCreateMode) {
    const typeXSelect = document.getElementById("type-xselect");
    const typeValue = (typeXSelect?.value || "").trim();

    if (!typeValue) {
      showFlash("danger", "Please select a resource agent Type before creating a primitive.");
      return;
    }
  }

  const params = document.getElementById("kvgroup-instance_attributes");
  const metaAttributes = document.getElementById("kvgroup-meta_attributes");
  const operations = document.getElementById("kvgroup-operations");
  const utilizations = document.getElementById("kvgroup-resource_utilizations");

  for (const group of [params, metaAttributes, operations, utilizations]) {
    const result = group.submitSanityCheck();
    if (!result.ok) {
      showFlash("danger", result.message);
      return;
    }
  }

  if (!isCreateMode) {
    Promise.all([
      params.submit(),
      metaAttributes.submit(),
      operations.submit(),
      utilizations.submit(),
    ])
      .then(() => {
        // refresh the page after submitting all 3
        const url = new URL(window.location.href);
        url.searchParams.set("flash", "updated");

        // wait 1 sec and update the page.
        setTimeout(() => {
          window.location.href = url.toString();
        }, 1000);
      })
      .catch(err => {
        // TODO?: refresh the page even after fail
        console.error("Apply failed:", err);
        showFlash("danger", `There was a problem updating the primitive:\n${err.message}`);
      });
    return;
  }

  // isCreateMode (copying resource)
  const primitive = new Primitive(
    document.getElementById("input-resource-id"),
    document.getElementById("class-xselect"),
    document.getElementById("provider-xselect"),
    document.getElementById("type-xselect"),
    params,
    metaAttributes,
    operations
  );

  primitive.create();
}

function toggleUpdateCreateMode(createMode) {
  const resourceIdInput = document.getElementById('input-resource-id');
  const applyButton = document.getElementById('apply-create-button');
  const classXSelect = document.getElementById("class-xselect");
  const providerXSelect = document.getElementById("provider-xselect");
  const typeXSelect = document.getElementById("type-xselect");
  const copyRenameDeleteContainer = document.getElementById("copy-rename-delete-container");

  if (createMode) {
    resourceIdInput.value += "-1"; // it's concatenation
    resourceIdInput.readOnly = false;
    applyButton.textContent = "Create";

    if (copyRenameDeleteContainer) {
      copyRenameDeleteContainer.style.display = "none";
    }

    classXSelect.enableEdit();
    providerXSelect.enableEdit();
    typeXSelect.enableEdit();
    updateCreateButtonState();
  } else {
    resourceIdInput.readOnly = true;
    applyButton.textContent = "Apply";
    applyButton.disabled = false; // unnecessary, but to be sure
    applyButton.classList.add("btn-success");

    classXSelect.disableEdit();
    providerXSelect.disableEdit();
    typeXSelect.disableEdit();
  }
}

function changeClass() {
  const classXSelect = document.getElementById("class-xselect");
  const providerXSelect = document.getElementById("provider-xselect");

  const selected = classXSelect.selectedOption();
  if (!selected) return;

  const className = selected.getName();
  const newArgs = { Class: className };

  providerXSelect.setAttribute("api-args", JSON.stringify(newArgs));

  const btn = document.getElementById('apply-create-button');
  if (btn.textContent === "Create") {
    btn.disabled = true;
    btn.classList.remove("btn-success");
  }
  /* Manually trigger the cascade (it's a #WORKAROUND,
   * The changeProvider wont trigger on the page load). */
  providerXSelect.reload(newArgs).then(() => {
    changeProvider();
  });
}

function changeProvider() {
  const providerXSelect = document.getElementById("provider-xselect");
  const classXSelect = document.getElementById("class-xselect");
  const typeXSelect = document.getElementById("type-xselect");

  if (!providerXSelect || !classXSelect || !typeXSelect) {
    console.warn("One of the selects is not found");
    return;
  }

  const classOption = classXSelect.selectedOption();
  const providerOption = providerXSelect.selectedOption();

  if (!classOption) {
    console.warn("Class not selected");
    return;
  }

  const args = { Class: classOption.getName() };
  if (providerOption) {
    args.Provider = providerOption.getName();
  }

  //typeXSelect.reload(args);
  typeXSelect.reload(args).then(updateCreateButtonState);
}

function updateCreateButtonState() {
  const btn = document.getElementById('apply-create-button');
  const type = document.getElementById('type-xselect');
  if (btn.textContent !== "Create") return;

  //const ready = (type.value && String(type.value).trim() !== "");
  const opt = type.selectedOption?.();
  const ready = !!(opt && String(opt.getName()).trim() !== "");

  btn.disabled = !ready;
  btn.classList.toggle("btn-success", ready);
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
 * input-resource-id is only left w/o it's own listener. */
const input = document.getElementById("input-resource-id");
input.addEventListener("mouseenter", () => {
  fieldShortdesc.textContent = "Resource ID";
  fieldLongdesc.textContent = "Unique identifier for the resource. May not contain spaces.";
});
