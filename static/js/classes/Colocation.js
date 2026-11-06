class Colocation {
    constructor(id, score, nodeAttribute, resources) {
        this.id = id;
        this.score = score;
        this.nodeAttribute = nodeAttribute;
        this.resources = resources;
    }

    async create() {
        const response = await fetch("/api/cib/colocation/create", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                id: this.id.value.trim(),
                score: this.score.value,
                node_attr: this.nodeAttribute.value,
                resources: this.resources.getValues(),
            }),
        });
        if (!response.ok) throw new Error(await response.text() || "Unable to create constraint.");
        const result = await response.json();
        if (result.status !== "ok") throw new Error(result.error || "Unable to create constraint.");
        window.location.href = result.redirect;
    }
}
