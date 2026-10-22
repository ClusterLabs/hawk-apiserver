class RemoveClusterPopup extends Popup {
    #resolve;
    result;

    constructor(name) {
        super();
        this.result = new Promise(resolve => { this.#resolve = resolve; });

        const header = document.createElement("div");
        header.className = "modal-header";
        const close = document.createElement("button");
        close.type = "button";
        close.className = "close";
        close.setAttribute("aria-label", "Cancel");
        close.textContent = "×";
        close.onclick = () => this.close();
        const icon = document.createElement("div");
        icon.className = "text-center";
        icon.innerHTML = '<i class="fas fa-3x fa-exclamation-triangle text-warning" aria-hidden="true"></i>';
        header.append(close, icon);
        this._appendModalContentChild(header);

        const body = document.createElement("div");
        body.className = "modal-body text-center";
        body.textContent = `Remove cluster ${name} from dashboard?`;
        this._appendModalContentChild(body);

        const footer = document.createElement("div");
        footer.className = "modal-footer remove-cluster-footer";
        const cancel = document.createElement("button");
        cancel.type = "button";
        cancel.className = "btn btn-default cancel close";
        cancel.textContent = "Cancel";
        cancel.onclick = () => this.close();
        const ok = document.createElement("button");
        ok.type = "button";
        ok.className = "btn btn-danger commit";
        ok.textContent = "OK";
        ok.onclick = () => {
            this.#resolve?.(true);
            this.#resolve = null;
            this.close();
        };
        footer.append(cancel, ok);
        this._appendModalContentChild(footer);
        requestAnimationFrame(() => cancel.focus());
    }

    close() {
        this.#resolve?.(false);
        this.#resolve = null;
        super.close();
    }
}
