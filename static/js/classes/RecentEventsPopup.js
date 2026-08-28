const headerFieldNode = "Node"
const headerFieldResource = "Resource"

class RecentEventsPopup extends Popup {
  #name;
  #headerField;

  constructor(name, events, headerField) {
    super();

    this.#name = name;
    this.#headerField = headerField;

    // Header
    const header = document.createElement("div");
    header.className = "modal-header";
    header.innerHTML = `<h3 class="modal-title"><i class="fa fa-history page"></i> Recent events: ${name}</h3>`;

    const xBtn = document.createElement("button");
    xBtn.className = "close";
    xBtn.type = "button";
    xBtn.innerHTML = `<span aria-hidden="true">&times;</span><span class="sr-only">Close</span>`;
    xBtn.onclick = () => this.close();
    header.appendChild(xBtn);

    this._appendModalContentChild(header);

    // Body
    const modalBody = document.createElement("div");
    modalBody.className = "modal-body";

    const eventsTable = this.#makeTable(events);
    if(eventsTable) modalBody.appendChild(eventsTable);

    this._appendModalContentChild(modalBody);

    // Footer
    const footer = document.createElement("div");
    footer.className = "modal-footer";

    const closeBtn = document.createElement("button");
    closeBtn.className = "btn btn-default cancel";
    closeBtn.textContent = "Close";
    closeBtn.onclick = () => this.close();

    footer.appendChild(closeBtn);
    this._appendModalContentChild(footer);
  }

  #makeTable(events) {
    if(!events) return null;

    const resultDiv = document.createElement("div");
    resultDiv.className = "panel panel-default";

    const content = document.createElement("div");
    resultDiv.appendChild(content);
    content.className = "panel-collapse collapse in";
    content.role = "tabpanel";
    content.id = "r-events-body";

    const table = document.createElement("table")
    table.className = "table table-condensed";
    content.appendChild(table);

    const tbody = document.createElement("tbody");
    table.appendChild(tbody);

    const trHeading = document.createElement("tr");
    tbody.appendChild(trHeading);
    trHeading.innerHTML = `
      <th style="text-align: center">RC</th>
      <th style="text-align: center">${this.#headerField}</th>
      <th style="text-align: center">Operation Key</th>
      <th style="text-align: center">Last Change</th>
      <th style="text-align: center">State</th>
      <th style="text-align: center">Call ID</th>
      <th style="text-align: center">Exec</th>
      <th style="text-align: center">Complete</th>
    `;

    events.sort((a, b) => b.LastRcChange.localeCompare(a.LastRcChange)); // descending order, like in RoR
    for (const { ID, CallID, ExecTime, LastRcChange, OnNode, ResourceID
          , ResourceRole, ResourceTargetRole, ResourceMaintenance, ResourceManaged
          , Operation, OperationKey, OpStatus, RCCode  } of events) {

      const resourceRole = ResourceRole || "Unknown";
      const resourceTargetRole = ResourceTargetRole || "";
      const description = [];
      if (resourceRole.toLowerCase() == "stopped" && resourceTargetRole.toLowerCase() == "stopped")
        description.push("disabled");
      if (ResourceMaintenance == false && ResourceManaged) {
        // pass
      }
      else if (ResourceMaintenance && ResourceManaged == false) {
        description.push("maintenance");
      }
      else if (ResourceMaintenance == false && ResourceManaged == false) {
        description.push("unmanaged");
      }
      else if (ResourceMaintenance && ResourceManaged) { // shoudn't happen
        description.push("maintenance");
      }

      const fullStatus = description.length
        ? `${resourceRole} (${description.join(", ")})`
        : resourceRole;

      const tr = document.createElement("tr");
      tbody.appendChild(tr);
      const date = new Date(LastRcChange * 1000);
      const formattedDate = date.toLocaleString("en-US", {
          weekday: "short",
          month: "short",
          day: "2-digit",
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
          year: "numeric",
          hour12: false
      }).replace(",", "");

      let complete = OpStatus;
      if (OpStatus == "0")
        complete = '<i class="fa fa-check"></i>';
      else
        complete = '<i class="fa fa-refresh text-muted"></i>';

      tr.innerHTML = `
        <td style="text-align: center">${RCCode}</td>
        <td style="text-align: center">${this.#headerField == headerFieldNode ? OnNode : ResourceID}</td>
        <td style="text-align: center">${OperationKey}</td>
        <td style="text-align: center">${formattedDate}</td>
        <td style="text-align: center">${fullStatus.toLowerCase()}</td>
        <td style="text-align: center">${CallID}</td>
        <td style="text-align: center">${ExecTime}ms</td>
        <td style="text-align: center">${complete}</td>
        `;

      if (RCCode != "0") {
        tr.style.backgroundColor = "#f2dede";
      }
    }

    return resultDiv;
  }
}

class ResourceRecentEventsPopup extends RecentEventsPopup {
  constructor(name, events) {
    super(name, events, headerFieldNode);
  }
}

class NodeRecentEventsPopup extends RecentEventsPopup {
  constructor(name, events) {
    super(name, events, headerFieldResource);
  }
}
