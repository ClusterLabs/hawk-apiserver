class XKvGroup extends window.XKvGroupBase {
  #container;
  #name;
  #inputs = [];
  #selectRow; // kvsection-row for the select
  #selectObj;
  #resourceID;
  #resourceAgent;
  #optionsApi;
  #submitApi;
  #readyResolve;
  #ready; // => initialized, now you can createOption(...)
  #inited = false;

  constructor() {
    super();
    this.#ready = new Promise(res => (this.#readyResolve = res));
  }

  connectedCallback() {
    if (this.#inited) return;
    this.#inited = true;

    this.#name = this.getAttribute("name");
    this.#resourceID = this.getAttribute("resource-id");
    this.#resourceAgent = this.getAttribute("resource-agent");
    this.#optionsApi = this.getAttribute("options-api");
    this.#submitApi = this.getAttribute("submit-api");

    this.#container = this.shadowRoot;

    this.#selectRow = document.createElement("div");
    this.#selectRow.className = "kvsection-row";

    const controlWrap = document.createElement("div");
    controlWrap.className = "kv-control";
    this.#selectRow.appendChild(controlWrap);

    // Select
    // #FIXME?: class Select already has it's own Init method with it's own api-entry,
    // but override it here. Obviously it's code repeating.
    this.#selectObj = new Select(this.#selectRow, controlWrap, "", "", "", "", true, true, "");
    this.#selectObj.setOnChangeEvent(() => this.#onCreateOption());

    // Button
    const addBtn = document.createElement("button");
    addBtn.className = "btn btn-default";
    addBtn.onclick = () => this.#onCreateOption();

    // Icon +
    const icon = document.createElement("i");
    icon.className = "fa fa-plus";

    addBtn.appendChild(icon);

    controlWrap.appendChild(addBtn);
    this.#container.appendChild(this.#selectRow);

    // Fetch data
    this.#resourceID = window.resourceData?.ResourceID || "";
    this.#resourceAgent = window.resourceData?.ResourceAgent || "";
    this.#init(true)
      .catch(err => console.error("Failed to init x-kvgroup:", err))
      .finally(() => this.#readyResolve());
  }

  #init(appendBlank) {
    if (appendBlank) {
      this.#selectObj.appendBlankOption();
    }

    return fetch(this.#optionsApi, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ResourceID: this.#resourceID, ResourceAgent: this.#resourceAgent})
    })
      .then(async res => {
        if (!res.ok) throw new Error(await res.text() || "Unknown error");
        return res.json();
      })
      .then(content => {
        const contentOptions = content.Options || [];
        contentOptions.forEach(o => {
          const optionObj = new SelectOption(o.Name, o.Type, o.DefaultValue, o.Shortdesc, o.Longdesc, o.PossibleValues);
          this.#selectObj.appendOption(optionObj);
          if (o.CibID) {
            const frontendValue = "";
            this.#createInput(o.Name, o.DefaultValue, o.Shortdesc, o.Longdesc, o.Type,
              o.PossibleValues, o.Required, o.CibID, o.CibValue, frontendValue);
            this.#selectObj.hideOption(o.Name);
          }
        });
        this.#selectObj.sort();
        this.#selectObj.setValue("");
        this.#updateSelectVisibility();
      });
  }

  #updateSelectVisibility() {
    const show = !this.#selectObj.empty();
    this.#selectRow.style.display = show ? "" : "none";
  }

  // Ensure interval/timeout rows exist if those options are available
  // (designed specificaly for OperationPopup)
  async enforceIntervalTimeout() {
    await this.#ready;
    for (const k of ["interval", "timeout"]) {
      if (this.#selectObj.getOption(k)) await this.createOption(k, "");
    }
  }

  async applyKvMap(kvMap = {}) {
    await this.#ready;
    for (const [k, v] of Object.entries(kvMap)) {
      await this.createOption(k, v);
    }
  }

  // Public: create an input/select by option name (stable key)
  async createOption(name, frontendValue = "") {
    if (!name) return null;
    await this.#ready;

    const op = this.#selectObj.getOption(name);
    if (!op) return false;

    if (this.#inputs.some(i => i.getName?.() === name)) return true; // already created => ok

    const cibID = "";
    const cibValue = "";

    this.#createInput(op.getName(), op.getDefaultValue(), op.getShortDesc(), op.getLongDesc(),
      op.getType(), op.getPossibleValues(), op.getRequired(), cibID, cibValue, frontendValue);

    // Match behavior of UI selection
    op.hide();
    this.#updateSelectVisibility();
    return true;
  }

  // TODO: utilize createOption
  #onCreateOption() {
    const op = this.#selectObj.selectedOption();
    if (!op) return;

    const cibID = "";
    const cibValue = "";
    const frontendValue = "";

    this.#createInput(op.getName(), op.getDefaultValue(), op.getShortDesc(), op.getLongDesc(), op.getType(),
      op.getPossibleValues(), op.getRequired(), cibID, cibValue, frontendValue);

    op.hide();
    this.#updateSelectVisibility();
  }

  #createInput(name, defaultValue, shortdesc, longdesc, type, possibleValues, required, cibID, cibValue, frontendValue) {
    var selectOrInput;
    const divContainer = document.createElement("div"); // a new kvsection-row above the select
    divContainer.className = "kvsection-row";
    const controlWrap = document.createElement("div");
    controlWrap.className = "kv-control";
    divContainer.appendChild(controlWrap);

    if (possibleValues?.length) {
      selectOrInput = new Select(divContainer, controlWrap, name, "", "", "", true, true, cibID);
      possibleValues.forEach(value => {
        const selectOption = new SelectOption(value, "", "", "", "", "");
        selectOrInput.appendOption(selectOption);
      });
    } else if (type == "boolean") {
      const OptionTrue = new SelectOption("true", "", "", "", "", "");
      const OptionFalse = new SelectOption("false", "", "", "", "", "");
      selectOrInput = new Select(divContainer, controlWrap, name, "", "", "", true, true, cibID);
      selectOrInput.appendOption(OptionTrue);
      selectOrInput.appendOption(OptionFalse);
    } else {
      selectOrInput = new Input(divContainer, controlWrap, name, defaultValue,
        required, cibID, cibValue, frontendValue, false);
    }

    const initial = frontendValue || cibValue || defaultValue || "";
    if (selectOrInput instanceof Select) {
      selectOrInput.setValue(initial);
    }

    const removeBtn = document.createElement("button");
    removeBtn.innerHTML = '<i class="fas fa-minus"></i>';
    removeBtn.className = "btn btn-default";
    removeBtn.onclick = () => {
      divContainer.remove();
      this.restoreOption(selectOrInput);
    }
    controlWrap.appendChild(removeBtn);

    const fieldShortdesc = document.getElementById('field-shortdesc');
    const fieldLongdesc = document.getElementById('field-longdesc');
    divContainer.addEventListener("mouseenter", () => {
      if (!fieldShortdesc || !fieldLongdesc) return;
      fieldShortdesc.innerHTML = `<code>${name}</code>`;
      fieldLongdesc.innerHTML = longdesc  || "";
      if (defaultValue) {
        fieldLongdesc.innerHTML += `<em> Default: <code>${defaultValue}</code></em>`;
      }
    });

    this.#inputs.push(selectOrInput);

    if (this.#selectRow && this.#container.contains(this.#selectRow)) {
      this.#container.insertBefore(divContainer, this.#selectRow);
    } else {
      this.#container.appendChild(divContainer);
    }
  }

  restoreOption(inputObj) {
    this.#inputs = this.#inputs.filter(i => i !== inputObj);
    this.#selectObj.showOption(inputObj.getName());
    this.#selectObj.setValue("");
    this.#updateSelectVisibility();
  }

  getFrontendKValues() {
    const result = {};
    this.#inputs.forEach(input => {
      result[input.getName()] = input.getFrontendValue();
    });
    return result;
  }

  getInputs() {
    return this.#inputs;
  }

  ready() { return this.#ready; }

  submitSanityCheck() {
    const attributes = this.#inputs.map(inputObj => {
      const opID = inputObj.getCibID();
      const attrName = inputObj.getName();
      const value = inputObj.getFrontendValue();
      return { id: opID, name: attrName, value };
    });

    for (const input of attributes) {
      if (String(input.value).trim() === "") {
        return {ok : false, message: `${input.name} must not be empty`};
      }
    }

    return { ok: true, message: "" };
  }

  submit() {
    const attributes = this.#inputs.map(inputObj => {
      const opID = inputObj.getCibID();
      const attrName = inputObj.getName();
      const value = inputObj.getFrontendValue();
      return { id: opID, name: attrName, value };
    });

    const primitive = {
      id: this.#resourceID,
      [this.#name]: {
        id: `${this.#resourceID}-${this.#name}`,
        nvpair: attributes
      }
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
        console.log("x-kvgroup submit status:", status);
        return status;
      });
  }
}

customElements.define("x-kvgroup", XKvGroup);
