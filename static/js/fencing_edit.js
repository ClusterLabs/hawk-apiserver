"use strict";

const fencingForm = document.getElementById("fencing-editor");
const fencingList = document.getElementById("fencing-levels");
const fencingApply = document.getElementById("fencing-apply");
const fencingAdd = document.getElementById("fencing-add");
let fencingLevels = [];
let fencingDevices = [];
let fencingReady = false;
let fencingSaving = false;

function fencingNotice(message, danger = false) {
    const alert = document.createElement("x-alert");
    alert.setAttribute("type", danger ? "danger" : "success");
    alert.setAttribute("message", message);
    alert.setAttribute("visible", "true");
    document.getElementById("fencing-notifications").replaceChildren(alert);
}
function fencingElement(tag, text, className) {
    const element = document.createElement(tag);
    if (text !== undefined) element.textContent = text;
    if (className) element.className = className;
    return element;
}
function fencingChanged() { fencingApply.disabled = !fencingReady || fencingSaving; }
function renderFencingLevels() {
    fencingList.replaceChildren();
    fencingLevels.forEach((level, index) => {
        const panel = fencingElement("li", undefined, "panel panel-default fencing-level");
        const heading = fencingElement("div", undefined, "panel-heading");
        for (const [offset, title, icon] of [[-1, "Move up", "arrow-up"], [1, "Move down", "arrow-down"]]) {
            const button = fencingElement("button", undefined, "btn btn-default btn-sm");
            button.type = "button";
            button.setAttribute("aria-label", `${title} level ${index + 1}`);
            button.append(fencingElement("i", undefined, `fa fa-${icon}`));
            button.disabled = index + offset < 0 || index + offset >= fencingLevels.length;
            button.onclick = () => {
                [fencingLevels[index], fencingLevels[index + offset]] = [fencingLevels[index + offset], fencingLevels[index]];
                renderFencingLevels(); fencingChanged();
            };
            heading.append(button);
        }
        const remove = fencingElement("button", "×", "close");
        remove.type = "button";
        remove.setAttribute("aria-label", `Delete level ${index + 1}`);
        remove.onclick = () => new DeletePopup("this level", () => {
            fencingLevels.splice(index, 1); renderFencingLevels(); fencingChanged();
        });
        heading.append(remove);
        const body = fencingElement("div", undefined, "panel-body");
        function field(name, labelText, value, onInput, type = "text") {
            const row = fencingElement("div", undefined, "fencing-field");
            const label = fencingElement("label", labelText);
            const input = fencingElement("input", undefined, "form-control");
            input.id = `fencing-${index}-${name}`;
            input.type = type;
            input.value = value;
            label.htmlFor = input.id;
            input.addEventListener("input", () => { onInput(input.value); fencingChanged(); });
            row.append(label, input); body.append(row);
            return { row, input };
        }
        const target = field("target", "Target", level.target, value => { level.target = value; });
        target.input.required = true;
        const select = fencingElement("select", undefined, "form-control");
        select.setAttribute("aria-label", `Target type for level ${index + 1}`);
        for (const [value, title] of [["node", "Node"], ["pattern", "Pattern"], ["attribute", "Attribute"]]) {
            const option = fencingElement("option", title); option.value = value; select.append(option);
        }
        select.value = level.type;
        select.onchange = () => { level.type = select.value; renderFencingLevels(); fencingChanged(); };
        target.row.insertBefore(select, target.input);
        if (level.type === "attribute") {
            field("value", "Value", level.value || "", value => { level.value = value; }).input.required = true;
        }
        const idx = field("index", "Index", level.index, value => { level.index = Number(value); }, "number").input;
        idx.required = true; idx.min = "1"; idx.max = "9"; idx.step = "1";
        const devices = field("devices", "Devices", level.devices.join(", "), value => {
            level.devices = value.split(/[\s,]+/).filter(Boolean);
        }).input;
        devices.setAttribute("list", "fencing-devices");
        panel.append(heading, body); fencingList.append(panel);
    });
}
fencingAdd.onclick = () => {
    fencingLevels.push({ type: "node", target: "", value: "", index: 1, devices: [] });
    renderFencingLevels(); fencingChanged();
    fencingList.lastElementChild.querySelector("input").focus();
};
fencingForm.addEventListener("submit", async event => {
    event.preventDefault();
    if (!fencingReady || fencingSaving || fencingApply.disabled || !fencingForm.reportValidity()) return;
    const invalid = fencingLevels.flatMap((level, index) => level.devices
        .filter(id => !fencingDevices.includes(id)).map(id => `Level #${index + 1}: fencing device ${id} not found or not a fencing device`));
    if (invalid.length) { fencingNotice(invalid.join("\n"), true); return; }
    fencingSaving = true;
    // Keep the submitted state fixed until the save completes.
    fencingForm.querySelectorAll("button, input, select").forEach(control => { control.disabled = true; });
    try {
        const response = await fetch("/cib/live/fencing/edit", {
            method: "POST", headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ fencing: fencingLevels }),
        });
        const text = await response.text();
        let result;
        try { result = JSON.parse(text); } catch { throw new Error(text || "Unable to save fencing topology."); }
        if (!response.ok || !result.success) throw new Error(result.error || "Unable to save fencing topology.");
        fencingNotice("Changes applied successfully.");
        fencingApply.disabled = true;
    } catch (error) {
        fencingNotice(error.message, true); fencingApply.disabled = false;
    } finally {
        fencingSaving = false;
        renderFencingLevels(); fencingAdd.disabled = false;
    }
});
async function loadFencing() {
    try {
        const response = await fetch("/api/cib/fencing/fetch", { cache: "no-store" });
        const text = await response.text();
        let data;
        try { data = JSON.parse(text); } catch { throw new Error(text || "Unable to load fencing topology."); }
        if (!response.ok) throw new Error(data.error || "Unable to load fencing topology.");
        fencingLevels = data.fencing;
        fencingDevices = data.devices;
        const choices = document.getElementById("fencing-devices");
        for (const id of fencingDevices) { const option = fencingElement("option"); option.value = id; choices.append(option); }
        renderFencingLevels(); fencingReady = true; fencingAdd.disabled = false;
    } catch (error) { fencingNotice(error.message, true); }
    finally { fencingForm.setAttribute("aria-busy", "false"); }
}
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/config/edit"]')?.setAttribute("aria-current", "page");
loadFencing();
