class NodeDetailsPopup extends Popup {
  #name;
  #id;
  #fenceHistory;

  constructor(id, name, fenceHistory) {
    super();

    this.#id = id;
    this.#name = name;
    this.#fenceHistory = fenceHistory;

    // Header
    const header = document.createElement("div");
    header.className = "modal-header";
    header.innerHTML = `<h3 class="modal-title"><i class="fa fa-server page"></i> Node: ${name}</h3>`;

    const xBtn = document.createElement("button");
    xBtn.className = "close";
    xBtn.type = "button";
    xBtn.innerHTML = `<span aria-hidden="true">&times;</span><span class="sr-only">Close</span>`;
    xBtn.onclick = () => this.close();
    header.appendChild(xBtn);

    this._appendModalContentChild(header);

    // Body
    const modalBody = document.createElement("div");
    modalBody.className = "modal-body";
    modalBody.innerText = "Loading, please wait...";
    this._appendModalContentChild(modalBody);
    this.#loadContent(modalBody);

    // Footer
    const footer = document.createElement("div");
    footer.className = "modal-footer";

    const closeBtn = document.createElement("button");
    closeBtn.className = "btn btn-default cancel";
    closeBtn.textContent = "Close";
    closeBtn.onclick = () => this.close();

    footer.appendChild(closeBtn);
    this._appendModalContentChild(footer);
  }

  async #loadContent(modalBody) {
    const fenceHistoryDiv = this.#makeFencingHistoryDiv();
    const idTable = this.#makeIdTable();

    const [utilizationTable, attributesTable] = await Promise.all([
      this.#makeKvTable("utilization", "/api/cib/node/utilizations/fetch"),
      this.#makeKvTable("attributes", "/api/cib/node/attributes/fetch")
    ]);

    modalBody.innerHTML = ""; // clear the "Loading..." text

    if(fenceHistoryDiv) modalBody.appendChild(fenceHistoryDiv);
    if(idTable) modalBody.appendChild(idTable);

    const utilizationHeading = document.createElement("h4");
    utilizationHeading.innerText = "Utilization";
    modalBody.appendChild(utilizationHeading);
    modalBody.appendChild(utilizationTable);

    const hr = document.createElement("hr");
    const attributesHeading = document.createElement("h4");
    attributesHeading.innerText = "Attributes";
    modalBody.appendChild(hr);
    modalBody.appendChild(attributesHeading);
    modalBody.appendChild(attributesTable);
  }

  #makeFencingHistoryDiv() {
    const fenceHistoryDiv = document.createElement("div");
    fenceHistoryDiv.classList = "alert alert-warning";
    fenceHistoryDiv.role = "warning";
    fenceHistoryDiv.innerText = this.#fenceHistory;
    return fenceHistoryDiv;
  }

  #makeIdTable() {
    const idTable = document.createElement("table");
    idTable.classList = "table table-condensed";
    idTable.innerHTML = `<tbody><tr><th class="col-xs-4">ID</th><td class="col-xs-8">${this.#id}</td></tr></tbody>`;
    return idTable;
  }

  async #makeKvTable(desc, api) {
    const res = await statusFetch(api, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({CibObject: this.#name})
    });
    if (!res.ok) throw new Error(`HTTP error: ${res.status}`);
    const nvpairs = await res.json() || [];

    if (nvpairs.length == 0) {
      const emptyTableAlert = document.createElement("div");
      emptyTableAlert.classList = "alert alert-info";
      emptyTableAlert.innerText = desc;
      return emptyTableAlert;
    }

    const table = document.createElement("table");
    table.classList = "table table-striped";
    const tbody = document.createElement("tbody");
    table.appendChild(tbody);

    nvpairs.forEach(nv => {
      const tr = document.createElement("tr");
      tr.innerHTML = `<th class="col-xs-4">${nv.name}</th><td class="col-xs-8">0 / ${nv.value} (0%)</td>`;
      tbody.appendChild(tr);
    });
    return table;
  }
}

window.NodeDetailsPopup = NodeDetailsPopup;
