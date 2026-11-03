class Primitive {
    #resourceIDInput;
    #classSelect;
    #providerSelect;
    #typeSelect;
    #paramsKVGroup;
    #metaKVGroup;
    #operationsKVGroup;
    #utilizationsKVGroup;
    constructor(resourceIDInput, classSelect, providerSelect, typeSelect,
        paramsKVGroup, metaKVGroup, operationsKVGroup, utilizationsKVGroup) {
        this.#resourceIDInput = resourceIDInput;
        this.#classSelect = classSelect;
        this.#providerSelect = providerSelect;
        this.#typeSelect = typeSelect;

        this.#paramsKVGroup = paramsKVGroup;
        this.#metaKVGroup = metaKVGroup;
        this.#operationsKVGroup = operationsKVGroup;
        this.#utilizationsKVGroup = utilizationsKVGroup;
    }

    async create() {
        const newID = this.#resourceIDInput.value.trim();

        const className = this.#classSelect.value || "";
        const providerName = this.#providerSelect.value || "";
        const typeName = this.#typeSelect.value || "";

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
            utilizations: Object.entries(this.#utilizationsKVGroup?.getFrontendKValues() || {}).map(([name, value]) => ({ id: `${newID}-utilization-${name}`, name, value })),
            operations: operationsList
        };

        return fetch("/api/cib/primitive/create", {
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
                if (status.error || status.status !== "ok") throw new Error(status.error || "Unable to create primitive.");
                window.location.href = `/cib/live/primitives/${encodeURIComponent(newID)}/edit?flash=created`;
            });
    }
}
