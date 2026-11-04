class Group {
    constructor(idInput, childrenList, metaAttributes) {
        this.idInput = idInput;
        this.childrenList = childrenList;
        this.metaAttributes = metaAttributes;
    }

    async create() {
        const group = {
            id: this.idInput.value.trim(),
            children: Array.from(this.childrenList.querySelectorAll('input[name="group[children][]"]:checked'))
                .map(input => input.value),
            meta: this.metaAttributes.getFrontendKValues(),
        };
        const response = await fetch("/api/cib/group/create", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(group),
        });
        if (!response.ok) throw new Error(await response.text() || "Unable to create group.");
        const result = await response.json();
        if (result.status !== "ok") throw new Error(result.error || "Unable to create group.");
        window.location.href = result.redirect;
    }
}
