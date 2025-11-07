/* it's basic input, useful for instance and meta attributes
 * TODO: implement OperationsInput for the operations (or maybe
 * improve this class). */
class Input {
    #externalContainer;
    #alignContainer;
    #name;
    #defaultValue;
    #required;
    #cibID;
    #cibValue;        // that's what in cib.xml
    #input;

    constructor(externalContainer, alignContainer, name, defaultValue, required, cibID, cibValue, frontendValue, readonly)
    {
        this.#externalContainer = externalContainer;
        this.#alignContainer = alignContainer;
        this.#name = name;
        this.#defaultValue = defaultValue;
        this.#required = required;
        this.#cibID = cibID;
        this.#cibValue = cibValue;

        const label = document.createElement("label");
        label.textContent = name;

        this.#input = document.createElement("input");
        this.#input.type = "text";

        this.#input.value = frontendValue || cibValue || defaultValue || "";
        if (readonly) this.#input.readOnly = true;
        this.#input.className = "form-control";
        // let's give a name to the input to be able to find it from the selenium test
        this.#input.name = "renamePopupInput" + name; // --> "renamePopupInputRename" or "renamePopupInputTo"

        //this.#externalContainer.appendChild(label);
        // the label should come before the #alignContainer
        this.#externalContainer.appendChild(label);
        if(this.#alignContainer && (this.#alignContainer != this.#externalContainer)) {
            // It's safe to include the #alignContainer again.
            // wrt DOM, a browser will simpli move it in the end (and not duplicate)
            this.#externalContainer.appendChild(this.#alignContainer);
            this.#alignContainer.appendChild(this.#input);
        } else {
            this.#externalContainer.appendChild(this.#input);
        }
    }

    getHTML() {
        return this.#input;
    }

    getName() { return this.#name; }
    getDefaultValue() { return this.#defaultValue; }
    getType() { return "text"; }
    getRequired() { return this.#required; }
    getCibID() { return this.#cibID; }
    getCibValue() { return this.#cibValue; }
    getFrontendValue() { return this.#input.value; }
}

// Expose globally
window.Input = Input;
