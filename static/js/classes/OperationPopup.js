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
        const legacyDOM = this.#externalOperationKvSection.hasAttribute("light-dom");
        // Create overlay
        this.#divContainer = document.createElement("div");
        this.#divContainer.className = "modal";
        this.#divContainer.style.display = "none";
        if (legacyDOM) this.#divContainer.id = "modal";

        // Modal content
        const modalContent = document.createElement("div");
        modalContent.className = "modal-content";
        let contentRoot = modalContent;
        if (legacyDOM) {
            const dialog = document.createElement("div");
            dialog.style.display = "contents";
            dialog.appendChild(modalContent);
            this.#divContainer.appendChild(dialog);
            contentRoot = document.createElement("form");
            modalContent.appendChild(contentRoot);
        } else {
            this.#divContainer.appendChild(modalContent);
        }

        // Header
        const header = document.createElement("div");
        header.className = "modal-header";
        header.textContent = "Configure operation: ";

        const strong = document.createElement("strong");
        strong.textContent = this.getName();
        header.appendChild(strong);
        contentRoot.appendChild(header);

        // Body
        const modalBody = document.createElement("div");
        modalBody.className = "modal-body";

        this.#internalXKvGroup = document.createElement("x-kvgroup");
        if (legacyDOM) {
            this.#internalXKvGroup.setAttribute("light-dom", "");
            this.#internalXKvGroup.setAttribute("field-prefix", "op");
        }
        this.#internalXKvGroup.setAttribute("name", this.getName());
        // /api/cib/resource/operation/attributes/fetch <=> opDefaults
        this.#internalXKvGroup.setAttribute("options-api",
            "/api/cib/resource/operation/attributes/fetch");

        // TODO: avoid using the window.resourceData global structure.
        const ResourceID = window.resourceData.ResourceID;
        const ResourceAgent = window.resourceData.ResourceAgent;
        this.#internalXKvGroup.setAttribute("resource-id", ResourceID);
        this.#internalXKvGroup.setAttribute("resource-agent", ResourceAgent);

        const additionalArguments = {
            Operation: this.getName()
        };

        this.#internalXKvGroup.setAttribute("api-arguments", JSON.stringify(additionalArguments));

        if (legacyDOM) {
            const fieldset = document.createElement("fieldset");
            fieldset.className = "operation-compat-container";
            let parent = fieldset;
            for (let depth = 0; depth < 4; depth++) {
                const wrapper = document.createElement("div");
                parent.appendChild(wrapper);
                parent = wrapper;
            }
            parent.appendChild(this.#internalXKvGroup);
            modalBody.appendChild(fieldset);
        } else {
            modalBody.appendChild(this.#internalXKvGroup);
        }
        contentRoot.appendChild(modalBody);

        // 1) there is a race between both --> sync them
        // 2) when synced, they should appear AFTER modalContent.appendChild(modalBody);
        //    otherwise --> deadlock
        await this.#internalXKvGroup.applyKvMap(this.#initialKvMap);
        await this.#internalXKvGroup.enforceIntervalTimeout();

        // Footer
        const footer = document.createElement("div");
        footer.className = "modal-footer";

        const cancelBtn = document.createElement("button");
        cancelBtn.type = "button";
        cancelBtn.textContent = "Cancel";
        cancelBtn.onclick = () => this.close();

        const okBtn = document.createElement(legacyDOM ? "input" : "button");
        if (legacyDOM) {
            okBtn.type = "submit";
            okBtn.name = "submit";
            okBtn.value = "OK";
        } else {
            okBtn.textContent = "OK";
        }
        okBtn.className = "text-danger";
        okBtn.style.marginLeft = "10px";
        const apply = event => {
            event.preventDefault();
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
        if (legacyDOM) contentRoot.onsubmit = apply;
        else okBtn.onclick = apply;

        footer.append(cancelBtn, okBtn);
        contentRoot.appendChild(footer);
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
        // If this is considered unsafe, we can style separately.
        label.innerHTML = `<strong>${labelText}</strong>`;

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
