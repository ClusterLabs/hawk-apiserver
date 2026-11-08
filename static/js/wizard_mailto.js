function showMailtoHelp(event) {
    const row = event.target.closest("[data-param-help]");
    if (!row) return;
    document.querySelectorAll(".mailto-param-help").forEach(section => {
        section.hidden = section.id !== `mailto-help-${row.dataset.paramHelp}`;
    });
}

const mailtoForm = document.getElementById("edit_wizard_mailto");
mailtoForm.addEventListener("mouseover", showMailtoHelp);
mailtoForm.addEventListener("focusin", showMailtoHelp);
const resourceID = document.getElementById("mailto.id");
resourceID?.addEventListener("input", () => {
    resourceID.setCustomValidity(resourceID.value.includes(" ") ? "Must be a valid resource ID" : "");
});

// Use the same crmsh grammar and color theme as the Ruby wizard preview.
document.querySelectorAll("pre code.crmsh").forEach(block => hljs.highlightBlock(block));
