class XKvGroupBase extends HTMLElement {
  #lightRoot;
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

  // Legacy document XPath cannot cross a shadow root. Opt in only where needed.
  get contentRoot() {
    if (!this.hasAttribute("light-dom")) return this.shadowRoot;
    if (!this.#lightRoot) {
      this.shadowRoot.appendChild(document.createElement("slot"));
      const style = document.createElement("style");
      const scope = ":is(x-kvgroup[light-dom], x-operations-kvgroup[light-dom])";
      // Light-DOM compatibility must not import Bootstrap's control styling.
      // Reset only the component controls, then apply the same rules as shadow DOM.
      style.textContent = `
        ${scope} :where(.form-control, .btn, .badge, label, code) {
          all: revert;
        }
      ` + Array.from(window.XCSS?.debugSheet?.cssRules || []).map(rule => {
        const selectors = rule.selectorText.split(",").map(selector => {
          const value = selector.trim();
          // Preserve opt-in host styling when rendering the same controls in light DOM.
          if (value.startsWith(":host(")) return value.replace(/^:host\(([^)]+)\)/, `${scope}$1`);
          return `${scope} ${value}`;
        });
        return `${selectors.join(",")} { ${rule.style.cssText} }`;
      }).join("\n");
      // Legacy XPath requires an anchor for Edit; visually it remains a button.
      const buttonRule = Array.from(window.XCSS?.debugSheet?.cssRules || [])
        .find(rule => rule.selectorText === "button.btn");
      if (buttonRule) style.textContent += `
        ${scope} a.btn {
          ${buttonRule.style.cssText}
          color: ButtonText;
          font: 400 13.3333px sans-serif;
          box-sizing: border-box;
          text-decoration: none;
        }
      `;
      this.appendChild(style);
      this.#lightRoot = this;
    }
    return this.#lightRoot;
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
