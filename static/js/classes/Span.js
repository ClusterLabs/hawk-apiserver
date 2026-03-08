class Span {
  #key;
  #span;
  constructor(key, value) {
      this.#key = key;

      this.#span = document.createElement("span");
      this.setValue(value);
      //this.#span.setAttribute("data-role", key);
      this.#span.className = "badge bg-secondary";
      this.#span.style.padding = "2px 6px";
      this.#span.style.background = Span.colorMap[key] || Span.colorMap["other"];
      this.#span.style.color = "white";
      this.#span.style.borderRadius = "4px";
    }

    getKey() { return this.#key; }
    getValue() { return this.#span.textContent; };
    setValue(value) { this.#span.textContent = value; };

    // Return the HTML element
    getHTML() {
      return this.#span;
    }

    // Static color map
    static colorMap = {
      interval: "#007bff",
      timeout: "#28a745",
      enabled: "#17a2b8",
      onFail: "#ffc107",
      role: "#dc3545",
      depth: "#6f42c1",
      recordPending: "#fd7e14",
      description: "#6610f2",
      intervalOrigin: "#20c997",
      requires: "#e83e8c",
      startDelay: "#fd7e14",
      recordPending: "#20c997",
      other: "#6c757d"
    };
  }

  window.Span = Span;
