"use strict";

async function loadCommands() {
    const container = document.getElementById("commands");
    const error = document.getElementById("commands-error");

    try {
        // Keep the Rails command log as the source of truth during the UI port.
        const response = await fetch("/cib/live/commands.json", {
            headers: { Accept: "application/json" },
            cache: "no-store",
        });
        if (!response.ok) throw new Error(`Request failed (${response.status}).`);
        const commands = await response.json();
        if (!Array.isArray(commands) || !commands.every(command => typeof command === "string")) {
            throw new Error("Invalid command log response.");
        }
        const entries = document.createDocumentFragment();
        for (const command of commands.slice().reverse()) {
            const newline = command.indexOf("\n");
            const pre = document.createElement("pre");
            const code = document.createElement("code");
            code.className = "crmsh";
            code.title = newline < 0 ? command : command.slice(0, newline);
            code.textContent = newline < 0 ? "" : command.slice(newline + 1);
            pre.append(code);
            entries.append(pre);
        }
        container.replaceChildren(entries);
    } catch (cause) {
        error.textContent = `Unable to load command log. ${cause.message}`;
        error.hidden = false;
    } finally {
        container.setAttribute("aria-busy", "false");
    }
}

// Open the sidebar section containing the current page.
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("troubleshootMenu").classList.add("in");
const link = document.querySelector('a[href="/cib/live/commands"]');
link.setAttribute("aria-current", "page");
loadCommands();
