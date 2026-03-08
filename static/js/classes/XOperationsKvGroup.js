class XOperationsKvGroup extends window.XKvGroupBase {
  #container;
  #inputs = [];
  #selectObj;
  #resourceID;
  #optionsApi;
  #submitApi;
  #selectRow;

  constructor() {
    super();
  }

  connectedCallback() {
    // Extract attributes
    this.#optionsApi = this.getAttribute("options-api");
    this.#submitApi = this.getAttribute("submit-api");

    this.#container = this.shadowRoot;

    this.#selectRow = document.createElement("div");
    this.#selectRow.className = "kvsection-row";

    const controlWrap = document.createElement("div");
    controlWrap.className = "kv-control";
    this.#selectRow.appendChild(controlWrap);

    // Select
    const onclickHandler = () => {
      const optionObj = this.#selectObj.selectedOption();
      if (!optionObj) return;
      new OperationPopup(this, optionObj.getName(), optionObj.getDefaultValue()
        , optionObj.getShortDesc(), optionObj.getLongDesc(), optionObj.getType()
        , optionObj.getPossibleValues(), optionObj.getRequired(), "", {}, null, false);
      // reset back to blank
      this.#selectObj.setValue("");
    }

    this.#selectObj = new Select(this.#selectRow, controlWrap, "", "", "", "", true, true, "");
    this.#selectObj.setOnChangeEvent(() => onclickHandler());
    const selectEl = this.#selectObj.getHTML();
    selectEl.classList.add("form-control");

    // Button
    const addBtn = document.createElement("button");
    addBtn.className = "btn btn-default";
    addBtn.onclick = onclickHandler;

    // Icon +
    const icon = document.createElement("i");
    icon.className = "fa fa-plus";
    addBtn.appendChild(icon);

    controlWrap.appendChild(addBtn);
    this.#container.appendChild(this.#selectRow);

    this.#resourceID = window.resourceData?.ResourceID || "";
    const resourceAgent = window.resourceData?.ResourceAgent || "";

    this.#init({ ResourceID: this.#resourceID, ResourceAgent: resourceAgent }, true);
  }

  #createOption(name, type, defaultValue, shortdesc, longdesc, possibleValues) {
    const optionObj = new SelectOption(name, type, defaultValue, shortdesc, longdesc, possibleValues);
    this.#selectObj.appendOption(optionObj);
  }

  #init(apiArguments, appendBlank) {
    if (appendBlank) this.#selectObj.appendBlankOption();

    fetch(this.#optionsApi, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(apiArguments)
    })
      .then(res => res.json())
      .then(content => this.#populateOptions(content.Options || []))
      .catch(err => console.error("Failed to initialize x-operations-kvgroup:", err));
  }

  #populateOptions(options) {
    options.forEach(o => {
      this.#createOption(o.Name, o.Type, o.DefaultValue, o.Shortdesc, o.Longdesc, o.PossibleValues);
      if (o.CibID) {
        // o.CibNameValues array --> kvMap object
        const kvMap = Object.fromEntries(
          (o.CibNameValues || []).map(nv => [nv.Name, nv.Value])
        );
        this.createInput(o.Name, o.DefaultValue, o.Shortdesc, o.Longdesc, o.Type,
          o.PossibleValues, o.Required, o.CibID, kvMap);
      }
    });
    this.#selectObj.sort();
    this.#selectObj.setValue("");
  }

  submitSanityCheck() {
    for (const inputObj of this.#inputs) {
      const name = inputObj.getName();
      const kvalues = inputObj.getFrontendKValues();

      if (!kvalues || Object.keys(kvalues).length === 0) {
        return { ok: false, message: `${name} must not be empty` };
      }
    }

    return { ok: true, message: "" };
  }

  // submit the changes to the cib.xml
  submit() {
    /* Always submit, even with an empty operations list —
     * this signals the backend to remove existing operations. */
    const operations = this.#inputs.map(inputObj => {
      const opID = inputObj.getCibID();
      const attrName = inputObj.getName();
      const kvalues = inputObj.getFrontendKValues();
      return { id: opID, name: attrName, ...kvalues }; // append kvalues
    });

    const primitive = {
      id: this.#resourceID,
      operations: operations
    };

    return fetch(this.#submitApi, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(primitive)
    })
      .then(async res => {
        if (!res.ok) throw new Error(await res.text() || "Unknown error");
        return res.json();
      })
      .then(status => {
        console.log("x-operations-kvgroup submit status:", status);
        return status;
      });
  }

  updateInput(operationInput, kvMap) {
    operationInput.update(kvMap);
  }

  createInput(name, defaultValue, shortdesc, longdesc, type
    , possibleValues, required, cibID, kvMap) {

    const newRow = new OperationInput(name, defaultValue, shortdesc, longdesc, type
      , possibleValues, required, cibID, kvMap, this);

    if (this.#selectRow && this.#container.contains(this.#selectRow)) {
      this.#container.insertBefore(newRow.getHTML(), this.#selectRow);
    } else {
      this.#container.appendChild(newRow.getHTML());
    }

    this.#inputs.push(newRow);
  }

  restoreOption(inputObj) {
    this.#inputs = this.#inputs.filter(i => i !== inputObj);
  }

  getFrontendOperations() {
    return this.#inputs.map(input => {
      return {
        name: input.getName(),
        ...input.getFrontendKValues()
      };
    });
  }

}

customElements.define("x-operations-kvgroup", XOperationsKvGroup);
