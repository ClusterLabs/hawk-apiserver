let reportDraftRange;
let reportDraftPreset;
let reportCalendarMonths;

function initializeReportRangePicker() {
    const picker = document.getElementById("report-range-picker");
    picker.addEventListener("toggle", () => {
        if (picker.open) resetReportRangeDraft();
    });
    document.addEventListener("click", event => {
        if (picker.open && !picker.contains(event.target)) cancelReportRange();
    });
    picker.addEventListener("keydown", event => {
        if (event.key === "Escape") {
            cancelReportRange();
            picker.querySelector("summary").focus();
        }
        if (event.key === "Enter" && event.target.matches("input")) {
            event.preventDefault();
            applyReportRange();
        }
    });
}

function resetReportRangeDraft() {
    reportDraftRange = {
        from: new Date(document.getElementById("report-from").value),
        to: new Date(document.getElementById("report-to").value),
    };
    reportDraftPreset = document.getElementById("report-range").value;
    reportCalendarMonths = {
        from: new Date(reportDraftRange.from.getFullYear(), reportDraftRange.from.getMonth(), 1),
        to: new Date(reportDraftRange.to.getFullYear(), reportDraftRange.to.getMonth(), 1),
    };
    document.getElementById("report-calendars").hidden = true;
    document.querySelector(".report-range-fields").classList.remove("expanded");
    document.getElementById("report-range-error").hidden = true;
    updateReportRangeDraft();
}

function updateReportRangeDraft() {
    for (const side of ["from", "to"]) {
        document.getElementById(`report-draft-${side}`).value = localReportTime(reportDraftRange[side]).replace("T", " ");
    }
    document.querySelectorAll("[data-range]").forEach(button => {
        button.setAttribute("aria-pressed", button.dataset.range === reportDraftPreset ? "true" : "false");
    });
    renderReportCalendar("from");
    renderReportCalendar("to");
}

function selectReportPreset(preset) {
    reportDraftPreset = preset;
    if (preset === "custom") {
        updateReportRangeDraft();
        expandReportCalendars();
        return;
    }
    reportDraftRange = reportPresetDates(preset);
    for (const side of ["from", "to"]) {
        reportCalendarMonths[side] = new Date(reportDraftRange[side].getFullYear(), reportDraftRange[side].getMonth(), 1);
    }
    document.getElementById("report-range-error").hidden = true;
    updateReportRangeDraft();
}

function expandReportCalendars() {
    document.getElementById("report-calendars").hidden = false;
    document.querySelector(".report-range-fields").classList.add("expanded");
    renderReportCalendar("from");
    renderReportCalendar("to");
}

function readReportDraftDate(side) {
    const input = document.getElementById(`report-draft-${side}`);
    const value = input.value.trim().replace(" ", "T");
    const date = new Date(value);
    if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2})?$/.test(value) || !Number.isFinite(date.getTime()) || !localReportTime(date).startsWith(value)) {
        showReportRangeError("Enter a valid date as YYYY-MM-DD HH:mm:ss.");
        return null;
    }
    return date;
}

function editReportDate(side) {
    const date = readReportDraftDate(side);
    if (!date) return;
    if (date.getTime() !== reportDraftRange[side].getTime()) reportDraftPreset = "custom";
    reportDraftRange[side] = date;
    reportCalendarMonths[side] = new Date(date.getFullYear(), date.getMonth(), 1);
    document.getElementById("report-range-error").hidden = true;
    updateReportRangeDraft();
}

function showReportRangeError(message) {
    const error = document.getElementById("report-range-error");
    error.textContent = message;
    error.hidden = false;
}

function applyReportRange() {
    const from = readReportDraftDate("from");
    const to = readReportDraftDate("to");
    if (!from || !to) return;
    if (to <= from) {
        showReportRangeError("Until must be after From.");
        return;
    }
    if (from.getTime() !== reportDraftRange.from.getTime() || to.getTime() !== reportDraftRange.to.getTime()) reportDraftPreset = "custom";
    reportDraftRange = { from, to };
    for (const side of ["from", "to"]) {
        document.getElementById(`report-${side}`).value = localReportTime(reportDraftRange[side]);
    }
    document.getElementById("report-range").value = reportDraftPreset;
    updateReportRangeLabel();
    document.getElementById("report-range-picker").open = false;
}

function cancelReportRange() {
    document.getElementById("report-range-picker").open = false;
}

function reportWeekNumber(date) {
    const day = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()));
    day.setUTCDate(day.getUTCDate() + 4 - (day.getUTCDay() || 7));
    return Math.ceil(((day - new Date(Date.UTC(day.getUTCFullYear(), 0, 1))) / 86400000 + 1) / 7);
}

function reportCalendarSelect(label, values, selected, onChange) {
    const select = document.createElement("select");
    select.setAttribute("aria-label", label);
    for (const [value, text] of values) {
        const option = document.createElement("option");
        option.value = value;
        option.textContent = text;
        select.append(option);
    }
    select.value = selected;
    select.addEventListener("change", () => onChange(Number(select.value)));
    return select;
}

function renderReportCalendar(side) {
    const container = document.getElementById(`report-calendar-${side}`);
    const month = reportCalendarMonths[side];
    const selected = reportDraftRange[side];
    const grid = document.createElement("div");
    grid.className = "report-calendar-grid";
    const header = document.createElement("div");
    header.className = "report-calendar-header";
    function moveMonth(offset) {
        reportCalendarMonths[side] = new Date(month.getFullYear(), month.getMonth() + offset, 1);
        renderReportCalendar(side);
    }
    function monthButton(text, label, offset) {
        const button = document.createElement("button");
        button.type = "button";
        button.textContent = text;
        button.setAttribute("aria-label", `${side === "from" ? "From" : "Until"}: ${label}`);
        button.addEventListener("click", () => moveMonth(offset));
        return button;
    }
    const months = Array.from({ length: 12 }, (_, index) => [index, new Date(2026, index, 1).toLocaleString("en", { month: "long" })]);
    const firstYear = Math.min(1970, month.getFullYear());
    const lastYear = Math.max(new Date().getFullYear() + 10, month.getFullYear());
    const years = Array.from({ length: lastYear - firstYear + 1 }, (_, index) => [firstYear + index, firstYear + index]);
    header.append(monthButton("➜", "Previous month", -1));
    header.firstChild.style.transform = "rotate(180deg)";
    header.append(reportCalendarSelect(`${side}: month`, months, month.getMonth(), value => {
        reportCalendarMonths[side] = new Date(month.getFullYear(), value, 1);
        renderReportCalendar(side);
    }));
    header.append(reportCalendarSelect(`${side}: year`, years, month.getFullYear(), value => {
        reportCalendarMonths[side] = new Date(value, month.getMonth(), 1);
        renderReportCalendar(side);
    }));
    header.append(monthButton("➜", "Next month", 1));
    grid.append(header);
    const table = document.createElement("table");
    const weekdays = document.createElement("tr");
    for (const name of ["W", "Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"]) {
        const heading = document.createElement("th");
        heading.textContent = name;
        heading.scope = "col";
        if (name === "W") heading.className = "report-week";
        weekdays.append(heading);
    }
    const thead = document.createElement("thead");
    thead.append(weekdays);
    table.append(thead);
    const body = document.createElement("tbody");
    const first = new Date(month.getFullYear(), month.getMonth(), 1);
    first.setDate(first.getDate() - (first.getDay() + 6) % 7);
    const fromDay = new Date(reportDraftRange.from.getFullYear(), reportDraftRange.from.getMonth(), reportDraftRange.from.getDate());
    for (let week = 0; week < 6; week++) {
        const row = document.createElement("tr");
        const weekCell = document.createElement("td");
        const monday = new Date(first.getFullYear(), first.getMonth(), first.getDate() + week * 7);
        weekCell.className = "report-week";
        weekCell.textContent = reportWeekNumber(monday);
        row.append(weekCell);
        for (let day = 0; day < 7; day++) {
            const date = new Date(monday.getFullYear(), monday.getMonth(), monday.getDate() + day);
            const cell = document.createElement("td");
            const button = document.createElement("button");
            button.type = "button";
            button.textContent = date.getDate();
            button.className = "report-calendar-day";
            button.setAttribute("aria-label", `${side}: ${localReportTime(date).slice(0, 10)}`);
            const isSelected = date.toDateString() === selected.toDateString();
            button.setAttribute("aria-pressed", String(isSelected));
            if (date.getMonth() !== month.getMonth()) button.classList.add("outside-month");
            if (date > reportDraftRange.from && date < reportDraftRange.to) button.classList.add("in-range");
            if (isSelected) button.classList.add("selected");
            button.disabled = side === "to" && date < fromDay;
            button.addEventListener("click", () => {
                reportDraftRange[side] = new Date(date.getFullYear(), date.getMonth(), date.getDate(), selected.getHours(), selected.getMinutes(), selected.getSeconds());
                reportDraftPreset = "custom";
                reportCalendarMonths[side] = new Date(date.getFullYear(), date.getMonth(), 1);
                updateReportRangeDraft();
            });
            cell.append(button);
            row.append(cell);
        }
        body.append(row);
    }
    table.append(body);
    grid.append(table);
    const time = document.createElement("div");
    time.className = "report-calendar-time";
    for (const [unit, count, value] of [["hour", 24, selected.getHours()], ["minute", 60, selected.getMinutes()]]) {
        if (unit === "minute") time.append(document.createTextNode(" : "));
        time.append(reportCalendarSelect(`${side}: ${unit}`, Array.from({ length: count }, (_, index) => [index, unit === "minute" ? String(index).padStart(2, "0") : index]), value, next => {
            if (unit === "hour") selected.setHours(next);
            else selected.setMinutes(next);
            reportDraftPreset = "custom";
            updateReportRangeDraft();
        }));
    }
    container.replaceChildren(grid, time);
}
