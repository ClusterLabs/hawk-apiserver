class DeletePopup extends Popup {
  #resourceID;

  constructor() {
    super();

    this.#resourceID = window.resourceData.ResourceID;

    // Header
    const header = document.createElement("div");
    header.className = "modal-header";

    // lol, there is a |x|-close button in the header in hawk, let's also add it here
    const closeBtn = document.createElement("button");
    closeBtn.className = "close";
    closeBtn.type = "button";
    closeBtn.innerHTML = `<span aria-hidden="true">&times;</span><span class="sr-only">Close</span>`;
    closeBtn.onclick = () => this.close();
    header.appendChild(closeBtn);

    const icon = document.createElement("div");
    icon.className = "text-center";
    icon.innerHTML = `<i class="fas fa-3x fa-exclamation-triangle text-warning"></i>`;
    header.appendChild(icon);

    this._appendModalContentChild(header);

    // Body
    const modalBody = document.createElement("div");
    modalBody.className = "modal-body";

    const centerBlock = document.createElement("div");
    centerBlock.className = "center-block";
    centerBlock.innerHTML = `Are you sure you want to delete: <strong>${this.#resourceID}</strong> ?`;
    modalBody.appendChild(centerBlock);

    this._appendModalContentChild(modalBody);

    // Footer
    const footer = document.createElement("div");
    footer.className = "modal-footer";

    const cancelBtn = document.createElement("button");
    cancelBtn.className = "btn btn-default cancel";
    cancelBtn.textContent = "Cancel";
    cancelBtn.onclick = () => this.close();

    const okBtn = document.createElement("button");
    okBtn.className = "btn btn-danger commit";
    okBtn.textContent = "OK";
    okBtn.onclick = () => this.#handleDelete();

    footer.append(cancelBtn, okBtn);
    this._appendModalContentChild(footer);
  }

  #handleDelete() {
    fetch('/api/cib/delete-primitive/', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(this.#resourceID)
    })
      .then(async res => {
        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || "Unknown error");
        }
        return res.json();
      })
      .then(() => {
        window.location.href = `/cib/live/resources/types?flash=deleted`;
      })
      .catch(err => {
        console.error("Delete failed:", err);

        const msg =
          `Failed to delete ${this.#resourceID}: ` +
          err.message;

        window.location.href =
          `/cib/live/primitives/${this.#resourceID}/edit` +
          `?flash=error&msg=${encodeURIComponent(msg)}`;
      });

    this.close();
  }

  close() {
    super.close();
  }
}

window.DeletePopup = DeletePopup;
