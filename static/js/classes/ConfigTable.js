"use strict";

function configElement(tag, text, className) {
    const element = document.createElement(tag);
    if (text !== undefined) element.textContent = text;
    if (className) element.className = className;
    return element;
}

async function configJSON(url, options = {}) {
    const response = await fetch(url, { cache: "no-store", ...options, headers: { Accept: "application/json", "X-Requested-With": "XMLHttpRequest", ...options.headers } });
    let data;
    try { data = await response.json(); } catch { throw new Error(`Request failed (${response.status}).`); }
    if (!response.ok || data?.error || data?.success === false) throw new Error(data.error || data.message || `Request failed (${response.status}).`);
    return data;
}

class ConfigTable {
    constructor(table, options = {}) {
        this.table = table;
        this.kind = table.dataset.configTable;
        this.rows = [];
        this.query = "";
        this.page = 1;
        this.size = options.pageSize || 25;
        this.sort = this.kind === "nodes" ? "name" : "id";
        this.descending = false;
        this.sequence = 0;
        this.error = table.parentElement.querySelector(".config-table-error");
        this.columns = [...options.columns];
        this.url = options.url;
        this.readRows = options.rows || (data => data);
        if (this.kind !== "fencing") {
            this.columns.push([null, "Operations", options.operations]);
            this.createControls();
        }
        this.head = table.createTHead();
        this.body = table.createTBody();
        this.render();
    }

    createControls() {
        const toolbar = configElement("div", undefined, "config-table-toolbar fixed-table-toolbar");
        const search = configElement("input", undefined, "form-control");
        search.type = "search";
        search.placeholder = "Search";
        search.setAttribute("aria-label", `Search ${this.kind}`);
        search.addEventListener("input", () => { this.query = search.value.toLowerCase(); this.page = 1; this.render(); });
        const refresh = configElement("button", undefined, "btn btn-default");
        refresh.type = "button";
        refresh.title = "Refresh";
        refresh.setAttribute("aria-label", `Refresh ${this.kind}`);
        refresh.append(configElement("i", undefined, "fa fa-refresh"));
        refresh.addEventListener("click", () => this.load());
        toolbar.append(search, refresh);
        this.table.before(toolbar);
        this.pagination = configElement("div", undefined, "config-table-pagination fixed-table-pagination");
        this.table.after(this.pagination);
    }

    async load() {
        const sequence = ++this.sequence;
        this.table.setAttribute("aria-busy", "true");
        try {
            const data = this.readRows(await configJSON(this.url));
            if (!Array.isArray(data)) throw new Error("Invalid table response.");
            if (sequence !== this.sequence) return;
            this.rows = data;
            this.error.hidden = true;
            this.render();
        } catch (error) {
            if (sequence !== this.sequence) return;
            this.error.textContent = `Unable to load ${this.kind}. ${error.message}`;
            this.error.hidden = false;
        } finally {
            if (sequence === this.sequence) this.table.setAttribute("aria-busy", "false");
        }
    }

    cellValue(column, row) {
        return column[2] ? column[2](row) : row[column[0]] ?? "";
    }

    render() {
        const header = this.head.insertRow();
        this.head.replaceChildren(header);
        for (const column of this.columns) {
            const cell = configElement("th");
            cell.scope = "col";
            if (this.kind !== "fencing") {
                if (column[1] === "Name") cell.className = "col-sm-4";
                else if (column[1] === "Type" || column[1] === "Operations") cell.className = "col-sm-2";
            }
            if (column[0]) {
                const button = configElement("button", column[1]);
                button.type = "button";
                button.append(configElement("span", undefined, `fa fa-${this.sort === column[0] ? (this.descending ? "sort-desc" : "sort-asc") : "sort"} config-sort`));
                cell.setAttribute("aria-sort", this.sort === column[0] ? (this.descending ? "descending" : "ascending") : "none");
                button.addEventListener("click", () => {
                    this.descending = this.sort === column[0] ? !this.descending : false;
                    this.sort = column[0];
                    this.render();
                });
                cell.append(button);
            } else cell.textContent = column[1];
            header.append(cell);
        }
        let rows = this.rows;
        if (this.kind !== "fencing") {
            rows = rows.filter(row => this.columns.slice(0, -1).some(column => {
                const value = this.cellValue(column, row);
                return String(value instanceof Node ? value.textContent : value).toLowerCase().includes(this.query);
            })).slice().sort((a, b) => String(a[this.sort] ?? "").localeCompare(String(b[this.sort] ?? "")) * (this.descending ? -1 : 1));
        }
        const total = rows.length;
        const pages = Math.max(1, Math.ceil(total / this.size));
        this.page = Math.min(this.page, pages);
        if (this.pagination) rows = rows.slice((this.page - 1) * this.size, this.page * this.size);
        this.body.replaceChildren();
        for (const row of rows) {
            const tr = this.body.insertRow();
            for (const column of this.columns) {
                const td = tr.insertCell();
                const value = this.cellValue(column, row);
                if (value instanceof Node) td.append(value);
                else td.textContent = value;
                if (column[1] === "Operations") td.className = "config-operations";
            }
        }
        if (!rows.length) {
            const td = this.body.insertRow().insertCell();
            td.colSpan = this.columns.length;
            td.className = "text-center";
            td.textContent = "No matching records found";
        }
        if (this.pagination) this.renderPagination(total, pages);
    }

    renderPagination(total, pages) {
        const info = configElement("div", `Showing ${total ? (this.page - 1) * this.size + 1 : 0} to ${Math.min(this.page * this.size, total)} of ${total} rows `);
        const sizes = configElement("select", undefined, "form-control input-sm");
        sizes.setAttribute("aria-label", `Rows per page for ${this.kind}`);
        for (const size of [10, 25, 50, 100, 200]) {
            const option = configElement("option", String(size));
            option.value = size;
            option.selected = size === this.size;
            sizes.append(option);
        }
        sizes.addEventListener("change", () => { this.size = Number(sizes.value); this.page = 1; this.render(); });
        info.append(sizes, " records per page");
        const nav = configElement("ul", undefined, "pagination pagination-sm");
        const first = Math.max(1, Math.min(this.page - 2, pages - 4));
        const entries = [["«", 1], ["‹", this.page - 1], ...Array.from({ length: Math.min(5, pages) }, (_, n) => [String(first + n), first + n]), ["›", this.page + 1], ["»", pages]];
        for (const [label, page] of entries) {
            const current = label === String(page) && page === this.page;
            const disabled = page < 1 || page > pages || ((label === "«" || label === "»") && page === this.page);
            const li = configElement("li", undefined, current ? "active" : "");
            const link = configElement("a", label);
            link.href = "#";
            if (current) link.setAttribute("aria-current", "page");
            if (disabled) { li.classList.add("disabled"); link.setAttribute("aria-disabled", "true"); }
            link.addEventListener("click", event => { event.preventDefault(); if (!disabled) { this.page = page; this.render(); } });
            li.append(link);
            nav.append(li);
        }
        this.pagination.replaceChildren(info, nav);
    }
}

