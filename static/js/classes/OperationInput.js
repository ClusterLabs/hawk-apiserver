/* it's basic input, useful for instance and meta attributes
 * TODO: implement OperationsInput for the operations (or maybe
 * improve this class). */
class OperationInput {
    #divContainer;
    #name;
    #defaultValue;
    #shortdesc;
    #longdesc;
    #type;
    #possibleValues;
    #required;
    #cibID;
    //#kvMap; // we copy the kvMap -> spans, and later take kv from there in getFrontendKValues
    #operationsKvSection;
    #spans;
    #spansPlaceholder;

    constructor(name, defaultValue, shortdesc, longdesc, type
        , possibleValues, required, cibID, kvMap, operationsKvSection)
    {
        this.#name = name;
        this.#defaultValue = defaultValue;
        this.#shortdesc = shortdesc;
        this.#longdesc = longdesc;
        this.#type = type;
        this.#possibleValues = possibleValues;
        this.#required = required;
        this.#cibID = cibID;
        this.#operationsKvSection = operationsKvSection;
        this.#spans = [];

        this.#divContainer = document.createElement("div");
        this.#divContainer.className = "kvsection-row";

        // Blank label (required for css alignment)
        const blankLabel = document.createElement("label");
        blankLabel.textContent = "";
        this.#divContainer.appendChild(blankLabel);

        /* it's the div that looks like an input.
         * and exactly here we store the operation attributes */
        const inputGroup = document.createElement("div");
        inputGroup.className = "form-control d-flex align-items-center";

        // TODO: all these styles shound be in a css,
        // but it's such a pain, just let them be here for a while
        inputGroup.style.height = "auto";        // override Bootstrap 38px
        inputGroup.style.padding = "2px 10px";
        inputGroup.style.fontSize = "14px";
        inputGroup.style.lineHeight = "normal";

        inputGroup.style.width = "60%";
        inputGroup.style.display = "flex";
        inputGroup.style.gap = "6px";
        inputGroup.style.justifyContent = "flex-start";
        // optional, just to make it semantically closer to an input
        inputGroup.setAttribute("role", "textbox");
        inputGroup.setAttribute("aria-disabled", "true");

        // Left-aligned: operation name
        const opName = document.createElement("code");
        opName.textContent = name;
        opName.style.display = "flex";
        opName.style.alignItems = "center";
        opName.style.padding = "0px 6px";
        opName.style.background = "#f8d7da";
        opName.style.borderRadius = "4px";
        inputGroup.appendChild(opName);

        // Right-aligned container
        this.#spansPlaceholder = document.createElement("div");
        this.#spansPlaceholder.style.display = "flex";
        this.#spansPlaceholder.style.alignItems = "center";
        this.#spansPlaceholder.style.gap = "6px";
        this.#spansPlaceholder.style.marginLeft = "auto"; // THIS makes it go right

        this.update(kvMap);

        // Edit button
        const editBtn = document.createElement("button");
        editBtn.title = "Edit operation";
        const icon = document.createElement("i");
        icon.className = "fa fa-pencil-alt";
        editBtn.appendChild(icon);
        editBtn.className = "btn btn-info btn-sm";
        editBtn.onclick = () => {
            // don't use the old kvMap, it might be outdated
            const currentKvMap = this.getFrontendKValues();
            new OperationPopup(this.#operationsKvSection, this.#name, this.#defaultValue
                , this.#shortdesc, this.#longdesc, this.#type, this.#possibleValues
                , this.#required, this.#cibID, currentKvMap, this, true);
          }
        this.#spansPlaceholder.appendChild(editBtn);

        const fieldShortdesc = document.getElementById('field-shortdesc');
        const fieldLongdesc = document.getElementById('field-longdesc');
        this.#divContainer.addEventListener("mouseenter", () => {
            fieldShortdesc.innerHTML = `<code>${name}</code>`;
            fieldLongdesc.innerHTML = longdesc;
            if (defaultValue) {
              fieldLongdesc.innerHTML += `<em> Default: <code>${defaultValue}</code></em>`;
            }
        });

        // Remove button
        const removeBtn = document.createElement("button");
        removeBtn.title = "Remove operation";
        removeBtn.innerHTML = '<i class="fas fa-minus"></i>';
        removeBtn.className = "btn btn-danger btn-sm";
        removeBtn.onclick = () => {
            this.#divContainer.remove();
            operationsKvSection.restoreOption(this);
        }
        /* I placed the remove button inside the input like in Hawk.
         * But I think it doesn't look ok. The inputs of 'parameters'
         * and 'meta attributes' place their remove
         * buttons outside the input, so maybe in future the remove button
         * should be moved outside (#TODO)*/
        this.#spansPlaceholder.appendChild(removeBtn);

        inputGroup.appendChild(this.#spansPlaceholder);

        this.#divContainer.appendChild(inputGroup);
    }

    update(kvMap) {
        /* The o.CibNameValues variable (ref: XOperationsKvGroup.init) is an array,
         * but it should be an object. Don't forget to convert it to the object. */
        if (Array.isArray(kvMap)) {
            alert(
              "Invalid kvMap format:\n\nExpected a key-value object like:\n" +
              '  { "interval": "10s", "timeout": "20s" }\n\n' +
              "But received an array like:\n" +
              '  [ { Name: "interval", Value: "10s" }, { Name: "timeout", Value: "20s" } ]\n\n' +
              "Please convert the array into an object before passing it."
            );
            throw new Error(
              "kvMap should be a plain object (e.g. { interval: \"10s\", timeout: \"20s\" }), not an array like [ { Name: \"interval\", Value: \"10s\" }, ... ]"
            );
          }

        if (typeof kvMap === 'object' && kvMap !== null) {
            // Plain key-value object: { interval: "10s", timeout: "20s" }

            // First, update or add spans
            const handledKeys = new Set();
            Object.entries(kvMap).forEach(([key, value]) => {
                const existingSpan = this.#spans.find(span => span.getKey() === key);

                if (existingSpan) {
                    existingSpan.setValue(value);
                } else {
                    const newSpan = new Span(key, value);
                    this.#spans.push(newSpan);
                    this.#spansPlaceholder?.appendChild(newSpan.getHTML());
                }
                handledKeys.add(key);
            });

            // Now remove spans not in kvMap
            const remainingSpans = [];
            for (const span of this.#spans) {
                const key = span.getKey();
                if (handledKeys.has(key)) {
                    remainingSpans.push(span); // keep
                } else {
                    const html = span.getHTML();
                    if (this.#spansPlaceholder?.contains(html)) {
                        this.#spansPlaceholder.removeChild(html);
                    }
                }
            }
            this.#spans = remainingSpans;
        }
    }

    getHTML() {
        return this.#divContainer;
    }

    getName() { return this.#name; }
    getDefaultValue() { return this.#defaultValue; }
    getShortDesc() { return this.#shortdesc; }
    getLongDesc() { return this.#longdesc; }
    getType() { return this.#type; }
    getPossibleValues() { return this.#possibleValues; }
    getRequired() { return this.#required; }
    getCibID() { return this.#cibID; }
    getCibValue() {
        // this.#cibValue
        throw new Error("this.#cibValue is a fiction, there must be a vector instead.");
    }
    getFrontendKValues() {
        const result = {};
        this.#spans.forEach(span => {
            result[span.getKey()] = span.getValue();
        });
        return result;
    }
}

// Expose globally
window.OperationInput = OperationInput;
