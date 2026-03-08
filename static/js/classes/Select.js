class Select {
  #externalContainer; // all components
  #alignContainer; // select/input + button
  #value; // the source of truth. input and select just display
  #inputEl;
  #selectEl;
  #optionsMap;
  #apiEntryGetOptions;
  #apiEntrySetOptions; // Not used (yet), #TODO: reimplement the schema so that it's being used
  #name;
  #enabled;
  #cibID; // e.g. ID in <nvpair id="dummy1-meta_attributes-allow-migrate" name="allow-migrate" value=""/>
          // not sure it's the right place for it,
          // because thus we are loosing abstraction from cib. (#FIXME later)
  #onChangeEvent = null;

  constructor(externalContainer, alignContainer, name, value, apiEntryGetOptions, apiEntrySetOptions, enabled, shadowed, cibID) {
    this.#externalContainer = externalContainer;
    this.#alignContainer = alignContainer;
    this.#apiEntryGetOptions = apiEntryGetOptions;
    this.#apiEntrySetOptions = apiEntrySetOptions;
    this.#name = name;
    this.#enabled = enabled;
    this.#cibID = cibID;

    const label = document.createElement("label");
    // even if the label is empty, it should exist for alignment (still?)
    label.textContent = name;
    if (!name) {
      label.style.display = "none";   // only hides the EMPTY label
    }

    this.#externalContainer.appendChild(label);

    // readonly mirror input (always rendered)
    this.#inputEl = document.createElement("input");
    this.#inputEl.className = "form-control";
    this.#inputEl.readOnly = true;

    // editable select
    this.#selectEl = document.createElement("select");
    this.#selectEl.className = "form-control";

    if (this.#alignContainer && (this.#alignContainer != this.#externalContainer)) {
      this.#alignContainer.appendChild(this.#inputEl);
      this.#alignContainer.appendChild(this.#selectEl);
      this.#externalContainer.appendChild(this.#alignContainer);
    } else {
      this.#externalContainer.appendChild(this.#inputEl);
      this.#externalContainer.appendChild(this.#selectEl);
    }

    // key: option name or ID, value: Option instance
    this.#optionsMap = new Map();


    if (!this.#enabled) {
      this.#inputEl.style.display = "block";
      this.#selectEl.style.display = "none";
    } else {
      this.#inputEl.style.display = "none";
      this.#selectEl.style.display = "block";
    }

    // uncomment to debug
    /*
    this.#inputEl.style.display  = "block";
    this.#selectEl.style.display = "block";
    this.#selectEl.disabled      = false;
    */

    this.setValue(value);

    if(shadowed) {
      this.#applyShadowStyles();
    }

    this.#selectEl.addEventListener("change", (e) => {
      this.setValue(this.#selectEl.value);   // always keep #value in sync
      this.#onChangeEvent?.(e);                 // optional external callback
    });
  }

  disableEdit() {
    this.#inputEl.style.display = "block";
    this.#selectEl.style.display = "none";
    this.#selectEl.disabled      = true;
  }

  enableEdit() {
    this.#inputEl.style.display = "none";
    this.#selectEl.style.display = "block";
    this.#selectEl.disabled      = false;
  }

  #applyShadowStyles() {
    this.#inputEl.style.backgroundColor = "#f5f5f5";
  }

  setOnChangeEvent(onclickHandler) {
    this.#onChangeEvent = onclickHandler;
  }

  // FIXME: Use the init here instead of the KVSection
  init(apiArguments, appendBlank) {
    if (appendBlank) {
      this.appendBlankOption();
    }

    return fetch(this.#apiEntryGetOptions, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(apiArguments)
      })
      .then(async res => {
        if (!res.ok) {
          const text = await res.text(); // read error body
          throw new Error(text || "Unknown error");
        }
        return res.json();
      })
        .then(content => {
          const contentOptions = content.Options || [];
          contentOptions.forEach(o => {
            const optionObj = new SelectOption(o.Name, o.Type, o.DefaultValue, o.Shortdesc, o.Longdesc, o.PossibleValues);
            this.appendOption(optionObj);
          });

          // after options mount, sync/refresh the input and select
          const v = this.getValue();
          this.setValue(v);
        });
  }

  reload(apiArgs = {}) {
    this.#selectEl.innerHTML = "";
    this.#optionsMap.clear();
    return this.init(apiArgs, false);
  }

  appendOption(optionObj) {
    const name = optionObj.getName();

    if (this.#optionsMap.has(name)) {
      console.warn(`Option "${name}" already exists in Select`);
      return;
    }

    this.#selectEl.appendChild(optionObj.getHTML());
    this.#optionsMap.set(name, optionObj);
  }

  appendBlankOption() {
    const blankOptionObj = new SelectOption("", "", "", "", "", []);
    this.appendOption(blankOptionObj);
  }

  getOption(name) {
    return this.#optionsMap.get(name) || null;
  }

  hideOption(name) {
    const option = this.getOption(name);
    if (option) {
      option.hide();
    } else {
      console.log(`Option ${name} not found.`);
    }
  }

  showOption(name) {
    const option = this.getOption(name);
    if (option) {
      option.show();
    } else {
      console.log(`Option ${name} not found.`);
    }
  }

  selectedOption() {
    const domOption = this.#selectEl.selectedOptions[0];
    if (!domOption) return null;

    const optionObj = this.getOption(domOption.value);
    return optionObj;
  }

  sort() {
    const options = Array.from(this.#optionsMap.values());

    options.sort((a, b) => a.getName().localeCompare(b.getName()));

    // don't shuffle, but rather clear and create again
    this.#selectEl.innerHTML = "";
    this.#optionsMap.clear();

    options.forEach(o => this.appendOption(o));
  }

  getHTML() {
    return this.#selectEl;
  }

  // needed by XSelect
  setValue(value) {
    this.#value = value;
    this.#inputEl.value = value;
    if (this.#optionsMap.has(value) || value === "") {
      this.#selectEl.value = value;
    } else {
      console.warn(`[Select:${this.#name}] value "${value}" not found in options`);
    }
  }

  // used by XSelect's value getter
  getValue() {
    return this.#value; // The source of truth
  }

  getName() {
    return this.#name;
  }

  getFrontendValue() {
    return this.getValue();
  }

  getCibID() {
    return this.#cibID;
  }

  size() {
    let count = 0;
    for (const opt of this.#optionsMap.values()) {
      if (opt.countMe())
        count++;
    }
    return count;
  }

  empty() {
    return this.size() === 0;
  }

}

// Expose globally
window.Select = Select;
