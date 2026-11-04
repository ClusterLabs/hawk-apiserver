class Tag {
    constructor(idInput, objectsList) {
        this.idInput = idInput;
        this.objectsList = objectsList;
    }

    async create() {
        const tag = {
            id: this.idInput.value.trim(),
            refs: Array.from(this.objectsList.querySelectorAll('input[name="tag[refs][]"]:checked'))
                .map(input => input.value),
        };
        const response = await fetch("/api/cib/tag/create", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(tag),
        });
        if (!response.ok) throw new Error(await response.text() || "Unable to create tag.");
        const result = await response.json();
        if (result.status !== "ok") throw new Error(result.error || "Unable to create tag.");
        window.location.href = result.redirect;
    }
}
