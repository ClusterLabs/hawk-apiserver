// Resource sets follow Hawk's constraintList ordering and shared role behavior.
class ConstraintResources {
    constructor(container, available) {
        this.container = container;
        this.remaining = [...available];
        this.entries = [];
        this.render();
    }

    getValues() {
        return this.entries.map(entry => ({
            sequential: entry.resources.length === 1,
            action: entry.action,
            resources: [...entry.resources],
        }));
    }

    select(options, value, label) {
        const select = document.createElement("select");
        select.className = "form-control";
        select.setAttribute("aria-label", label);
        select.add(new Option("", ""));
        for (const [key, text] of options) select.add(new Option(text, key));
        select.value = value;
        return select;
    }

    roles(value) {
        return this.select([["Started", "Started"], ["Master", "Promoted"], ["Slave", "Unpromoted"], ["Stopped", "Stopped"]], value, "Role");
    }

    button(parent, title, icon, callback, disabled = false) {
        const button = document.createElement("button");
        button.type = "button";
        button.className = "btn btn-default";
        button.title = title;
        button.setAttribute("aria-label", title);
        button.disabled = disabled;
        const symbol = document.createElement("i");
        symbol.className = `fa fa-${icon} fa-fw`;
        symbol.setAttribute("aria-hidden", "true");
        button.append(symbol);
        button.addEventListener("click", () => {
            callback();
            this.container.dispatchEvent(new Event("change", { bubbles: true }));
        });
        parent.append(button);
    }

    render() {
        this.container.replaceChildren();
        this.entries.forEach((entry, setIndex) => {
            const well = document.createElement("div");
            well.className = "well well-sm";
            entry.resources.forEach((resource, index) => {
                const row = document.createElement("div");
                row.className = "colocation-resource-row";
                const label = document.createElement("label");
                label.textContent = resource;
                const group = document.createElement("div");
                group.className = "colocation-input-group";
                const role = this.roles(entry.action);
                role.setAttribute("aria-label", `Role for ${resource}`);
                role.addEventListener("change", () => {
                    entry.action = role.value;
                    this.render();
                });
                group.append(role);
                const first = setIndex === 0 && index === 0;
                this.button(group, index === 0 ? "Link with entry above" : "Unlink from entry above", index === 0 ? "link" : "cut", () => {
                    if (index === 0) {
                        this.entries[setIndex - 1].resources.push(...entry.resources);
                        this.entries.splice(setIndex, 1);
                    } else {
                        this.entries.splice(setIndex + 1, 0, { action: entry.action, resources: [resource] });
                        entry.resources.splice(index, 1);
                    }
                    this.render();
                }, first);
                this.button(group, "Swap with entry above", "arrow-up", () => {
                    if (index === 0) {
                        [this.entries[setIndex - 1], this.entries[setIndex]] = [entry, this.entries[setIndex - 1]];
                    } else {
                        [entry.resources[index - 1], entry.resources[index]] = [resource, entry.resources[index - 1]];
                    }
                    this.render();
                }, first);
                this.button(group, "Remove", "minus", () => {
                    this.remaining.push(resource);
                    entry.resources.splice(index, 1);
                    if (!entry.resources.length) this.entries.splice(setIndex, 1);
                    this.render();
                });
                row.append(label, group);
                well.append(row);
            });
            this.container.append(well);
        });
        if (!this.remaining.length) return;
        const row = document.createElement("div");
        row.className = "colocation-resource-row colocation-addition";
        const resource = this.select(this.remaining.map(id => [id, id]), "", "Resource");
        resource.name = "temp[selector]";
        const group = document.createElement("div");
        group.className = "colocation-input-group";
        const role = this.roles("Started");
        role.name = "temp[value]";
        const add = () => {
            if (!resource.value) return;
            this.entries.push({ action: role.value, resources: [resource.value] });
            this.remaining.splice(this.remaining.indexOf(resource.value), 1);
            this.render();
        };
        // Ruby adds the selection on change as well as through the plus button.
        resource.addEventListener("change", add);
        role.addEventListener("change", add);
        group.append(role);
        this.button(group, "Create", "plus", add);
        row.append(resource, group);
        this.container.append(row);
    }
}
