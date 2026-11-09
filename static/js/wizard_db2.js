const db2Form = document.querySelector("[data-nested-wizard]");
const nestedWizardName = db2Form.dataset.nestedWizard;
const db2Steps = Array.from(db2Form.querySelectorAll(".db2-step"));
let db2Step = 0;

function showDB2Step(index) {
    db2Step = index;
    db2Steps.forEach((step, number) => {
        step.hidden = number !== index;
        step.classList.toggle("active", number === index);
    });
    db2Form.querySelectorAll(".db2-step-link").forEach((link, number) => {
        link.parentElement.classList.toggle("active", number === index);
        link.setAttribute("aria-selected", String(number === index));
    });
    document.getElementById("db2-previous").hidden = index === 0;
    document.getElementById("db2-next").hidden = index === db2Steps.length - 1;
    document.getElementById("db2-verify").hidden = index !== db2Steps.length - 1;
}

function nextDB2Step() {
    for (const input of db2Steps[db2Step].querySelectorAll("input, select")) {
        if (!input.checkValidity()) {
            input.reportValidity();
            return;
        }
    }
    if (db2Step < db2Steps.length - 1) showDB2Step(db2Step + 1);
}

function showDB2Help(event) {
    const row = event.target.closest("[data-param-help]");
    const input = row?.querySelector("[name]");
    if (!input || !input.name.startsWith(`${nestedWizardName}.`)) return;
    const id = input.name.replaceAll(".", "-");
    document.querySelectorAll(".db2-param-help").forEach(section => section.hidden = section.id !== id);
}

db2Form.addEventListener("mouseover", showDB2Help);
db2Form.addEventListener("focusin", showDB2Help);
db2Form.addEventListener("input", event => {
    const input = event.target;
    if (input.dataset.paramType === "resource") input.setCustomValidity(input.value.includes(" ") ? "Must be a valid resource ID" : "");
});
if (db2Steps.length) {
    document.getElementById("db2-next").addEventListener("click", nextDB2Step);
    document.getElementById("db2-previous").addEventListener("click", () => showDB2Step(db2Step - 1));
    // Ruby's wizard disallows skipping validation by clicking step indicators.
    db2Form.querySelectorAll(".db2-step-link").forEach(link => link.addEventListener("click", event => event.preventDefault()));
    for (const group of window.db2WizardData.groups) {
        new WizardAdvancedParameters(document.getElementById(group.containerID), group.parameters, group.values, group.prefix);
    }
    showDB2Step(0);
}
document.querySelectorAll("pre code.crmsh").forEach(block => hljs.highlightBlock(block));
