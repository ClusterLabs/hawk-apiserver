class XSelect extends HTMLElement {
  #divContainer;
  #select;
  #apiEntryGetOptions;

  constructor() {
    super();
    this.attachShadow({ mode: "open" });
  }

  connectedCallback() {
    // read attributes locally (no persistent fields)
    const label = this.getAttribute("label") || "";
    const description = this.getAttribute("description") || "";
    const valueAttr = this.getAttribute("value") || "";
    const hideWhenEmpty = this.getAttribute("hide-when-empty") === "true";

    if (hideWhenEmpty && !valueAttr) {
      this.visible = false;
    }

    this.#apiEntryGetOptions = this.getAttribute("options-api") || "";
    const disabled = this.getAttribute("disabled") === "true";
    const apiArgsAttr = this.getAttribute("api-args");
    let apiArgs = {};
    if (apiArgsAttr) {
      try { apiArgs = JSON.parse(apiArgsAttr); }
      catch { console.error("Invalid JSON in api-args:", apiArgsAttr); }
    }

    // shadow styles so .form-control works inside shadow
    const style = document.createElement("style");
    // MAGIC ALARM (#FIXME): width:60%
    // it aligns horizontally the
    //        class-xselect
    //        provider-xselect
    //        type-xselect
    // with
    //        kvgroup-instance_attributes
    //        kvgroup-operations
    //        kvgroup-meta_attributes
    style.textContent = `
      .kvsection-row { display:flex; align-items:center; gap:26px; margin-bottom:8px; }
      .form-control { width:60%; padding:6px 10px; border:1px solid #ccc; border-radius:4px; font-size:14px; box-sizing:border-box; }
      .kvsection-row > label { margin-left:auto; flex:0 0 80px; text-align:right; }
      .form-control:focus { border-color: #66afe9; outline: 0; box-shadow: 0 0 8px rgba(102, 175, 233, 0.6); }
    `;
    this.shadowRoot.appendChild(style);

    // layout (simple row container)
    this.#divContainer = document.createElement("div");
    this.#divContainer.className = "kvsection-row";
    this.shadowRoot.appendChild(this.#divContainer);

    // SELECT engine renders: label -> readonly input -> select
    this.#select = new Select(this.#divContainer, null, label, valueAttr, this.#apiEntryGetOptions, "", !disabled, true, "");

    // bubble change events to <x-select> (across shadow boundary)
    // Without composed: true, <x-select> would fire the change event inside its shadow root,
    // but the outer page (<body>, window, etc.) would never know anything happened.
    const emitChange = () => this.dispatchEvent(new Event("change", { bubbles: true, composed: true }));
    this.#select.setOnChangeEvent(emitChange);

    this.#divContainer.addEventListener("mouseenter", () => {
      const fieldShortdesc = document.getElementById('field-shortdesc');
      const fieldLongdesc = document.getElementById('field-longdesc');
      if (!fieldShortdesc || !fieldLongdesc) return;        // not there yet
      fieldShortdesc.innerHTML = label;
      fieldLongdesc.innerHTML = description;
    });

    // initialize and set initial value if provided
    /* We set the value already in the constructor (it's easier),
       so now there is no need in extra this.#select.setValue(valueAttr);
    */
    this.#select.init(apiArgs, false)
      .then(() => {
        if (valueAttr && this.#select.getOption(valueAttr)) {
          this.#select.setValue(valueAttr);
        }

        // trigger the onchange event only after init
        // i.e. set providers after loading the classes
        // and i.e. set agent types after loading the provider
        emitChange();
      })
      .catch(err => console.error("x-select init error:", err));

  }

  setOnChangeEvent(cb)     { this.setOnChangeCallback(cb); } // legacy alias
  init(args, appendBlank)  { return this.#select.init(args, !!appendBlank); }
  reload(args)             {
    return this.#select.reload(args);
  }

  appendOption(opt)        { this.#select.appendOption(opt); }
  appendBlankOption(text)  { this.#select.appendBlankOption(text); }
  getOption(name)          { return this.#select.getOption(name); }
  hideOption(name)         { this.#select.hideOption(name); }
  showOption(name)         { this.#select.showOption(name); }
  selectedOption()         { return this.#select.selectedOption(); }
  sort()                   { this.#select.sort(); }
  getHTML()                { return this.#select.getHTML(); }
  setValue(name)           { this.#select.setValue(name); }
  get value()              { return this.#select.getValue?.() || this.#select.selectedOption()?.getName(); }

  set visible(v) {
    this.style.display = v ? "" : "none";
  }

  get visible() {
    return this.style.display !== "none";
  }

  // pass-throughs for compatibility and OOP friendliness
  setOnChangeCallback(cb) {
    const emitChange = () => this.dispatchEvent(new Event("change", { bubbles: true, composed: true }));
    this.#select.setOnChangeEvent((e) => { cb?.(e); emitChange(); });
  }

  disconnectedCallback() {
    const selectEl = this.#select?.getHTML();
    if (selectEl) selectEl.onchange = null;
  }

  disableEdit() {
    this.#select.disableEdit();
  }

  enableEdit() {
    this.#select.enableEdit();
  }
}

customElements.define("x-select", XSelect);
