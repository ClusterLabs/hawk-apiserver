class Alert {
    constructor(idInput, pathInput, recipients, params, meta) {
        this.idInput = idInput;
        this.pathInput = pathInput;
        this.recipients = recipients;
        this.params = params;
        this.meta = meta;
    }

    async create() {
        const result = await this.save("create");
        window.location.href = result.redirect;
    }

    async update() {
        return this.save("update");
    }

    async save(operation) {
        const response = await fetch(`/api/cib/alert/${operation}`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                id: this.idInput.value.trim(), path: this.pathInput.value,
                recipients: this.recipients, params: this.params.getFrontendKValues(),
                meta: this.meta.getFrontendKValues(),
            }),
        });
        if (!response.ok) throw new Error(await response.text() || "Unable to save alert.");
        const result = await response.json();
        if (result.status !== "ok") throw new Error(result.error || "Unable to save alert.");
        return result;
    }
}
