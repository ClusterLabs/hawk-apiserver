function showResourceWizardHelp(event) {
    const row = event.target.closest("[data-param-help]");
    if (!row) return;
    document.querySelectorAll(".resource-param-help").forEach(section => {
        section.hidden = section.id !== `${resourceWizardName}-help-${row.dataset.paramHelp}`;
    });
}


const resourceWizardForm = document.querySelector("[data-resource-wizard]");
const resourceWizardName = resourceWizardForm.dataset.resourceWizard;
resourceWizardForm.addEventListener("mouseover", showResourceWizardHelp);
resourceWizardForm.addEventListener("focusin", showResourceWizardHelp);
const resourceID = document.getElementById(`${resourceWizardName}.id`);
resourceID?.addEventListener("input", () => {
    resourceID.setCustomValidity(resourceID.value.includes(" ") ? "Must be a valid resource ID" : "");
});
const advanced = document.getElementById("wizard-advanced-fields");
if (advanced) new WizardAdvancedParameters(advanced, window.resourceWizardData.parameters, window.resourceWizardData.values, resourceWizardName);
document.querySelectorAll("pre code.crmsh").forEach(block => hljs.highlightBlock(block));
