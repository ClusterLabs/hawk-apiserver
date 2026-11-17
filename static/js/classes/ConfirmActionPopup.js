class ConfirmActionPopup extends Popup {
  #arg;
  #api;
  #errMessage;
  #onConfirm;

  constructor(arg, api, message, errMessage, onConfirm = null) {
    super();

    this.#arg = arg;
    this.#api = api;
    this.#errMessage = errMessage;
    this.#onConfirm = onConfirm;

    // Header
    const header = document.createElement("div");
    header.className = "modal-header";

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
    centerBlock.className = "text-center";

    centerBlock.innerHTML = message;

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
    okBtn.onclick = () => this.#execute();

    footer.append(cancelBtn, okBtn);
    this._appendModalContentChild(footer);
  }

  #execute() {
    if (this.#onConfirm) {
      this.close();
      this.#onConfirm();
      return;
    }

    statusFetch(this.#api, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(this.#arg)
    })
      .then(async res => {
        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || "Unknown error");
        }
        return res.json();
      })
      .then(result => {
        this.#showAlert("success", result.message || "Action completed successfully");
      })
      .catch(err => {
        console.error(this.#errMessage, err);
        this.#showAlert("danger", `${this.#errMessage} ${err.message}`);
      });

    this.close();
  }

  #showAlert(type, message) {
    const alert = document.createElement("x-alert");
    alert.id = "status-alert";
    alert.setAttribute("type", type);
    alert.setAttribute("message", message);
    alert.setAttribute("visible", "true");
    document.getElementById("status-alert")?.replaceWith(alert);
  }
}

class ResourceClearPopup extends ConfirmActionPopup {
  constructor(name) {
    const message = `This will remove any migration constraints for the` +
      ` resource <strong>${name}</strong>. Do you want to continue?`;

    super(name, "/api/cib/resource/clear", message, "Clear failed:");
  }
}

class ResourceMaintenanceOnOffPopup extends ConfirmActionPopup {
  constructor(name, isOn = false) {
    const api = `/api/cib/resource/maintenance-${isOn ? "on" : "off"}`;
    const message = `This will <strong>${isOn ? "enable" : "disable"}</strong> maintenance` +
      ` mode for resource <strong>${name}</strong>. Do you want to continue?`;
    const errMessage = `Resource maintenance mode ${isOn ? "on" : "off"} failed:`;

    super(name, api, message, errMessage);
  }
}

class ResourcePromoteDemotePopup extends ConfirmActionPopup {
  constructor(name, promote = true) {
    const action = promote ? "promote" : "demote";
    const api = promote
      ? "/api/cib/clone/promote"
      : "/api/cib/clone/demote";
    const message = `This will <strong>${action}</strong>` +
      ` the resource <strong>${name}</strong>. Do you want to continue?`;
    const errMessage = `${promote ? "Promote" : "Demote"} failed:`;

    super(name, api, message, errMessage);
  }
}

class ResourceStartStopPopup extends ConfirmActionPopup {
  constructor(name, start = true) {
    const action = start ? "start" : "stop";
    const api = start
      ? "/api/cib/resource/start"
      : "/api/cib/resource/stop";
    const message = `This will <strong>${action}</strong>` +
      ` the resource <strong>${name}</strong>. Do you want to continue?`;
    const errMessage = `${start ? "Start" : "Stop"} failed:`;

    super(name, api, message, errMessage);
  }
}

class NodeMaintenanceOnOffPopup extends ConfirmActionPopup {
  constructor(name, isOn = false) {
    const api = `/api/cib/node/maintenance-${isOn ? "on" : "off"}`;
    const message = isOn
      ? `This will put node <strong>${name}</strong> in maintenance mode. ` +
        `All resources on this node will become unmanaged. Do you want to continue?`
      : `This will bring <strong>${name}</strong> ` +
        `out of maintenance mode. Do you want to continue?`;
    const errMessage = `Node maintenance mode ${isOn ? "on" : "off"} failed:`;

    super(name, api, message, errMessage);
  }
}

class NodeStandbyOnOffPopup extends ConfirmActionPopup {
  constructor(name, isOn = false) {
    const api = `/api/cib/node/standby-${isOn ? "on" : "off"}`;
    const message = isOn
      ? `This will put node <strong>${name}</strong> on standby. ` +
        `All resources will be stopped and/or moved to another node. Do you want to continue?`
      : `This will bring <strong>${name}</strong> online ` +
        `if it is currently on standby. Do you want to continue?`;
    const errMessage = `Node standby ${isOn ? "on" : "off"} failed:`;

    super(name, api, message, errMessage);
  }
}

class NodeFencePopup extends ConfirmActionPopup {
  constructor(name) {
    const api = `/api/cib/node/fence`;
    const message = window.batchConfig
      ? `Inject an unclean-node event for <strong>${name}</strong> into the simulator? No physical fencing action will run.`
      : `This will attempt to immediately fence node <strong>${name}</strong>. Do you want to continue?`;
    const errMessage = `Node fencing failed:`;
    super(name, api, message, errMessage);
  }
}

class NodeClearstatePopup extends ConfirmActionPopup {
  constructor(name) {
    const api = `/api/cib/node/clearstate`;
    const message = `Clear the state of node <strong>${name}</strong>. The node is afterwards assumed clean and offline. ` +
      `This command can be used to manually confirm that a node has been fenced. Be ` +
      `careful! This can cause data corruption if the node is not cleanly down! Do you ` +
      `want to clear the state?`;
    const errMessage = `Node clearstate failed:`;
    super(name, api, message, errMessage);
  }
}

window.ResourceClearPopup = ResourceClearPopup;
window.ResourceMaintenanceOnOffPopup = ResourceMaintenanceOnOffPopup;
window.ResourcePromoteDemotePopup = ResourcePromoteDemotePopup;
window.ResourceStartStopPopup = ResourceStartStopPopup;
window.NodeMaintenanceOnOffPopup = NodeMaintenanceOnOffPopup;
window.NodeStandbyOnOffPopup = NodeStandbyOnOffPopup;
window.NodeFencePopup = NodeFencePopup;
window.NodeClearstatePopup = NodeClearstatePopup;
