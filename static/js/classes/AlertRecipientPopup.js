class AlertRecipientPopup extends Popup {
    constructor(recipient, editing, onApply) {
        super();
        const form = document.createElement("form");
        form.className = "alert-recipient-form";
        form.innerHTML = `
            <div class="modal-header"><h3 class="modal-title">${editing ? "Edit Recipient" : "Add Recipient"}</h3></div>
            <div class="modal-body">
                <div class="kvsection-row"><label for="recipient-path">Path</label>
                    <input id="recipient-path" name="value" class="form-control alert-control" required></div>
                <details open><summary>Instance Attributes</summary><div class="recipient-params"></div></details>
                <details open><summary>Meta Attributes</summary><div class="recipient-meta"></div></details>
            </div>
            <div class="modal-footer"><button class="btn btn-primary" name="submit" type="submit" disabled>${editing ? "Apply" : "Add"}</button></div>`;
        const path = form.querySelector("[name=value]");
        path.value = recipient.value;
        const params = new AlertAttributes(form.querySelector(".recipient-params"), "recipient[params]", recipient.params);
        const meta = document.createElement("x-kvgroup");
        meta.setAttribute("name", "recipient_meta");
        meta.setAttribute("field-prefix", "recipient[meta]");
        meta.setAttribute("options-api", "/api/cib/alert/meta-attributes/fetch");
        meta.setAttribute("initial-values", JSON.stringify(recipient.meta));
        form.querySelector(".recipient-meta").append(meta);
        this._appendModalContentChild(form);
        Promise.all([meta.loaded, params.loaded]).then(ready => {
            if (ready.every(value => value === true)) form.querySelector("[type=submit]").disabled = false;
        });
        form.addEventListener("submit", event => {
            event.preventDefault();
            const sanity = meta.submitSanityCheck();
            if (!sanity.ok) return;
            onApply({ id: recipient.id, value: path.value, params: params.getFrontendKValues(), meta: meta.getFrontendKValues() });
            this.close();
        });
        path.focus();
    }
}
