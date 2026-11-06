class LocationRules {
    constructor(container) {
        this.container = container;
        this.setValues([{ score: "INFINITY", role: "", operator: "and", expressions: [] }]);
    }

    setValues(values) {
        this.rules = values;
        this.render();
    }

    getValues() {
        return this.rules;
    }

    field(parent, labelText, value, options, onChange) {
        const row = document.createElement("div");
        row.className = "kvsection-row";
        const label = document.createElement("label");
        label.textContent = labelText;
        const input = document.createElement(options ? "select" : "input");
        input.className = "form-control location-control";
        input.setAttribute("aria-label", labelText);
        if (options) {
            for (const [value, text] of options) input.add(new Option(text, value));
        }
        input.value = value || "";
        input.addEventListener("input", () => onChange(input.value));
        input.addEventListener("change", () => onChange(input.value));
        row.append(label, input);
        parent.append(row);
        return input;
    }

    button(parent, title, icon, callback) {
        const button = document.createElement("button");
        button.className = "btn btn-default btn-xs";
        button.type = "button";
        button.title = title;
        button.setAttribute("aria-label", title);
        button.innerHTML = `<i class="fa fa-${icon}" aria-hidden="true"></i>`;
        button.addEventListener("click", callback);
        parent.append(button);
    }

    render() {
        this.container.replaceChildren();
        this.rules.forEach((rule, index) => {
            const fieldset = document.createElement("fieldset");
            const legend = document.createElement("legend");
            legend.textContent = "Rule";
            const buttons = document.createElement("span");
            buttons.className = "pull-right";
            this.button(buttons, "Add Rule", "plus", () => {
                this.rules.splice(index + 1, 0, { score: "INFINITY", role: "", operator: "and", expressions: [] });
                this.render();
            });
            this.button(buttons, "Remove Rule", "minus", () => {
                if (this.rules.length > 1) { this.rules.splice(index, 1); this.render(); }
            });
            legend.append(buttons);
            fieldset.append(legend);
            this.field(fieldset, "Score", rule.score, null, value => rule.score = value);
            this.field(fieldset, "Role", rule.role, [["", ""], ["Started", "Started"], ["Master", "Promoted"], ["Slave", "Promotable"]], value => rule.role = value);
            this.field(fieldset, "Boolean Operator", rule.operator || "and", [["and", "and"], ["or", "or"]], value => rule.operator = value);
            const well = document.createElement("div");
            well.className = "well location-expressions";
            const title = document.createElement("legend");
            title.textContent = "Expressions";
            well.append(title);
            rule.expressions.forEach((expression, expressionIndex) => {
                const row = document.createElement("div");
                row.className = "location-expression";
                const attribute = document.createElement("input");
                attribute.className = "form-control input-sm";
                attribute.setAttribute("aria-label", "Attribute");
                attribute.value = expression.attribute || "";
                attribute.addEventListener("input", () => expression.attribute = attribute.value);
                const operation = document.createElement("select");
                operation.className = "form-control input-sm";
                operation.setAttribute("aria-label", "Operation");
                const operations = expression.kind === "attr-def" ? [["defined", "is defined"], ["not_defined", "is not defined"]] : [["eq", "="], ["ne", "≠"], ["lt", "<"], ["lte", "≤"], ["gte", "≥"], ["gt", ">"]];
                for (const [value, text] of operations) operation.add(new Option(text, value));
                operation.value = expression.operation;
                operation.addEventListener("change", () => expression.operation = operation.value);
                const attributeGroup = document.createElement("div");
                attributeGroup.className = "location-expression-attribute";
                attributeGroup.append(attribute, operation);
                const valueGroup = document.createElement("div");
                valueGroup.className = "location-expression-value";
                row.append(attributeGroup, valueGroup);
                if (expression.kind === "attr-def") valueGroup.append(operation);
                if (expression.kind !== "attr-def") {
                    const value = document.createElement("input");
                    value.className = "form-control input-sm";
                    value.setAttribute("aria-label", "Value");
                    value.value = expression.value || "";
                    value.addEventListener("input", () => expression.value = value.value);
                    const type = document.createElement("select");
                    type.className = "form-control input-sm";
                    type.setAttribute("aria-label", "Type");
                    for (const [value, text] of [["", ""], ["string", "String"], ["integer", "Integer"], ["version", "Version"]]) type.add(new Option(text, value));
                    type.value = expression.type || "";
                    type.addEventListener("change", () => expression.type = type.value);
                    valueGroup.append(value, type);
                }
                this.button(valueGroup, "Remove Expression", "minus", () => { rule.expressions.splice(expressionIndex, 1); this.render(); });
                well.append(row);
            });
            const select = this.field(well, "", "", [["", ""], ["uname", "#uname"], ["attr-val", "[attribute] = [value]"], ["attr-def", "[attribute] defined"]], () => {});
            select.setAttribute("aria-label", "Expression kind");
            const addition = document.createElement("div");
            addition.className = "location-control location-expression-addition";
            select.parentElement.append(addition);
            select.classList.remove("location-control");
            addition.append(select);
            addition.parentElement.classList.add("location-expression-addition-row");
            this.button(addition, "Add Expression", "plus", () => {
                if (!select.value) return;
                rule.expressions.push({ kind: select.value, operation: select.value === "attr-def" ? "defined" : "eq", attribute: select.value === "uname" ? "#uname" : "", value: "", type: "" });
                this.render();
            });
            fieldset.append(well);
            this.container.append(fieldset);
        });
    }
}
