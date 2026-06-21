class XInputsKvGroup extends window.XKvGroupBase {
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
    addBtn.className = "btn btn-default";
    addBtn.onclick = () => this.#onCreateOption();

    // Icon +
    const icon = document.createElement("i");
    icon.className = "fa fa-plus";

    addBtn.appendChild(icon);

    controlWrap.appendChild(addBtn);
    this.#container.appendChild(this.#inputRow);

    // Fetch data
    this.#init()
      .catch(err => console.error("Failed to init x-kvgroup:", err))
      .finally(() => this.#readyResolve()); // TODO: what is () in #readyResolve() ?
  }

  #init(appendBlank) {
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

          // TODO: test
          this.#createInput(name, "", "", "", "", false, "", cibValue, "");
        });
      });
  }

  #onCreateOption() {
    const cibID = "";
    const cibValue = "";
    const name = this.#inputObj.getFrontendValue();

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

    const initial = frontendValue || cibValue || defaultValue || "";

    const removeBtn = document.createElement("button");
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
      fieldShortdesc.innerHTML = `<code>${name}</code>`;
      fieldLongdesc.innerHTML = longdesc  || "";
      if (defaultValue) {
        fieldLongdesc.innerHTML += `<em> Default: <code>${defaultValue}</code></em>`;
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
