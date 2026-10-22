class ClusterDetailsPopup extends Popup {
  #tbody;

  constructor() {
    super();

    // Header
    const header = document.createElement("div");
    header.className = "modal-header";

    const closeBtn = document.createElement("button");
    closeBtn.className = "close";
    closeBtn.type = "button";
    closeBtn.innerHTML = `<span aria-hidden="true">&times;</span><span class="sr-only">Close</span>`;
    closeBtn.onclick = () => this.close();
    header.appendChild(closeBtn);

    const h3 = document.createElement("h3");
    h3.className = "modal-title";
    h3.innerHTML = `<i class="fas fa-info page"></i> Cluster Details`;
    header.appendChild(h3);

    this._appendModalContentChild(header);

    // Body
    const modalBody = document.createElement("div");
    modalBody.className = "modal-body help";

    const table = document.createElement("table");
    table.className = "table table-bordered metadata";

    this.#tbody = document.createElement("tbody");
    table.appendChild(this.#tbody);
    modalBody.appendChild(table);
    this._appendModalContentChild(modalBody);

    this.#fetchData();
  }

  async #fetchData() {
    try {
      const res = await fetch("/api/cib/cluster/details/fetch", {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ host: window.location.hostname})
      });

      if (!res.ok) throw new Error(`HTTP error: ${res.status}`);
      const { summary, status, epoch, host, dc, schema,
        lastWritten, updateOrigin, updateUser, haveQuorum,
        version, stack, fencingEnabled } = await res.json(); // Expected: array of { Summary { Name, Value }}

      this.#addRow("Status", status);
      this.#addRow("Epoch", epoch);
      this.#addRow("Host", host);
      this.#addRow("DC", dc);
      this.#addRow("Schema", schema);
      this.#addRow("LastWritten", lastWritten);
      this.#addRow("UpdateOrigin", updateOrigin);
      this.#addRow("UpdateUser", updateUser);
      this.#addRow("HaveQuorum", haveQuorum);
      this.#addRow("Version", version);
      this.#addRow("Stack", stack);
      this.#addRow("FencingEnabled", fencingEnabled);
      setClusterStatusIndicator(summary, status);
    } catch (err) {
      console.error("Failed to fetch cluster details:", err);
      const tr = document.createElement("tr");
      tr.innerHTML = `<td colspan="2" class="text-danger">Error fetching cluster details</td>`;
      this.#tbody.appendChild(tr);
    }
  }

  #addRow(name, value) {
    const tr = document.createElement("tr");

    const th = document.createElement("th");
    th.className = "col-xs-4";
    th.textContent = name;
    th.style.textAlign = "center";

    const td = document.createElement("td");
    td.className = "col-xs-8";
    td.style.textAlign = "center";

    const span = document.createElement("span");
    span.textContent = value;
    span.style.fontFamily = "monospace";
    td.appendChild(span);

    tr.append(th, td);
    this.#tbody.appendChild(tr);
  }
}
