class Location {
    constructor(idInput, resources, discovery, rules) {
        this.idInput = idInput;
        this.resources = resources;
        this.discovery = discovery;
        this.rules = rules;
    }

    async create() {
        const response = await fetch("/api/cib/location/create", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                id: this.idInput.value.trim(),
                resources: Array.from(this.resources.selectedOptions, option => option.value),
                discovery: this.discovery.value,
                rules: this.rules.getValues(),
            }),
        });
        if (!response.ok) throw new Error(await response.text() || "Unable to create constraint.");
        const result = await response.json();
        if (result.status !== "ok") throw new Error(result.error || "Unable to create constraint.");
        window.location.href = result.redirect;
    }
}
