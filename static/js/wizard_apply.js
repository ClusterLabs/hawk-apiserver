function showWizardNotification(type, message) {
    const notification = document.createElement("div");
    notification.className = `alert alert-${type}`;
    notification.textContent = message;
    if (type === "info") {
        const icon = document.createElement("i");
        icon.className = "fa fa-refresh fa-2x health-progress-icon";
        icon.setAttribute("aria-hidden", "true");
        notification.prepend(icon, " ");
    }
    document.querySelector(".notifications").replaceChildren(notification);
}

async function clickApplyWizard(event) {
    event.preventDefault();
    const form = event.currentTarget;
    const button = form.querySelector('[name="submit"]');
    if (button.disabled) return;
    button.disabled = true;
    const actions = form.querySelectorAll(".actions .list-group-item");
    actions.forEach(action => action.classList.add("disabled"));
    showWizardNotification("info", form.dataset.progressMessage || "Collecting information and checking cluster health...");
    try {
        const response = await fetch(form.action, {
            method: "POST",
            body: new URLSearchParams({ pa: form.querySelector('[name="pa"]').value }),
        });
        const result = await response.json();
        if (!response.ok) throw new Error(Array.isArray(result) ? result.join("\n") : result.error || "Unable to apply wizard.");
        actions.forEach(action => {
            action.classList.remove("disabled");
            action.classList.add("list-group-item-success");
        });
        if (result.output && actions.length) {
            const last = actions[actions.length - 1];
            const report = document.createElement("button");
            report.type = "button";
            report.className = "list-group-item btn-primary";
            for (const child of last.childNodes) report.append(child.cloneNode(true));
            report.addEventListener("click", () => {
                const popup = new Popup();
                const output = document.createElement("pre");
                output.className = "health-output";
                output.textContent = result.output;
                popup._appendModalContentChild(output);
            });
            last.replaceChildren(report);
        }
        showWizardNotification("success", "Changes applied successfully.");
    } catch (err) {
        actions[0]?.classList.remove("disabled");
        actions[0]?.classList.add("list-group-item-danger");
        showWizardNotification("danger", err.message);
    }
}

const wizardForm = document.querySelector("[data-wizard-form]");
if (wizardForm.dataset.wizardApply) wizardForm.addEventListener("submit", clickApplyWizard);
wizardForm.querySelector('[data-confirm]').addEventListener("click", event => {
    if (!window.confirm(event.currentTarget.dataset.confirm)) event.preventDefault();
});
wizardForm.querySelector(".previous.disabled")?.addEventListener("click", event => event.preventDefault());
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/wizards"]').setAttribute("aria-current", "page");
