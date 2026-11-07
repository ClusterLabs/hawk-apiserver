function showWizardHelp(id) {
    document.querySelectorAll(".wizard-help").forEach(section => {
        section.hidden = section.id !== id;
    });
}

function toggleWizardCategory(button) {
    const expanded = button.getAttribute("aria-expanded") !== "true";
    document.querySelectorAll(".wizard-category").forEach(category => {
        const active = category === button && expanded;
        category.setAttribute("aria-expanded", String(active));
        const list = document.getElementById(category.getAttribute("aria-controls"));
        list.hidden = !active;
        list.classList.toggle("in", active);
        category.querySelector(".wizard-chevron").className = `fa fa-chevron-${active ? "up" : "down"} wizard-chevron`;
    });
}

document.querySelectorAll(".wizard-category").forEach(button => {
    button.addEventListener("click", event => {
        // Handle the Ruby-compatible collapse markup here, without Bootstrap
        // also toggling the same category through its delegated handler.
        event.preventDefault();
        event.stopPropagation();
        toggleWizardCategory(button);
    });
    button.addEventListener("keydown", event => {
        if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            button.click();
        }
    });
});
document.querySelectorAll("[data-wizard-help]").forEach(link => {
    link.addEventListener("mouseenter", () => showWizardHelp(link.dataset.wizardHelp));
    link.addEventListener("focus", () => showWizardHelp(link.dataset.wizardHelp));
});
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/wizards"]').setAttribute("aria-current", "page");
