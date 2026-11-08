function showVirtualIPHelp(event) {
    const row = event.target.closest("[data-param-help]");
    if (!row) return;
    document.querySelectorAll(".virtual-ip-param-help").forEach(section => {
        section.hidden = section.id !== `virtual-ip-help-${row.dataset.paramHelp}`;
    });
}

class WizardAdvancedParameters {
    constructor(container, parameters, values) {
        this.container = container;
        this.parameters = parameters;
        this.values = {};
        for (const parameter of parameters) {
            if (Object.hasOwn(values, parameter.name)) this.values[parameter.name] = values[parameter.name];
            else if (parameter.value) this.values[parameter.name] = parameter.value;
        }
        this.render();
    }

    button(parent, label, icon, callback) {
        const button = document.createElement("button");
        button.type = "button";
        button.className = "btn btn-default";
        button.title = label;
        button.setAttribute("aria-label", label);
        const symbol = document.createElement("i");
        symbol.className = `fa fa-${icon} fa-fw`;
        symbol.setAttribute("aria-hidden", "true");
        button.append(symbol);
        button.addEventListener("click", callback);
        parent.append(button);
    }

    render() {
        this.container.replaceChildren();
        for (const [name, value] of Object.entries(this.values)) {
            const parameter = this.parameters.find(parameter => parameter.name === name);
            const row = document.createElement("div");
            row.className = "form-group";
            row.dataset.paramHelp = name;
            const label = document.createElement("label");
            label.className = "col-sm-5 control-label";
            label.htmlFor = `virtual-ip.${name}`;
            label.textContent = name;
            const column = document.createElement("div");
            column.className = "col-sm-7";
            const group = document.createElement("div");
            group.className = "wizard-advanced-group";
            const input = document.createElement(parameter.type === "boolean" ? "select" : "input");
            input.className = "form-control";
            input.id = input.name = `virtual-ip.${name}`;
            if (parameter.type === "boolean") {
                input.add(new Option("Yes", "true"));
                input.add(new Option("No", "false"));
            } else {
                input.type = parameter.type === "integer" ? "number" : "text";
            }
            input.value = value || "";
            input.addEventListener("input", () => this.values[name] = input.value);
            input.addEventListener("change", () => this.values[name] = input.value);
            group.append(input);
            this.button(group, `Remove ${name}`, "minus", () => {
                delete this.values[name];
                this.render();
            });
            column.append(group);
            row.append(label, column);
            this.container.append(row);
        }
        const remaining = this.parameters.filter(parameter => !Object.hasOwn(this.values, parameter.name))
            .sort((a, b) => a.name.localeCompare(b.name));
        if (!remaining.length) return;
        const row = document.createElement("div");
        row.className = "form-group addition";
        const column = document.createElement("div");
        column.className = "col-sm-7 col-sm-offset-5";
        const group = document.createElement("div");
        group.className = "wizard-advanced-group";
        const select = document.createElement("select");
        select.className = "form-control ignore";
        select.name = "temp[selector]";
        select.setAttribute("aria-label", "Advanced parameter");
        select.add(new Option("", ""));
        for (const parameter of remaining) select.add(new Option(parameter.name, parameter.name));
        const add = () => {
            const parameter = remaining.find(parameter => parameter.name === select.value);
            if (!parameter) return;
            this.values[parameter.name] = parameter.example || "";
            this.render();
            const input = document.getElementById(`virtual-ip.${parameter.name}`);
            input?.dispatchEvent(new Event("focusin", { bubbles: true }));
        };
        select.addEventListener("change", add);
        group.append(select);
        this.button(group, "Create", "plus", add);
        column.append(group);
        row.append(column);
        this.container.append(row);
    }
}

const virtualIPForm = document.getElementById("edit_wizard_virtual-ip");
virtualIPForm.addEventListener("mouseover", showVirtualIPHelp);
virtualIPForm.addEventListener("focusin", showVirtualIPHelp);
const resourceID = document.getElementById("virtual-ip.id");
resourceID?.addEventListener("input", () => {
    resourceID.setCustomValidity(resourceID.value.includes(" ") ? "Must be a valid resource ID" : "");
});
const advanced = document.getElementById("virtual-ip-advanced-fields");
if (advanced) new WizardAdvancedParameters(advanced, window.virtualIPData.parameters, window.virtualIPData.values);
document.querySelectorAll("pre code.crmsh").forEach(block => hljs.highlightBlock(block));
