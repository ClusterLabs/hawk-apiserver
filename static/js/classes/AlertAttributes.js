// Alert agents use arbitrary names, provided by the shared free-form KV control.
class AlertAttributes {
    constructor(container, prefix, values = {}) {
        this.group = document.createElement("x-inputs-kvgroup");
        this.group.setAttribute("field-prefix", prefix);
        this.group.setAttribute("initial-values", JSON.stringify(values));
        container.append(this.group);
    }

    get loaded() {
        return this.group.loaded;
    }

    getFrontendKValues() {
        return this.group.getFrontendKValues();
    }
}
