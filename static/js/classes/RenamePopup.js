class RenamePopup extends Popup {
    #resourceID;
    #newID;
    #okBtn;
    #api;

    constructor(resourceID, api = "/api/cib/resource/rename") {
        super();

        this.#resourceID = resourceID;
        this.#api = api;

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

    // to enable both primitives and clones
    #editPageURL(resourceID) {
        const url = new URL(window.location.href);
        const parts = url.pathname.split("/");

        // /cib/live/{primitives|clones}/{resource-id}/edit
        parts[4] = encodeURIComponent(resourceID);
        url.pathname = parts.join("/");
        url.search = "";

        return url;
    }

    #handleRename() {
        const oldID = this.#resourceID;
        const newID = (this.#newID.getFrontendValue() || "").trim();

        if (!newID) { alert("New ID can't be empty."); return; }
        if (newID === oldID) { alert("New ID must be different from the current ID."); return; }

        this.#okBtn.disabled = true;

        fetch(this.#api, {
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
                const url = this.#editPageURL(newID);
                url.searchParams.set("flash", "renamed");
                window.location.assign(url);
            })
            .catch(err => {
                console.error("Rename failed:", err);
                const msg = `Failed to rename ${oldID} -> ${newID}: ${err.message}`;

                const url = this.#editPageURL(oldID);
                url.searchParams.set("flash", "error");
                url.searchParams.set("msg", msg);
                window.location.assign(url);
            });

        this.close();
    }

    close() {
        super.close();
    }
}
