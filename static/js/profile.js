class ProfileRevertPopup extends Popup {
    constructor(onConfirm) {
        super();
        const body = document.createElement("div");
        body.className = "modal-body text-center";
        body.textContent = "Any changes will be lost - do you wish to proceed?";
        const footer = document.createElement("div");
        footer.className = "modal-footer";
        const cancel = document.createElement("button");
        cancel.type = "button";
        cancel.className = "btn btn-default cancel";
        cancel.textContent = "Cancel";
        cancel.onclick = () => this.close();
        const ok = document.createElement("button");
        ok.type = "button";
        ok.className = "btn btn-primary commit";
        ok.textContent = "OK";
        ok.onclick = () => { this.close(); onConfirm(); };
        footer.append(cancel, ok);
        this._appendModalContentChild(body);
        this._appendModalContentChild(footer);
    }
}

const profileForm = document.getElementById("edit_profile");
profileForm.querySelector('[name="revert"]').addEventListener("click", event => {
    event.preventDefault();
    new ProfileRevertPopup(() => { window.location.href = "/cib/live/profile/edit"; });
});
const profileURL = new URL(window.location.href);
if (profileURL.searchParams.has("updated")) {
    profileURL.searchParams.delete("updated");
    history.replaceState({}, "", profileURL);
}
