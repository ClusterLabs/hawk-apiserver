class RenamePopup extends Popup {
  #resourceID;
  #newID;
  #okBtn;

  constructor() {
    super();

    this.#resourceID = window.resourceData.ResourceID;

    // Header (no |x|-close button)
    const header = document.createElement("div");
    header.className = "modal-header";
    const title = document.createElement("h3");
    title.className = "modal-title";
    title.textContent = "Rename Resource";
    header.appendChild(title);
    this._appendModalContentChild(header);

    // Body
    const modalBody = document.createElement("div");
    modalBody.className = "modal-body";

    const externalWrapOldName = document.createElement("div");
    externalWrapOldName.className = "kvsection-row";

    const externalWrapNewName = document.createElement("div");
    externalWrapNewName.className = "kvsection-row";

    const oldID = new Input(externalWrapOldName, null, "Rename", this.#resourceID, true, "", "", this.#resourceID, true);
    this.#newID = new Input(externalWrapNewName, null, "To", this.#resourceID, true, "", "", this.#resourceID, false);

    modalBody.append(externalWrapOldName, externalWrapNewName);
    this._appendModalContentChild(modalBody);

    // Footer
    const footer = document.createElement("div");
    footer.className = "modal-footer";

    const cancelBtn = document.createElement("button");
    cancelBtn.className = "btn btn-default cancel";
    cancelBtn.textContent = "Cancel";
    cancelBtn.onclick = () => this.close();

    this.#okBtn = document.createElement("button");
    this.#okBtn.className = "btn btn-primary";
    this.#okBtn.innerHTML = `<i class="fas fa-save"></i> Rename`;
    this.#okBtn.onclick = () => this.#handleRename();
    this.#okBtn.type = "submit"; // to be able to find by selenium test

    footer.append(cancelBtn, this.#okBtn);
    this._appendModalContentChild(footer);

    // Focus this.#newID after the popup drawn
    requestAnimationFrame(() => this.#newID.getHTML().focus());

    // submit on Enter
    externalWrapNewName.addEventListener("keydown",
      e => {
        if (e.key === "Enter")
          this.#handleRename();
      }
    );
  }

  #handleRename() {
    const oldID = this.#resourceID;
    const newID = (this.#newID.getFrontendValue() || "").trim();

    if (!newID) { alert("New ID can't be empty."); return; }
    if (newID === oldID) { alert("New ID must be different from the current ID."); return; }

    this.#okBtn.disabled = true;

    fetch('/api/cib/rename-primitive/', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ oldID, newID })
    })
      .then(async res => {
        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || "Unknown error");
        }
        return res.json();
      })
      .then(() => {
        window.location.href = `/cib/live/primitives/${newID}/edit?flash=renamed`;
      })
      .catch(err => {
        console.error("Rename failed:", err);
        const msg =
          `Failed to rename ${oldID} -> ${newID}: ` +
          err.message;

        window.location.href =
          `/cib/live/primitives/${oldID}/edit` +
          `?flash=error&msg=${encodeURIComponent(msg)}`;
      });

    this.close();
  }

  close() {
    super.close();
  }
}

window.RenamePopup = RenamePopup;
