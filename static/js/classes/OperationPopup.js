class OperationPopup {
    #name;
    #defaultValue;
    #shortdesc;
    #longdesc;
    #type;
    #possibleValues;
    #required;
    #cibID;
    #initialKvMap;

    #isUpdate;
    #divContainer;
    #internalXKvGroup;
    #externalOperationKvSection;
    #operationInput;

    constructor(externalOperationKvSection, name, defaultValue
            , shortdesc, longdesc, type, possibleValues, required
            , cibID, kvMap, operationInput, isUpdate = false) {

        this.#externalOperationKvSection = externalOperationKvSection;

        this.#name = name;
        this.#defaultValue = defaultValue;
        this.#shortdesc = shortdesc;
        this.#longdesc = longdesc;
        this.#type = type;
        this.#possibleValues = possibleValues;
        this.#required = required;
        this.#cibID = cibID;
        this.#initialKvMap = kvMap;

        this.#operationInput = operationInput;

        this.#isUpdate = isUpdate;

        if(isUpdate && !operationInput) {
            const msg = "If it's an update operation, the operationInput must be defined. (we update it)";
            alert(msg);
            throw new Error(msg);
        }

        this.#createPopup();
        document.body.appendChild(this.#divContainer);
        this.#divContainer.style.display = "flex";
    }

    async #createPopup() {
        // Create overlay
        this.#divContainer = document.createElement("div");
        this.#divContainer.className = "modal";
        this.#divContainer.style.display = "none";

        // Modal content
        const modalContent = document.createElement("div");
        modalContent.className = "modal-content";
        this.#divContainer.appendChild(modalContent);

        // Header
        const header = document.createElement("div");
        header.className = "modal-header";
        header.textContent = "Configure operation: ";

        const strong = document.createElement("strong");
        strong.textContent = this.getName();
        header.appendChild(strong);
        modalContent.appendChild(header);

        // Body
        const modalBody = document.createElement("div");
        modalBody.className = "modal-body";

        this.#internalXKvGroup = document.createElement("x-kvgroup");
        this.#internalXKvGroup.setAttribute("name", this.getName());
        // /api/data-interface/resource-operation/fetch-attributes <=> opDefaults
        this.#internalXKvGroup.setAttribute("options-api", "/api/data-interface/resource-operation/fetch-attributes");

        const ResourceID = window.resourceData.ResourceID;
        const ResourceAgent = window.resourceData.ResourceAgent;

        const apiArgs = {
            ResourceID: ResourceID,
            ResourceAgent: ResourceAgent,
            Operation: this.getName(),
            OperationID: ""
          };

        this.#internalXKvGroup.setAttribute("api-arguments", JSON.stringify(apiArgs));

        modalBody.appendChild(this.#internalXKvGroup);
        modalContent.appendChild(modalBody);

        // 1) there is a race between both --> sync them
        // 2) when synced, they should appear AFTER modalContent.appendChild(modalBody);
        //    otherwise --> deadlock
        await this.#internalXKvGroup.applyKvMap(this.#initialKvMap);
        await this.#internalXKvGroup.enforceIntervalTimeout();

        // Footer
        const footer = document.createElement("div");
        footer.className = "modal-footer";

        const cancelBtn = document.createElement("button");
        cancelBtn.textContent = "Cancel";
        cancelBtn.onclick = () => this.close();

        const okBtn = document.createElement("button");
        okBtn.textContent = "OK";
        okBtn.className = "text-danger";
        okBtn.style.marginLeft = "10px";
        okBtn.onclick = () => {
            const currentKvMap = this.#internalXKvGroup.getFrontendKValues();
            if (this.#isUpdate) {
                this.#externalOperationKvSection.updateInput(this.#operationInput, currentKvMap);
            } else {
                this.#externalOperationKvSection.createInput(this.#name, this.#defaultValue
                    , this.#shortdesc, this.#longdesc, this.#type
                    , this.#possibleValues, this.#required, this.#cibID, currentKvMap);
            }
            this.close();
        }

        footer.append(cancelBtn, okBtn);
        modalContent.appendChild(footer);
    }

    // #TODO?: move applyKvMap to XKvGroup
    async #applyKvMap() {
        for (const [k, v] of Object.entries(this.#initialKvMap || {})) {
            await this.#internalXKvGroup.createOption(k, v);
        }
    }

    createLabeledInput(labelText, inputId, value) {
        const wrapper = document.createElement("div");
        wrapper.style.marginTop = "15px";

        const label = document.createElement("label");
        label.htmlFor = inputId;
        label.innerHTML = `<strong>${labelText}</strong>`; // If this is considered unsafe, we can style separately.

        const input = document.createElement("input");
        input.type = "text";
        input.id = inputId;
        input.className = "form-control";
        input.value = value;

        wrapper.append(label, input);
        return wrapper;
    }

    createOperationOptionRow(optionElement) {
        if (!optionElement) return;
        console.log("Would create row for:", optionElement.textContent);
        // Implement this as needed
    }

    close() {
        this.#divContainer.remove();
        this.#divContainer = null;
    }

    getName() { return this.#name; }
    getDefaultValue() { return this.#defaultValue; }
    getShortDesc() { return this.#shortdesc; }
    getLongDesc() { return this.#longdesc; }
    getType() { return this.#type; }
    getPossibleValues() { return this.#possibleValues; }
    getRequired() { return this.#required; }
    getCibID() { return this.#cibID; }
    getInternalKvSection() {
        return this.#internalXKvGroup;
    }
}

// Expose globally
window.OperationPopup = OperationPopup;
