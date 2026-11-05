// Alert agents accept arbitrary instance attribute names. Meta attributes use
// the existing x-kvgroup instead, with the mapping from Alert.mapping in Ruby.
class AlertAttributes {
    constructor(container, prefix, values = {}) {
        this.container = container;
        this.prefix = prefix;
        this.inputs = new Map();
        this.adder = document.createElement("div");
        this.adder.className = "kvsection-row";
        const control = document.createElement("div");
        control.className = "alert-control input-group";
        this.nameInput = document.createElement("input");
        this.nameInput.className = "form-control";
        this.nameInput.name = `temp_${prefix}`;
        this.nameInput.setAttribute("aria-label", "Instance attribute name");
        const add = document.createElement("button");
        add.type = "button";
        add.className = "btn btn-default";
        add.title = "Add Attribute";
        add.innerHTML = '<i class="fa fa-plus" aria-hidden="true"></i>';
        add.addEventListener("click", () => {
            const name = this.nameInput.value;
            if (name) this.add(name, "");
            this.nameInput.value = "";
        });
        this.nameInput.addEventListener("keydown", event => {
            if (event.key === "Enter") { event.preventDefault(); add.click(); }
        });
        control.append(this.nameInput, add);
        this.adder.append(control);
        container.append(this.adder);
        for (const [name, value] of Object.entries(values)) this.add(name, value);
    }

    add(name, value) {
        if (this.inputs.has(name)) { this.inputs.get(name).value = value; return; }
        const row = document.createElement("div");
        row.className = "kvsection-row";
        const label = document.createElement("label");
        label.textContent = name;
        const control = document.createElement("div");
        control.className = "alert-control input-group";
        const input = document.createElement("input");
        input.className = "form-control";
        input.name = `${this.prefix}[${name}]`;
        input.value = value;
        input.setAttribute("aria-label", name);
        const remove = document.createElement("button");
        remove.type = "button";
        remove.className = "btn btn-default";
        remove.title = "Remove Attribute";
        remove.innerHTML = '<i class="fa fa-minus" aria-hidden="true"></i>';
        remove.addEventListener("click", () => { this.inputs.delete(name); row.remove(); });
        control.append(input, remove);
        row.append(label, control);
        this.container.insertBefore(row, this.adder);
        this.inputs.set(name, input);
    }

    getFrontendKValues() {
        return Object.fromEntries(Array.from(this.inputs, ([name, input]) => [name, input.value]));
    }
}
