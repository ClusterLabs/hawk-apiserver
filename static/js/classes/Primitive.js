class Primitive {
  #paramsKVGroup;
  #metaKVGroup;
  #operationsKVGroup;
  constructor(resourceIDInput, classSelect, providerSelect, typeSelect,
              paramsKVGroup, metaKVGroup, operationsKVGroup) {
    this.resourceIDInput = resourceIDInput;
    this.classSelect = classSelect;
    this.providerSelect = providerSelect;
    this.typeSelect = typeSelect;

    this.#paramsKVGroup = paramsKVGroup;
    this.#metaKVGroup = metaKVGroup;
    this.#operationsKVGroup = operationsKVGroup;
  }

  #getSelectedValue(xSelectElem) {
    const selected = xSelectElem.selectedOption();
    return selected ? selected.getName() : "";
  }

  async create() {
    const newID = this.resourceIDInput.value;

    const className = this.#getSelectedValue(this.classSelect);
    const providerName = this.#getSelectedValue(this.providerSelect);
    const typeName = this.#getSelectedValue(this.typeSelect);

    const instanceAttrs = this.#paramsKVGroup.getFrontendKValues();
    const metaAttrs = this.#metaKVGroup.getFrontendKValues();
    const operations = this.#operationsKVGroup.getFrontendOperations(); // assumes such method exists

    const instanceAttrsList = [];
    for (const name in instanceAttrs) {
      instanceAttrsList.push({ id: `${newID}-instance_attributes-${name}`, name, value: instanceAttrs[name] });
    }

    const metaAttrsList = [];
    for (const name in metaAttrs) {
      metaAttrsList.push({ id: `${newID}-meta_attributes-${name}`, name, value: metaAttrs[name] });
    }

    const operationsList = operations.map(op => ({ id: `${newID}-operation-${op.name}`, ...op }));

    const primitive = {
      id: newID,
      class: className,
      provider: providerName,
      type: typeName,
      instance_attributes: {
        id: `${newID}-instance_attributes`,
        nvpair: instanceAttrsList
      },
      meta_attributes: {
        id: `${newID}-meta_attributes`,
        nvpair: metaAttrsList
      },
      operations: operationsList
    };

    return fetch("/api/cib/create-primitive/", {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(primitive)
    })
      .then(async res => {
        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || "Unknown error");
        }
        return res.json();
      })
      .then(status => {
        console.log("Create status:", status);
        window.location.href = `/cib/live/primitives/${newID}/edit?flash=created`;
      })
      .catch(err => {
        console.error("Create error:", err);
        window.location.href = `/cib/live/primitives/${newID}/edit?flash=error&msg=` + encodeURIComponent(err.message);
      });
  }
}

window.Primitive = Primitive;
