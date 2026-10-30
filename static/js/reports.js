let reports = [];
let reportsPage = 1;
let reportsSort = "to_time";
let reportsDescending = true;
let reportsRunning = null;
let reportsPollTimer = null;

function showFlash(type, message) {
    const alert = document.createElement("x-alert");
    alert.id = "reports-alert";
    alert.setAttribute("type", type);
    alert.setAttribute("message", message);
    alert.setAttribute("visible", "true");
    document.getElementById("reports-alert").replaceWith(alert);
}

async function reportsRequest(path, options = {}) {
    const response = await fetch(`/api/reports${path}`, {
        ...options,
        headers: { Accept: "application/json", ...options.headers },
        cache: "no-store",
        signal: AbortSignal.timeout(120000),
    });
    // Authentication middleware can redirect to the Rails login page.
    if (response.redirected || !response.headers.get("Content-Type")?.includes("application/json")) {
        throw new Error("The server did not return report data. Check your connection and sign in again if needed.");
    }
    const data = await response.json();
    if (!response.ok || data.error) throw new Error(data.error || `Request failed (${response.status}).`);
    return data;
}

function formatReportTime(value) {
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? "" : date.toISOString().replace("T", " ").replace(/\.\d+Z$/, " UTC");
}

function sortReports(field) {
    reportsDescending = reportsSort === field ? !reportsDescending : false;
    reportsSort = field;
    renderReports();
}

function renderReports() {
    const search = document.getElementById("reports-search").value.toLowerCase();
    const pageSize = Number(document.getElementById("reports-page-size").value);
    const filtered = reports.filter(report => [report.name, formatReportTime(report.from_time), formatReportTime(report.to_time)]
        .some(value => value.toLowerCase().includes(search)));
    filtered.sort((a, b) => {
        const order = reportsSort === "name" ? a.name.localeCompare(b.name) : new Date(a[reportsSort]) - new Date(b[reportsSort]);
        return order * (reportsDescending ? -1 : 1);
    });
    const pages = Math.max(1, Math.ceil(filtered.length / pageSize));
    reportsPage = Math.max(1, Math.min(reportsPage, pages));
    const body = document.getElementById("reports-body");
    body.replaceChildren();
    for (const report of filtered.slice((reportsPage - 1) * pageSize, reportsPage * pageSize)) {
        const row = document.createElement("tr");
        const name = document.createElement("td");
        const link = document.createElement("a");
        link.href = `/reports/${encodeURIComponent(report.id)}`;
        link.textContent = report.name.startsWith("hawk-") ? "hawk" : report.name;
        link.title = report.name;
        name.append(link);
        row.append(name);
        for (const value of [report.from_time === report.to_time ? "" : formatReportTime(report.from_time), formatReportTime(report.to_time)]) {
            const cell = document.createElement("td");
            cell.textContent = value;
            row.append(cell);
        }
        const actions = document.createElement("td");
        const group = document.createElement("div");
        group.className = "btn-group";
        for (const [title, icon, href] of [
            ["Show", "search", link.href],
            ["Download", "download", `/api/reports/${encodeURIComponent(report.id)}/download`],
        ]) {
            const action = document.createElement("a");
            action.className = "btn btn-default btn-xs";
            action.href = href;
            action.title = title;
            action.setAttribute("aria-label", title);
            const symbol = document.createElement("i");
            symbol.className = `fas fa-${icon}`;
            action.append(symbol);
            group.append(action);
        }
        const remove = document.createElement("button");
        remove.type = "button";
        remove.className = "btn btn-default btn-xs";
        remove.title = "Delete";
        remove.setAttribute("aria-label", "Delete");
        remove.innerHTML = '<i class="fas fa-trash" aria-hidden="true"></i>';
        remove.addEventListener("click", () => clickDeleteReportButton(report, remove));
        group.append(remove);
        actions.append(group);
        row.append(actions);
        body.append(row);
    }
    if (!filtered.length) {
        const row = document.createElement("tr");
        const cell = document.createElement("td");
        cell.colSpan = 4;
        cell.textContent = "No matching records found";
        row.append(cell);
        body.append(row);
    }
    const first = (reportsPage - 1) * pageSize + 1;
    const last = Math.min(reportsPage * pageSize, filtered.length);
    document.getElementById("reports-status").textContent = `Showing ${first} to ${last} of ${filtered.length} rows`;
    const fields = ["name", "from_time", "to_time"];
    document.querySelectorAll(".reports thead th[aria-sort]").forEach((heading, index) => {
        heading.setAttribute("aria-sort", reportsSort === fields[index] ? (reportsDescending ? "descending" : "ascending") : "none");
    });
    renderReportPagination(filtered.length ? pages : 0);
}

function renderReportPagination(pages) {
    const pagination = document.getElementById("reports-pagination");
    pagination.replaceChildren();
    const buttons = [["«", "First page", 1], ["‹", "Previous page", reportsPage - 1]];
    const start = Math.max(1, Math.min(reportsPage - 2, pages - 4));
    for (let page = start; page <= Math.min(pages, start + 4); page++) {
        buttons.push([String(page), `Page ${page}`, page]);
    }
    buttons.push(["›", "Next page", reportsPage + 1], ["»", "Last page", pages]);
    for (const [text, label, page] of buttons) {
        const item = document.createElement("li");
        const button = document.createElement("button");
        button.type = "button";
        button.textContent = text;
        button.setAttribute("aria-label", label);
        const numbered = label.startsWith("Page ");
        button.disabled = pages === 0 || page < 1 || page > pages || (!numbered && page === reportsPage);
        if (button.disabled) item.className = "disabled";
        if (numbered && page === reportsPage) {
            item.className = "active";
            button.setAttribute("aria-current", "page");
        }
        button.addEventListener("click", () => {
            reportsPage = page;
            renderReports();
        });
        item.append(button);
        pagination.append(item);
    }
}

async function refreshReports() {
    reports = await reportsRequest("");
    renderReports();
}

async function clickRefreshReportsButton() {
    try {
        await Promise.all([refreshReports(), refreshReportStatus()]);
    } catch (err) {
        document.getElementById("reports-status").textContent = "Unable to refresh reports.";
        showFlash("danger", err.message);
    }
}

async function refreshReportStatus() {
    try {
        const state = await reportsRequest("/running");
        document.getElementById("report-running").classList.toggle("hidden", !state.running);
        document.getElementById("report-controls").classList.toggle("hidden", state.running);
        document.getElementById("running-range").textContent = state.time.map(formatReportTime).join(" — ");
        if (reportsRunning === true && !state.running) {
            showFlash(state.report_generated ? "success" : "warning", state.report_generated ? "Report generation is complete." : `Report collection stopped. ${state.msg || "No result was recorded."}`);
            await refreshReports();
        }
        reportsRunning = state.running;
    } finally {
        clearTimeout(reportsPollTimer);
        // Retry status failures as well, without overlapping requests.
        reportsPollTimer = setTimeout(() => refreshReportStatus().catch(err => showFlash("danger", err.message)), 5000);
    }
}

function localReportTime(date) {
    return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 19);
}

function updateReportRangeLabel() {
    const values = ["report-from", "report-to"].map(id => {
        const date = new Date(document.getElementById(id).value);
        if (!Number.isFinite(date.getTime())) return "Select date";
        const offset = -date.getTimezoneOffset();
        const hours = String(Math.floor(Math.abs(offset) / 60)).padStart(2, "0");
        const minutes = String(Math.abs(offset) % 60).padStart(2, "0");
        return `${localReportTime(date).replace("T", " ")} ${offset < 0 ? "-" : "+"}${hours}${minutes}`;
    });
    document.getElementById("report-range-label").textContent = values.join(" - ");
}

function reportPresetDates(range) {
    const end = new Date();
    end.setMilliseconds(0);
    const start = new Date(end);
    if (range === "hour" || range === "six") start.setHours(start.getHours() - (range === "six" ? 6 : 1));
    else {
        start.setHours(0, 0, 0, 0);
        if (range === "yesterday") { start.setDate(start.getDate() - 1); end.setHours(0, 0, 0, 0); end.setMilliseconds(-1000); }
        else {
            if (range === "week") start.setDate(start.getDate() - 6);
            if (range === "month") { start.setDate(1); end.setMonth(end.getMonth() + 1, 0); }
            end.setHours(23, 59, 59, 0);
        }
    }
    return { from: start, to: end };
}

function changeReportRange() {
    const range = reportPresetDates(document.getElementById("report-range").value);
    document.getElementById("report-from").value = localReportTime(range.from);
    document.getElementById("report-to").value = localReportTime(range.to);
    updateReportRangeLabel();
}

async function clickGenerateReportButton(event) {
    event.preventDefault();
    document.getElementById("report-range-picker").open = false;
    const button = document.getElementById("generate-report");
    button.disabled = true;
    try {
        const from = new Date(document.getElementById("report-from").value);
        const to = new Date(document.getElementById("report-to").value);
        if (!Number.isFinite(from.getTime()) || !Number.isFinite(to.getTime()) || to <= from) throw new Error("Select valid dates with Until after From.");
        const result = await reportsRequest("/generate", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ from_time: from.toISOString(), to_time: to.toISOString() }) });
        showFlash("info", result.message);
        reportsRunning = true;
        await refreshReportStatus();
    } catch (err) { showFlash("danger", err.message); }
    finally { button.disabled = false; }
}

let reportUploadRequest = null;

function updateReportUpload() {
    const file = document.getElementById("report-upload").files[0];
    document.getElementById("report-upload-name").textContent = file ? file.name : "";
    for (const id of ["report-upload-icon", "remove-report-upload", "upload-report"]) {
        document.getElementById(id).hidden = !file;
    }
}

function removeReportUpload() {
    if (reportUploadRequest) {
        reportUploadRequest.abort();
        return;
    }
    document.getElementById("report-upload").value = "";
    updateReportUpload();
}

function uploadReportArchive(data) {
    return new Promise((resolve, reject) => {
        const request = new XMLHttpRequest();
        reportUploadRequest = request;
        request.open("POST", "/api/reports/upload");
        request.setRequestHeader("Accept", "application/json");
        request.responseType = "json";
        request.timeout = 120000;
        request.upload.onprogress = event => {
            if (!event.lengthComputable) return;
            const percent = Math.min(100, Math.round(event.loaded / event.total * 100));
            const bar = document.getElementById("report-upload-progress-bar");
            bar.style.width = `${percent}%`;
            bar.textContent = `${percent}%`;
            bar.setAttribute("aria-valuenow", percent);
        };
        request.onload = () => {
            const result = request.response;
            if (!request.getResponseHeader("Content-Type")?.includes("application/json") || !result) {
                reject(new Error("The server did not return report data. Check your connection and sign in again if needed."));
            } else if (request.status < 200 || request.status >= 300 || result.error) {
                reject(new Error(result.error || `Request failed (${request.status}).`));
            } else {
                resolve(result);
            }
        };
        request.onerror = () => reject(new Error("Unable to upload report. Check your connection and try again."));
        request.ontimeout = () => reject(new Error("Report upload timed out. Try again."));
        request.onabort = () => reject(new DOMException("Upload cancelled.", "AbortError"));
        request.send(data);
    });
}

async function clickUploadReportButton(event) {
    event.preventDefault();
    if (reportUploadRequest) return;
    const button = document.getElementById("upload-report");
    const input = document.getElementById("report-upload");
    const remove = document.getElementById("remove-report-upload");
    const row = document.querySelector(".report-upload-row");
    const progress = document.getElementById("report-upload-progress");
    const bar = document.getElementById("report-upload-progress-bar");
    button.disabled = true;
    document.getElementById("remove-report-upload-label").textContent = "Cancel";
    document.getElementById("remove-report-upload-icon").hidden = true;
    row.setAttribute("aria-busy", "true");
    progress.hidden = false;
    bar.style.width = "0%";
    bar.textContent = "0%";
    bar.setAttribute("aria-valuenow", 0);
    try {
        const data = new FormData(event.target);
        input.disabled = true;
        const result = await uploadReportArchive(data);
        reportUploadRequest = null;
        remove.disabled = true;
        showFlash("success", result.message);
        event.target.reset();
        updateReportUpload();
        await refreshReports();
    } catch (err) { showFlash(err.name === "AbortError" ? "info" : "danger", err.message); }
    finally {
        button.disabled = false;
        input.disabled = false;
        remove.disabled = false;
        reportUploadRequest = null;
        row.setAttribute("aria-busy", "false");
        progress.hidden = true;
        document.getElementById("remove-report-upload-label").textContent = "Remove";
        document.getElementById("remove-report-upload-icon").hidden = false;
    }
}

async function clickCancelReportButton() {
    const button = document.getElementById("cancel-report");
    button.disabled = true;
    try {
        const result = await reportsRequest("/cancel", { method: "POST" });
        showFlash("info", result.message);
    } catch (err) { showFlash("danger", err.message); }
    finally {
        try {
            await refreshReportStatus();
        } catch (err) { showFlash("danger", err.message); }
        button.disabled = false;
    }
}

async function clickDeleteReportButton(report, button) {
    if (!window.confirm(`Are you sure you wish to delete ${report.name}?`)) return;
    button.disabled = true;
    try {
        const result = await reportsRequest(`/${encodeURIComponent(report.id)}`, { method: "DELETE" });
        showFlash("success", result.message);
        await refreshReports();
    } catch (err) { showFlash("danger", err.message); }
    finally { button.disabled = false; }
}

document.getElementById("generate-report-form").addEventListener("submit", clickGenerateReportButton);
document.getElementById("upload-report-form").addEventListener("submit", clickUploadReportButton);
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("troubleshootMenu").classList.add("in");
document.querySelector('a[href="/reports"]').setAttribute("aria-current", "page");
async function loadReportCapabilities() {
    const button = document.getElementById("generate-report");
    button.disabled = true;
    try {
        const state = await reportsRequest("/capabilities");
        if (!state.can_generate) {
            document.querySelector('a[href="#generate"]').parentElement.classList.add("hidden");
            $('a[href="#upload"]').tab("show");
            showFlash("info", state.reason);
        } else { button.disabled = false; }
    } catch (err) { showFlash("danger", err.message); }
}

changeReportRange();
initializeReportRangePicker();
loadReportCapabilities();
clickRefreshReportsButton();
window.addEventListener("pagehide", () => clearTimeout(reportsPollTimer));
