class InfoPopup extends Popup {
  #resourceID;
  #resourceAgent;
  #shortdescEl;
  #longdescEl;

  constructor() {
    super();

    this.#resourceID = window.resourceData?.ResourceID || "";
    this.#resourceAgent = window.resourceData?.ResourceAgent || "";

    this._appendModalContentChild(this.#buildHeader());
    this._appendModalContentChild(this.#buildBody());
    this._appendModalContentChild(this.#buildFooter());
  }

  #buildHeader() {
    const header = document.createElement("div");
    header.className = "modal-header";

    const h4 = document.createElement("h4");

    const icon = document.createElement("i");
    icon.className = "fa fa-cogs page";
    h4.appendChild(icon);

    const code = document.createElement("code");
    code.textContent = this.#resourceAgent;
    h4.appendChild(document.createTextNode(" "));
    h4.appendChild(code);

    this.#shortdescEl = document.createElement("small");
    h4.appendChild(this.#shortdescEl);

    header.appendChild(h4);
    return header;
  }

  #buildBody() {
    const body = document.createElement("div");
    body.className = "modal-body";

    body.appendChild(this.#buildDescriptionSection());
    body.appendChild(this.#buildPanelGroup());

    return body;
  }

  #buildDescriptionSection() {
    const row = document.createElement("div");
    row.className = "row";

    this.#longdescEl = document.createElement("div");
    this.#longdescEl.className = "col-md-offset-1 col-md-10";

    //this.#longdescEl = document.createElement("p");

    //col.appendChild(this.#longdescEl);
    row.appendChild(this.#longdescEl);
    return row;
  }

  #buildPanelGroup() {
    const row = document.createElement("div");
    row.className = "row";

    const group = document.createElement("div");
    group.className = "panel-group";
    group.id = "agentinfo";
    group.setAttribute("role", "tablist");
    group.setAttribute("aria-multiselectable", "true");

    group.appendChild(this.#buildParametersPanel());
    group.appendChild(this.#buildActionsPanel());

    row.appendChild(group);
    return row;
  }

  #buildActionsPanel() {
    const table = document.createElement("table");
    table.className = "table";

    const tbody = document.createElement("tbody");
    tbody.appendChild( this.#makeHeaderRow("Name", "Timeout", "Interval", "Depth") );
    table.appendChild(tbody);

    fetch('/api/data-interface/fetch-resource-operations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({resourceID: this.#resourceID, resourceAgent: this.#resourceAgent})
    })
      .then(async res => {
        if (!res.ok) throw new Error(await res.text() || "Unknown error");
        return res.json();
      })
      .then(content => {
        const contentOptions = content.Options || [];
        contentOptions.forEach(o => {
          const tr = document.createElement("tr");

          const tdName = document.createElement("td");
          const code = document.createElement("code");
          code.textContent = o.Name;
          tdName.appendChild(code);

          const tdT = document.createElement("td");
          const tdI = document.createElement("td");
          const tdD = document.createElement("td");
          // TODO (robustness): check that o.DefaultValues[0] is "interval", ...[1] is "timeout", ...[2] is "depth"
          tdT.textContent = o.DefaultValues[0].Value; //tdT.textContent = timeout;
          tdI.textContent = o.DefaultValues[1].Value; //tdI.textContent = interval;
          tdD.textContent = o.DefaultValues[2].Value; //tdD.textContent = depth;

          tr.append(tdName, tdT, tdI, tdD);
          tbody.appendChild(tr);
        });
      })
      .catch(err => console.error("Failed to init InfoPopup::Actions", err));

    return this.#buildCollapsiblePanel(
      "Actions",
      "agentactions",
      "agentactioncontrol",
      false,
      table
    );
  }

  #buildParametersPanel() {
    return this.#buildCollapsiblePanel(
      "Parameters",
      "agentparams",
      "agentparamcontrol",
      false,
      this.#buildParametersTable()
    );
  }

  #renderMultiline(text) {
    if (!text) return "";
    const esc = text
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");
    return esc.replace(/\n/g, "<br>");
  }

  #buildParametersTable() {
    const table = document.createElement("table");
    table.className = "table";
    const tbody = document.createElement("tbody");
    tbody.appendChild(this.#makeHeaderRow("Name", "Shortdesc", "Longdesc", "Options"));
    table.appendChild(tbody);

    fetch('/api/data-interface/fetch-resource-params', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({resourceID: this.#resourceID, resourceAgent: this.#resourceAgent})
    })
      .then(async res => {
        if (!res.ok) throw new Error(await res.text() || "Unknown error");
        return res.json();
      })
      .then(content => {
        this.#shortdescEl.textContent = " - " + content.Shortdesc.trim();
        const contentOptions = content.Options || [];
        contentOptions.forEach(o => {
          tbody.appendChild(this.#makeParamRow(o.Name, o.Shortdesc, o.Longdesc));
        });

        // Split Longdesc into paragraphs by empty lines or single newline
        const paragraphs = content.Longdesc.trim().split(/\n\n+/);

        paragraphs.forEach(line => {
          const p = document.createElement("p");
          p.textContent = line;           // safe, no HTML injection
          this.#longdescEl.appendChild(p);
        });
      })
      .catch(err => console.error("Failed to init InfoPopup::Parameters", err));

    // What about asyncronization?
    return table;
  }

  #makeHeaderRow(...titles) {
    const tr = document.createElement("tr");
    titles.forEach(t => {
      const th = document.createElement("th");
      th.textContent = t;
      tr.appendChild(th);
    });
    return tr;
  }

  #makeParamRow(name, shortdesc, longdesc) {
    const tr = document.createElement("tr");

    const tdName = document.createElement("td");
    const code = document.createElement("code");
    code.textContent = name;
    tdName.appendChild(code);

    const tdShort = document.createElement("td");
    tdShort.textContent = shortdesc || "";

    const tdLong = document.createElement("td");
    const p = document.createElement("p");
    p.textContent = longdesc || "";
    tdLong.appendChild(p);

    const tdOpt = document.createElement("td"); // empty

    tr.append(tdName, tdShort, tdLong, tdOpt);
    return tr;
  }

  #buildCollapsiblePanel(titleText, bodyID, headingID, expanded, contentNode) {
    const panel = document.createElement("div");
    panel.className = "panel panel-default";

    // Heading
    const heading = document.createElement("div");
    heading.className = "panel-heading";
    heading.id = headingID;
    heading.setAttribute("role", "tab");

    const h4 = document.createElement("h4");
    h4.className = "panel-title";

    const a = document.createElement("a");
    a.href = `#${bodyID}`;
    a.dataset.toggle = "collapse";
    a.dataset.parent = "#agentinfo";
    a.setAttribute("role", "button");
    a.setAttribute("aria-expanded", expanded ? "true" : "false");
    a.setAttribute("aria-controls", bodyID);
    if (!expanded) a.classList.add("collapsed");
    a.textContent = titleText + " ";

    const pull = document.createElement("div");
    pull.className = "pull-right";
    const caret = document.createElement("span");
    caret.className = "caret";
    pull.appendChild(caret);
    a.appendChild(pull);

    h4.appendChild(a);
    heading.appendChild(h4);

    // Body
    const collapse = document.createElement("div");
    collapse.id = bodyID;
    collapse.className = "panel-collapse collapse" + (expanded ? " in" : "");
    collapse.setAttribute("role", "tabpanel");
    collapse.setAttribute("aria-labelledby", headingID);

    const panelBody = document.createElement("div");
    panelBody.className = "panel-body";
    panelBody.appendChild(contentNode);

    collapse.appendChild(panelBody);

    panel.append(heading, collapse);
    return panel;
  }

  #buildFooter() {
    const footer = document.createElement("div");
    footer.className = "modal-footer";

    const btn = document.createElement("button");
    btn.className = "btn btn-default";
    btn.type = "button";
    btn.textContent = "Close";
    btn.onclick = () => this.close();

    footer.appendChild(btn);
    return footer;
  }
}

window.InfoPopup = InfoPopup;
window.showInfoPopup = () => new InfoPopup();
