let recipients = [];
let recipientCount = 0;
let creatingAlert = false;
let alertReady = false;
const params = new AlertAttributes(document.getElementById("alert-params"), "alert[params]");
const meta = document.getElementById("kvgroup-meta_attributes");

function showFlash(type, message) {
    const alert = document.getElementById("alert-error");
    alert.className = `alert alert-${type}`;
    alert.textContent = message.trim();
    alert.hidden = false;
}

function editRecipient(recipient, editing) {
    new AlertRecipientPopup(recipient, editing, updated => {
        const index = recipients.findIndex(item => item.id === updated.id);
        if (index < 0) recipients.push(updated);
        else recipients[index] = updated;
        renderRecipients();
    });
}

function renderRecipients() {
    const list = document.getElementById("alert-recipients");
    list.replaceChildren();
    for (const recipient of recipients) {
        const row = document.createElement("div");
        row.className = "kvsection-row";
        const label = document.createElement("label");
        label.textContent = "To";
        const control = document.createElement("div");
        control.className = "alert-control input-group";
        const input = document.createElement("input");
        input.className = "form-control";
        input.value = recipient.value;
        input.readOnly = true;
        input.name = `alert[recipients][${recipient.id}][value]`;
        input.setAttribute("aria-label", "To");
        const edit = document.createElement("button");
        edit.type = "button";
        edit.className = "btn btn-default";
        edit.title = "Edit Recipient";
        edit.innerHTML = '<i class="fa fa-pencil" aria-hidden="true"></i>';
        edit.addEventListener("click", () => editRecipient(recipient, true));
        const remove = document.createElement("button");
        remove.type = "button";
        remove.className = "btn btn-default";
        remove.title = "Remove Recipient";
        remove.innerHTML = '<i class="fa fa-minus" aria-hidden="true"></i>';
        remove.addEventListener("click", () => {
            recipients = recipients.filter(item => item.id !== recipient.id);
            renderRecipients();
        });
        control.append(input, edit, remove);
        row.append(label, control);
        list.append(row);
    }
}

function clickCreateButton(event) {
    event.preventDefault();
    if (!alertReady || creatingAlert) return;
    const sanity = meta.submitSanityCheck();
    if (!sanity.ok) { showFlash("danger", sanity.message); return; }
    const alert = new Alert(document.getElementById("input-alert-id"), document.getElementById("input-alert-path"), recipients, params, meta);
    const button = document.getElementById("create-alert-button");
    creatingAlert = true;
    button.disabled = true;
    alert.create().catch(err => {
        showFlash("danger", err.message);
    }).finally(() => {
        creatingAlert = false;
        button.disabled = !alertReady;
    });
}

async function initializeAlertForm() {
    if (await meta.loaded !== true) { showFlash("danger", "Unable to load alert meta attributes."); return; }
    alertReady = true;
    document.getElementById("create-alert-button").disabled = false;
}

document.getElementById("alert-create-form").addEventListener("submit", clickCreateButton);
document.getElementById("add-recipient-button").addEventListener("click", () => {
    recipientCount += 1;
    editRecipient({ id: `alerts-recipient-temporary-id-${recipientCount}`, value: "", params: {}, meta: {} }, false);
});
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/resources/types"]').setAttribute("aria-current", "page");
initializeAlertForm();
