class Alert {
    constructor(idInput, pathInput, recipients, params, meta) {
        this.idInput = idInput;
        this.pathInput = pathInput;
        this.recipients = recipients;
        this.params = params;
        this.meta = meta;
    }

    async create() {
        const response = await fetch("/api/cib/alert/create", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                id: this.idInput.value.trim(), path: this.pathInput.value,
                recipients: this.recipients, params: this.params.getFrontendKValues(),
                meta: this.meta.getFrontendKValues(),
            }),
        });
        if (!response.ok) throw new Error(await response.text() || "Unable to create alert.");
        const result = await response.json();
        if (result.status !== "ok") throw new Error(result.error || "Unable to create alert.");
        window.location.href = result.redirect;
    }
}
