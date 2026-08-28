class AddClusterPopup extends Popup {
    #Add;
    #formErrors;
    #fields = {};

    #onSaved;

    constructor(onSaved) {
        super();
        this.#onSaved = onSaved;

        const form = document.createElement("form");
        form.id = "new_cluster";
        form.className = "form-horizontal";
        form.onsubmit = event => {
            event.preventDefault();
            this.#handleAdd();
        };
        this._appendModalContentChild(form);

        const header = document.createElement("div");
        header.className = "modal-header";

        const closeBtn = document.createElement("button");
        closeBtn.className = "close";
        closeBtn.type = "button";
        closeBtn.innerHTML = `<span aria-hidden="true">&times;</span><span class="sr-only">Close</span>`;
        closeBtn.onclick = () => this.close();
        header.appendChild(closeBtn);

        const title = document.createElement("h3");
        title.className = "modal-title";
        title.innerHTML = `<i class="fas fa-server page"></i> Add Cluster`;
        header.appendChild(title);
        form.appendChild(header);

        // Body
        const modalBody = document.createElement("div");
        modalBody.className = "modal-body";

        const formErrors = document.createElement("div");
        formErrors.className = "form-errors";
        formErrors.setAttribute("role", "alert");
        this.#formErrors = formErrors;
        modalBody.appendChild(formErrors);

        const fields = [
            { name: "name", label: "Cluster name", placeholder: "West Coast (example)" },
            { name: "host", label: "Hostname of node in cluster", placeholder: "node1.west.example.com" },
            { name: "port", label: "Server port to connect to", value: "7630" },
        ];

        fields.forEach(field => {
            const group = document.createElement("div");
            group.className = "form-group";

            const label = document.createElement("label");
            label.className = "col-sm-5 control-label";
            label.htmlFor = `cluster_${field.name}`;
            label.textContent = field.label;

            const inputContainer = document.createElement("div");
            inputContainer.className = "col-sm-7";

            const input = document.createElement("input");
            input.type = "text";
            input.className = "form-control text-field";
            input.id = `cluster_${field.name}`;
            input.name = `cluster[${field.name}]`;
            input.placeholder = field.placeholder || "";
            input.value = field.value || "";
            if (field.name === "name" || field.name === "host") {
                input.setAttribute("aria-required", "true");

            }

            this.#fields[field.name] = { input, group };
            inputContainer.appendChild(input);
            group.append(label, inputContainer);
            modalBody.appendChild(group);
        });
        form.appendChild(modalBody);

        // Footer
        const footer = document.createElement("div");
        footer.className = "modal-footer";

        this.#Add = document.createElement("button");
        this.#Add.className = "btn btn-primary";
        this.#Add.innerText = "Add";
        this.#Add.type = "submit"; // to be able to find by selenium test
        this.#Add.name = "submit";

        footer.appendChild(this.#Add);
        form.appendChild(footer);

        // Focus the first field after the popup is drawn.
        requestAnimationFrame(() => this.#fields.name.input.focus());
    }

    #showErrors(errors) {
        let firstInvalidInput;
        Object.entries(errors).forEach(([name, messages]) => {
            const error = document.createElement("div");
            error.className = "alert alert-danger";
            error.textContent = Array.isArray(messages) ? `${name}: ${messages.join(", ")}` : messages;
            this.#formErrors.appendChild(error);
            const field = this.#fields[name];
            if (field) {
                field.group.classList.add("has-error");
                field.input.setAttribute("aria-invalid", "true");
                firstInvalidInput ||= field.input;
            }
        });
        firstInvalidInput?.focus();
    }

    async #handleAdd() {
        if (this.#Add.disabled) return;
        this.#formErrors.replaceChildren();
        Object.values(this.#fields).forEach(({ input, group }) => {
            group.classList.remove("has-error");
            input.setAttribute("aria-invalid", "false");
        });

        const name = this.#fields.name.input.value.trim();
        const host = this.#fields.host.input.value.trim();
        const port = Number(this.#fields.port.input.value.trim());
        const errors = {};
        if (!name) errors.name = ["can't be blank"];
        if (!host) errors.host = ["can't be blank"];
        if (!Number.isInteger(port) || port < 1 || port > 65535) {
            errors.port = ["must be an integer between 1 and 65535"];
        }
        if (Object.keys(errors).length) {
            this.#showErrors(errors);
            return;
        }

        this.#Add.disabled = true;
        this.#Add.innerText = "Saving...";
        try {
            const response = await fetch("/api/dashboard/clusters", {
                method: "POST",
                headers: { "Content-Type": "application/json", "Accept": "application/json" },
                body: JSON.stringify({ name, host, port }),
            });
            if (response.redirected || response.status === 401 || response.status === 403) {
                throw new Error("Your session may have expired. Please sign in again.");
            }
            const result = await response.json();
            if (response.status === 422) {
                this.#showErrors(result);
                return;
            }
            if (!response.ok) {
                throw new Error(result.error || "Unable to save cluster configuration.");
            }
            this.close();
            this.#onSaved?.(result);
        } catch (error) {
            this.#showErrors({ error: error.message || "Unable to save cluster configuration." });
        } finally {
            this.#Add.disabled = false;
            this.#Add.innerText = "Add";
        }
    }
}
