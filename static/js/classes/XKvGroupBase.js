class XKvGroupBase extends HTMLElement {
  constructor() {
    super();
    this.attachShadow({ mode: "open" });

    // ADD debug adopted stylesheet
    if (window.XCSS?.debugSheet) {
      this.shadowRoot.adoptedStyleSheets = [
        ...this.shadowRoot.adoptedStyleSheets,
        window.XCSS.debugSheet
      ];
    }

    // shadow DOM doesn't export the + and - icons from outside, so we do it here
    this.shadowRoot.appendChild(this.#getFontAwesomeLink());
  }

  #getFontAwesomeLink() {
    const faLink = document.createElement("link");
    faLink.rel = "stylesheet";
    faLink.href = "https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.0/css/all.min.css";
    return faLink;
  }
}

// Expose to global scope
window.XKvGroupBase = XKvGroupBase;
