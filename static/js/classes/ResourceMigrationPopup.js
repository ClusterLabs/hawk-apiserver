class ResourceMigrationPopup extends Popup {
  #resourceID;
  #api;
  #nodes;
  #selectEl;

  constructor(name, nodes) {
    super();

    this.#resourceID = name;
    this.#nodes = nodes;
    this.#api = "/api/cib/resource/migrate";

    // Header
    const header = document.createElement("div");
    //header.className = "modal-header"; // to remove the horizontal line on top

    const closeBtn = document.createElement("button");
    closeBtn.className = "close";
    closeBtn.type = "button";
    closeBtn.innerHTML = `<span aria-hidden="true">&times;</span><span class="sr-only">Close</span>`;
    closeBtn.onclick = () => this.close();
    header.appendChild(closeBtn);

    this._appendModalContentChild(header);

    // Body
    const modalBody = document.createElement("div");
    modalBody.className = "modal-body";

    const centerBlock = document.createElement("div");
    centerBlock.className = "text-center";
    centerBlock.innerHTML = `Migrate <strong>${this.#resourceID}</strong>`;
    modalBody.appendChild(centerBlock);

    // TODO: add more space between centerBlock and this.#selectEl
    this.#selectEl = document.createElement("select");
    this.#selectEl.className = "form-control";
    this.#selectEl.add(new Option("Away from current node", ""));
    nodes.forEach(node => {
      this.#selectEl.add(new Option(node, node));
    });

    modalBody.appendChild(this.#selectEl);

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
    okBtn.onclick = () => this.#migrate();

    footer.append(cancelBtn, okBtn);
    this._appendModalContentChild(footer);
  }

  #migrate() {
    const Destination = this.#selectEl.value;
    statusFetch(this.#api, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ResourceID: this.#resourceID, Destination})
    })
      .then(async res => {
        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || "Unknown error");
        }
        return res.json();
      })
      .catch(err => {
        console.error(`Migration to  ${Destination} failed:`, err);
      });

    this.close();
  }
}

window.ResourceMigrationPopup = ResourceMigrationPopup;
