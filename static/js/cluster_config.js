"use strict";

const clusterConfigForm = document.getElementById("edit_crm_config");
const clusterConfigGroups = Array.from(clusterConfigForm.querySelectorAll("x-kvgroup"));
const clusterConfigApply = clusterConfigForm.querySelector('[name="submit"]');

function showClusterConfigNotification(message, danger = false) {
    const alert = document.createElement("x-alert");
    alert.setAttribute("type", danger ? "danger" : "success");
    alert.setAttribute("message", message);
    alert.setAttribute("visible", "true");
    document.getElementById("cluster-config-notifications").replaceChildren(alert);
}

async function initializeClusterConfig() {
    const loaded = await Promise.all(clusterConfigGroups.map(group => group.loaded));
    if (loaded.every(Boolean)) clusterConfigApply.disabled = false;
    else showClusterConfigNotification("Unable to load cluster configuration. Reload the page to retry.", true);
}

clusterConfigForm.addEventListener("submit", async event => {
    event.preventDefault();
    if (clusterConfigApply.disabled) return;
    clusterConfigApply.disabled = true;
    try {
        const values = Object.fromEntries(clusterConfigGroups.map(group => [group.getAttribute("name"), group.getFrontendKValues()]));
        const response = await fetch(clusterConfigForm.action, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ values }),
        });
        const result = await response.json();
        if (!response.ok || result.error) throw new Error(result.error || "Unable to save cluster configuration.");
        showClusterConfigNotification(result.message);
    } catch (error) { showClusterConfigNotification(error.message, true); }
    finally { clusterConfigApply.disabled = false; }
});
clusterConfigForm.querySelector('[name="revert"]').addEventListener("click", () => {
    if (window.confirm("Any changes will be lost - do you wish to proceed?")) window.location.reload();
});
clusterConfigForm.querySelectorAll(".config-section-toggle").forEach(button => button.addEventListener("click", () => {
    const expanded = button.getAttribute("aria-expanded") !== "true";
    button.setAttribute("aria-expanded", String(expanded));
    document.getElementById(button.getAttribute("aria-controls")).hidden = !expanded;
    button.querySelector("i").className = `fa fa-chevron-${expanded ? "up" : "down"} pull-right`;
}));
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/crm_config/edit"]').setAttribute("aria-current", "page");
initializeClusterConfig();
