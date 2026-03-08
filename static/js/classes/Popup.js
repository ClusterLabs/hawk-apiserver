class Popup {
  #escListener = (e) => { if (e.key === "Escape") this.close(); };
  #divContainer = null;
  #modalContent = null;

  constructor() {
    this.#divContainer = document.createElement("div");
    document.body.appendChild(this.#divContainer);

    this.#divContainer.className = "modal";
    this.#divContainer.style.display = "flex";

    this.#modalContent = document.createElement("div");
    this.#modalContent.className = "modal-content";
    this.#divContainer.appendChild(this.#modalContent);

    // close when clicking outside the modal
    document.addEventListener("keydown", this.#escListener);
    this.#divContainer.addEventListener("click", (e) => {
      if (e.target === this.#divContainer) this.close();
    });

    // some modern ARIA features
    this.#divContainer.setAttribute("role", "dialog");
    this.#divContainer.setAttribute("aria-modal", "true");
    //this.#divContainer.setAttribute("aria-label", label);
  }

  // @protected
  _appendModalContentChild(child) {
    this.#modalContent.appendChild(child);
  }

  close() {
    if (!this.#divContainer) return;
    document.removeEventListener("keydown", this.#escListener);
    this.#divContainer?.remove();
    this.#divContainer = null;
  }
}
