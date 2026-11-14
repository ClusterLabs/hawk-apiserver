class XInputsKvGroup extends XKvGroupBase {
  #container;
  #name;
  #inputs = [];
  #inputRow; // kvsection-row for the select
  #inputObj;
  #cibObject;
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

    this.#cibObject = this.getAttribute("cib-object");

    this.#optionsApi = this.getAttribute("options-api");
    this.#submitApi = this.getAttribute("submit-api");

    this.#container = this.shadowRoot;

    this.#inputRow = document.createElement("div");
    this.#inputRow.className = "kvsection-row";

    const controlWrap = document.createElement("div");
    controlWrap.className = "kv-control";
    this.#inputRow.appendChild(controlWrap);

    // Input
    this.#inputObj = new Input(this.#inputRow, controlWrap, "", "", true, "", "", "", false);

    // Button
    const addBtn = document.createElement("button");
    addBtn.type = "button";
    addBtn.title = "Add Attribute";
    addBtn.className = "btn btn-default";
    addBtn.onclick = () => this.#onCreateOption();

    // Icon +
    const icon = document.createElement("i");
    icon.className = "fa fa-plus";

    addBtn.appendChild(icon);

    controlWrap.appendChild(addBtn);
    this.#container.appendChild(this.#inputRow);
    const prefix = this.getAttribute("field-prefix");
    if (prefix) this.#inputObj.getHTML().name = `temp_${prefix}`;
    this.#inputObj.getHTML().setAttribute("aria-label", "Instance attribute name");
    this.#inputObj.getHTML().addEventListener("keydown", event => {
      if (event.key === "Enter") { event.preventDefault(); this.#onCreateOption(); }
    });

    // Fetch data
    this.loaded = this.#init()
      .then(() => {
        const values = JSON.parse(this.getAttribute("initial-values") || "{}");
        for (const [name, value] of Object.entries(values))
          this.#createInput(name, "", "", "", "", false, "", "", value);
        return true;
      })
      .catch(err => { console.error("Failed to init x-kvgroup:", err); return false; })
      .finally(() => this.#readyResolve());
  }

  #init(appendBlank) {
    if (!this.#cibObject) return Promise.resolve();
    return fetch(this.#optionsApi, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ CibObject: this.#cibObject })
    })
      .then(async res => {
        if (!res.ok) throw new Error(await res.text() || "Unknown error");
        return res.json();
      })
      .then(content => {
        const nvpairs = content || [];
        nvpairs.forEach(pair => {
          const name = pair.name;
          const cibValue = pair.value;

          this.#createInput(name, "", "", "", "", false, "", cibValue, "");
        });
      });
  }

  #onCreateOption() {
    const cibID = "";
    const cibValue = "";
    const name = this.#inputObj.getFrontendValue();
    if (!name.trim()) return;

    if (this.#createInput(name, "", "", "", "", false, cibID, cibValue, ""))
      this.#inputObj.clearFrontendValue();
  }

  #createInput(name, defaultValue, shortdesc, longdesc, type, required, cibID, cibValue, frontendValue) {
    if (this.#inputs.some(i => i.getName?.() === name)) return true; // already created => ok

    const divContainer = document.createElement("div"); // a new kvsection-row above the select
    divContainer.className = "kvsection-row";
    const controlWrap = document.createElement("div");
    controlWrap.className = "kv-control";
    divContainer.appendChild(controlWrap);

    const input = new Input(divContainer, controlWrap, name, defaultValue,
        required, cibID, cibValue, frontendValue, false);

    const prefix = this.getAttribute("field-prefix");
    if (prefix) input.getHTML().name = `${prefix}[${name}]`;
    input.getHTML().setAttribute("aria-label", name);

    const removeBtn = document.createElement("button");
    removeBtn.type = "button";
    removeBtn.title = "Remove Attribute";
    removeBtn.innerHTML = '<i class="fas fa-minus"></i>';
    removeBtn.className = "btn btn-default";
    removeBtn.onclick = () => {
      divContainer.remove();
      this.#inputs = this.#inputs.filter(i => i !== input);
    }
    controlWrap.appendChild(removeBtn);

    const fieldShortdesc = document.getElementById('field-shortdesc');
    const fieldLongdesc = document.getElementById('field-longdesc');
    divContainer.addEventListener("mouseenter", () => {
      if (!fieldShortdesc || !fieldLongdesc) return;
      const heading = document.createElement("code");
      heading.textContent = name;
      fieldShortdesc.replaceChildren(heading);
      fieldLongdesc.textContent = longdesc || "";
      if (defaultValue) {
        const note = document.createElement("em");
        const value = document.createElement("code");
        value.textContent = defaultValue;
        note.append(" Default: ", value);
        fieldLongdesc.append(note);
      }
    });

    this.#inputs.push(input);

    if (this.#inputRow && this.#container.contains(this.#inputRow)) {
      this.#container.insertBefore(divContainer, this.#inputRow);
    } else {
      this.#container.appendChild(divContainer);
    }

    return true;
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

  // Public: called from node_edit.js before submitting
  submitSanityCheck() {
    const inputs = this.#inputs.map(inputObj => {
      const name = inputObj.getName();
      const value = inputObj.getFrontendValue();
      return { name, value };
    });

    for (const input of inputs) {
      if (String(input.value).trim() === "") {
        return {ok : false, message: `${input.name} must not be empty`};
      }
    }

    return { ok: true, message: "" };
  }

  // Public: called from node_edit.js on clicking Apply button
  submit() {
    const inputs = this.#inputs.map(inputObj => {
      const name = inputObj.getName();
      const value = inputObj.getFrontendValue();
      return { name, value };
    });

    const CibObjectAndKV = {
      CibObject: this.#cibObject,
      nvpair: inputs
    };

    return fetch(this.#submitApi, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(CibObjectAndKV)
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

customElements.define("x-inputs-kvgroup", XInputsKvGroup);
