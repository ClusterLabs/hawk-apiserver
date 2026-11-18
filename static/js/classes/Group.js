class Group {
    constructor(idInput, childrenList, metaAttributes) {
        this.idInput = idInput;
        this.childrenList = childrenList;
        this.metaAttributes = metaAttributes;
    }

    async create() {
        const result = await this.save("create");
        window.location.href = result.redirect;
    }

    async update() {
        return this.save("update");
    }

    async save(operation) {
        const group = {
            id: this.idInput.value.trim(),
            children: Array.from(this.childrenList.querySelectorAll('input[name="group[children][]"]:checked'))
                .map(input => input.value),
            meta: this.metaAttributes.getFrontendKValues(),
        };
        const response = await fetch(`${document.getElementById("group-create-form")?.dataset.apiBase || "/api/cib/group"}/${operation}`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(group),
        });
        if (!response.ok) throw new Error(await response.text() || "Unable to save group.");
        const result = await response.json();
        if (result.status !== "ok") throw new Error(result.error || "Unable to save group.");
        return result;
    }
}
