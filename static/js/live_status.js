const STATUS_ACTIVE_TAB_KEY = "hawk.status.activeTab";

function getStatusTabLink(tabID) {
    if (!tabID) return null;
    return document.querySelector(`.status-panel .nav-tabs a[href="#${tabID}"][data-toggle="tab"]`);
}

function setActiveStatusTab(tabID) {
    const tabLink = getStatusTabLink(tabID);
    const tabPane = tabID ? document.getElementById(tabID) : null;
    if (!tabLink || !tabPane) return false;

    for (const link of document.querySelectorAll('.status-panel .nav-tabs a[data-toggle="tab"]')) {
        link.parentElement?.classList.toggle("active", link === tabLink);
    }

    for (const pane of document.querySelectorAll(".status-panel .tab-pane")) {
        pane.classList.toggle("active", pane === tabPane);
    }

    return true;
}

function rememberActiveStatusTab(tabID) {
    if (!setActiveStatusTab(tabID)) return;

    sessionStorage.setItem(STATUS_ACTIVE_TAB_KEY, tabID);
    if (window.location.hash !== `#${tabID}`) {
        history.replaceState(null, "", `#${tabID}`);
    }
}

function restoreActiveStatusTab() {
    const hashTab = window.location.hash.slice(1);
    const storedTab = sessionStorage.getItem(STATUS_ACTIVE_TAB_KEY);
    const tabID = getStatusTabLink(hashTab) ? hashTab : storedTab;

    if (tabID) setActiveStatusTab(tabID);

    for (const link of document.querySelectorAll('.status-panel .nav-tabs a[data-toggle="tab"]')) {
        link.addEventListener("click", () => rememberActiveStatusTab(link.getAttribute("href").slice(1)));
    }
}

let resourceDetailRowID = 0;
const resourceDetailRows = new Map();

function getExpandedResourceIDs(container) {
    const expandedIDs = new Set();
    for (const row of container.querySelectorAll("tr.resource-row")) {
        if (row.nextElementSibling?.classList.contains("detail-view")) {
            expandedIDs.add(row.dataset.resourceId);
        }
    }
    return expandedIDs;
}

function rowFromHTML(html) {
    const tbody = document.createElement("tbody");
    tbody.innerHTML = html.trim();
    return tbody.firstElementChild;
}

function syncRowAttributes(existingRow, freshRow) {
    for (const attr of Array.from(existingRow.attributes)) {
        if (!freshRow.hasAttribute(attr.name)) existingRow.removeAttribute(attr.name);
    }

    for (const attr of Array.from(freshRow.attributes)) {
        existingRow.setAttribute(attr.name, attr.value);
    }
}

function patchRowCells(existingRow, freshRow) {
    syncRowAttributes(existingRow, freshRow);

    const freshCells = Array.from(freshRow.children);
    for (const [index, freshCell] of freshCells.entries()) {
        const existingCell = existingRow.children[index];
        if (!existingCell) {
            existingRow.appendChild(freshCell);
            continue;
        }

        if (existingCell.querySelector(".btn-group.open")) continue;
        existingCell.replaceWith(freshCell);
    }

    while (existingRow.children.length > freshCells.length) {
        existingRow.lastElementChild.remove();
    }
}

function showTableRowsOrFallback(tbody, rows, fallbackID, colspan) {
    if (rows.length > 0) {
        tbody.replaceChildren(...rows);
        return;
    }

    let fallback = document.getElementById(fallbackID);
    if (!fallback) {
        fallback = document.createElement("td");
        fallback.id = fallbackID;
        fallback.style.textAlign = "center";
    }
    fallback.colSpan = colspan;
    fallback.textContent = "No matching records found";

    let fallbackRow = fallback.closest("tr");
    if (!fallbackRow || fallbackRow.parentElement !== tbody) {
        fallbackRow = document.createElement("tr");
        fallbackRow.appendChild(fallback);
    }
    tbody.replaceChildren(fallbackRow);
}

function toggleResourceDetails(toggleLink) {
    const mainRow = toggleLink.closest("tr");
    const detailRow = mainRow?.nextElementSibling;
    const expanded = detailRow?.classList.contains("detail-view");

    if (expanded) {
        detailRow.remove();
    } else {
        const newDetailRow = document.createElement("tr");
        newDetailRow.className = "detail-view";
        newDetailRow.innerHTML = resourceDetailRows.get(mainRow.getAttribute("data-detail-row-id"));
        mainRow.after(newDetailRow);
    }

    toggleLink.setAttribute("aria-expanded", String(!expanded));

    const icon = toggleLink.querySelector("i");
    icon.className = expanded
        ? "glyphicon glyphicon-plus icon-plus"
        : "glyphicon glyphicon-minus icon-minus";
}

function generateResourceBtnGroup(role, targetRole, isClone, name, type, active
    , isMaintenance, isManaged, nodeNames, constraintsExist, cibResource
) {
    const resourceEvents = [];
    for(const event of cibResource.Events ?? []) {
        resourceEvents.push({
            ...event,
            ResourceID: cibResource.ID, // all event fields + ResourceID
            ResourceRole: role,
            ResourceTargetRole: targetRole,
            ResourceMaintenance: isMaintenance,
            ResourceManaged: isManaged
        });
    }

    const dropdownUlMenu = document.createElement("ul");
    dropdownUlMenu.className = "dropdown-menu dropdown-menu-center";
    dropdownUlMenu.innerHTML = `
    <li>
        <a
            href="javascript:void(0)"
            onclick='new ResourceMaintenanceOnOffPopup("${name}", ${!isMaintenance})'
            id="maintenance_on"
            class='${isMaintenance ? "hidden" : "visible"}'
        >
            <i class='fa fa-fw fa-wrench'></i> Maintenance
        </a>
    </li>
    <li>
        <a
            href="javascript:void(0)"
            onclick='new ResourceMigrationPopup("${name}", ${JSON.stringify(nodeNames)})'
        >
        <i class="fa fa-fw fa-arrows"></i> Migrate
        </a>
    </li>
    <li>
        <a
            href="javascript:void(0)"
            onclick='new ResourceClearPopup("${name}")' class='${constraintsExist ? "visible" : "hidden"}'
        >
        <i class="fa fa-fw fa-chain-broken"></i> Clear
        </a>
    </li>
    <li>
        <a
            href="javascript:void(0)"
            onclick='new ResourceCleanupPopup("${name}", ${JSON.stringify(nodeNames)})'
        >
        <i class="fa fa-fw fa-eraser"></i> Cleanup
        </a>
    </li>
    <li role="separator" class="divider"></li>
    <li>
        <a
            href="javascript:void(0)"
            onclick='new ResourceRecentEventsPopup("${name}", ${JSON.stringify(resourceEvents)})'
        >
            <i class="fa fa-fw fa-history"></i> Recent events
        </a>
    </li>
    <li role="separator" class="divider"></li>
    <li>
        <a
            href="/cib/live/${isClone ? 'clones' : 'primitives' }/${name}/edit" class="edit"
        >
        <i class="fa fa-fw fa-pencil"></i> Edit
        </a>
    </li>`;

    const isPromoted = (targetRole || "").toLowerCase() === CLONE_PROMOTED;

    const btnGroup = document.createElement("div");
    btnGroup.className = "btn-group";
    btnGroup.role = "group";
    btnGroup.innerHTML = `
    <a
        class="top btn btn-default btn-xs"
        onclick="new ResourceStartStopPopup('${name}', ${!active})"
    >
        <i class="fas fa-${active ? 'stop' : 'play' }"></i>
    </a>
    <div class="btn-group" role="group">
        <a
            onclick="new ResourceMaintenanceOnOffPopup('${name}', ${!isMaintenance})"
            class="btn btn-default btn-xs ${isMaintenance ? 'visible' : 'hidden'}"
            id="maintenance_off"
            title="Disable Maintenance Mode"
        >
            <i class="fa fa-toggle-on"></i>
        </a>
        <a
            class="demote btn btn-default btn-xs ${isClone ? 'visible' : 'hidden' }"
            onclick="new ResourcePromoteDemotePopup('${name}', ${!isPromoted})"
            title="${isPromoted ? 'Demote' : 'Promote'}"
        >
            <i class="fa fa-thumbs-${isPromoted ? 'down' : 'up'}"></i>
        </a>
        <button
            class="btn btn-default btn-xs dropdown-toggle"
            type="button"
            data-toggle="dropdown"
            aria-haspopup="true"
            aria-expanded="false"
        >
            <i class="fa fa-caret-down" aria-hidden="true"></i>
        </button>
        ${dropdownUlMenu.outerHTML}
    </div>
    <a
        onclick='new ResourceDetailsPopup("${name}", "${type}", ${JSON.stringify(cibResource)})'
        class="details btn btn-default btn-xs "
        title="Details"
        data-toggle="modal"
        data-target="#modal"
    >
        <i class="fa fa-search"></i>
    </a>`;

    return btnGroup;
}

function getCibResource(resourceID, cibResources){
    for(const cibResource of cibResources) {
        if (cibResource.ID == resourceID)
            return cibResource;
    }
    return [];
}

function getCrmResource(resourceID, crmResources, crmClones, nodeName){
    for(const crmResource of crmResources ?? []) {
        if (crmResource.ID == resourceID)
            return crmResource;
    }
    for(const crmClone of crmClones ?? []) {
        for(const crmResource of crmClone.Resources ?? []) {
            if (crmResource.ID == resourceID &&
                crmResource.Nodes?.some(node => node.Name == nodeName)) {
                return crmResource;
            }
        }
    }
    return null;
}

function generateNodeBtnGroup(nodeID, nodeName, crmResources, crmClones, cibResources, fenceHistory) {
    const nodeEvents = [];
    // select events on nodeName
    for(const cibResource of cibResources ?? []) {
        const crmResource = getCrmResource(cibResource.ID, crmResources, crmClones, nodeName);
        for(const event of cibResource.Events ?? []) {
            if(event.OnNode == nodeName) {
                const resourceRole = crmResource?.Role || cibResource.Status || "Unknown";
                const resourceTargetRole = crmResource?.TargetRole || cibResource.TargetRole || "";
                nodeEvents.push({
                    ...event,
                    ResourceID: cibResource.ID, // all event fields + ResourceID
                    ResourceRole: resourceRole,
                    ResourceTargetRole: resourceTargetRole,
                    ResourceMaintenance: crmResource?.Maintenance ?? false,
                    ResourceManaged: crmResource?.Managed ?? true
                });
            }
        }
    }

    const dropdownUlMenu = document.createElement("ul");
    dropdownUlMenu.className = "dropdown-menu dropdown-menu-center";
    dropdownUlMenu.innerHTML = `
    <li>
        <a
            href="javascript:void(0)"
            onclick='new NodeFencePopup("${nodeName}")'
        >
            <i class='fa fa-fw fa-plug'></i> Fence
        </a>
    </li>
    <li>
        <a
            href="javascript:void(0)"
            onclick='new NodeClearstatePopup("${nodeName}")'
        >
            <i class="fa fa-fw fa-eraser"></i> Clear state
        </a>
    </li>
    <li role="separator" class="divider"></li>
    <li>
        <a
            href="/cib/live/nodes/${nodeID}/edit"
            class="edit"
        >
            <i class="fa fa-fw fa-pencil"></i> Edit
        </a>
    </li>`;

    const btnGroup = document.createElement("div");
    btnGroup.className = "btn-group";
    btnGroup.role = "group";
    btnGroup.innerHTML = `
    <div class="btn-group" role="group">
    <button
        class="btn btn-default btn-xs dropdown-toggle"
        type="button"
        data-toggle="dropdown"
        data-container="body"
        aria-haspopup="true"
        aria-expanded="false"
    >
        <i class="fa fa-caret-down" aria-hidden="true"></i>
    </button>
    ${dropdownUlMenu.outerHTML}
    </div>
    <a
        href="javascript:void(0)"
        onclick='new NodeRecentEventsPopup("${nodeName}", ${JSON.stringify(nodeEvents)})'
        class="events btn btn-default btn-xs"
        title="Recent events"
        data-toggle="modal"
        data-target="#modal-lg"
    >
        <i class="fa fa-history"></i>
    </a>`;

    // DOM is used to create detailsButton because fenceHistory might containt special symbols
    // and can't be simply inserted like onclick='nodeEvents("${nodeID}", "${nodeName}", "${fenceHistory}")'
    const detailsButton = document.createElement("a");
    detailsButton.href = "javascript:void(0)";
    detailsButton.className = "details btn btn-default btn-xs";
    detailsButton.title = "Details";
    detailsButton.dataset.toggle = "modal";
    detailsButton.dataset.target = "#modal";
    detailsButton.innerHTML = '<i class="fa fa-search"></i>';
    detailsButton.addEventListener("click", () => new NodeDetailsPopup(nodeID, nodeName, fenceHistory));
    btnGroup.appendChild(detailsButton);

    return btnGroup;
}

function generateResourceRow(role, targetRole, name, location, type, active, maintenance, managed, nodeNames, cibResource, isClone, cloneResourceID, expanded) {

    const circleIcon = document.createElement("i");
    circleIcon.title = role;
    const isMaintenance = maintenance === true || maintenance === "true";
    const isUnmanaged = managed === false || managed === "false";
    if (isMaintenance) {
        circleIcon.className = "fa fa-wrench fa-lg text-info";
    } else if(isUnmanaged) {
        circleIcon.className = "fa fa-exclamation-triangle fa-lg text-warning";
    } else if (role?.toLowerCase().includes("start")) {
        circleIcon.className = "fa fa-circle fa-lg text-success text-online";
    } else if (role?.toLowerCase().includes("master")) {
        circleIcon.className = "fa fa-circle fa-lg text-info";
    } else if (role?.toLowerCase().includes("slave")) {
        circleIcon.className = "fa-regular fa-circle-dot fa-lg text-success text-online";
    } else if (role?.toLowerCase().includes("stop")) {
        circleIcon.className = "fa fa-minus-circle fa-lg text-danger";
    } else {
        circleIcon.className = "fa fa-question fa-lg text-warning";
    }

    const constraints = cibResource?.Constraints ?? {};
    const constraintsExist = Boolean(
        (constraints.Colocations ?? []).length ||
        (constraints.Locations ?? []).length ||
        (constraints.Orders ?? []).length
    );

    const statusField = document.createElement("td");
    statusField.className = "col-sm-1";
    statusField.style.textAlign = "center";
    statusField.appendChild(circleIcon);

    // it's different from RoR-hawk. There we count only cli-ban- and cli-prefer- constraints
    for (const c of constraints.Locations ?? []) {
        const linkIcon = document.createElement("i");
        statusField.appendChild(linkIcon);
        linkIcon.className = "fa fa-link fa-status-small text-info";
        linkIcon.title = c.ID;
    }

    const btnGroup = generateResourceBtnGroup(role, targetRole, isClone, name, type, active
        , isMaintenance, !isUnmanaged, nodeNames, constraintsExist, cibResource);
    const detailRowID = String(++resourceDetailRowID);
    let detailRowContent;
    if (isClone) {
        const btnGroupSub = generateResourceBtnGroup(role
            , targetRole
            , false
            , cloneResourceID
            , type
            , active
            , isMaintenance
            , !isUnmanaged
            , nodeNames
            , constraintsExist
            , cibResource);
        detailRowContent = `
        <td colspan="6">
            <table class="table table-bordered table-striped table-hover resource-detail-table">
                <colgroup>
                    <col style="width: 32px;">
                    <col style="width: 8%;">
                    <col style="width: 31%;">
                    <col style="width: 25%;">
                    <col style="width: 17%;">
                    <col style="width: 19%;">
                </colgroup>
                <tbody>
                    <tr>
                        <td class="detail" style="text-align: center;"><i class="glyphicon glyphicon-arrow-right"></i></td>
                        <td style="text-align: center;">${circleIcon.outerHTML}</td>
                        <td style="text-align: center;">${cloneResourceID}</td>
                        <td style="text-align: center;">${location}</td>
                        <td style="text-align: center;"><a onclick='new RAInfoPopup("${cloneResourceID}", "${type}")'
                                                        data-toggle="modal" data-target="#modal-lg">${type}</a></td>
                        <td style="text-align: center;">${btnGroupSub.outerHTML}</td>
                    </tr>
                </tbody>
            </table>
        </td>`;
    } else {
        detailRowContent = `
        <td colspan="6" class="text-center">
            <div class="form-control-static text-muted">No child resources</div>
        </td>`;
    }
    resourceDetailRows.set(detailRowID, detailRowContent);
    const detailIconClass = expanded
        ? "glyphicon glyphicon-minus icon-minus"
        : "glyphicon glyphicon-plus icon-plus";

    const topRow = `
    <tr class="resource-row" data-resource-id="${name}" data-detail-row-id="${detailRowID}">
        <td class="detail">
            <a
                class="detail-icon"
                href="javascript:void(0)"
                aria-expanded="${expanded}"
                onclick="toggleResourceDetails(this)"
            >
                <i class="${detailIconClass}"></i>
            </a>
        </td>
        ${statusField.outerHTML}
        <td style="text-align: center;">${name}</td>
        <td style="text-align: center;">${location}</td>
        <td style="text-align: center;">
            <a
                onclick='new RAInfoPopup("${isClone ? cloneResourceID : name}", "${type}")'
                data-toggle="modal"
                data-target="#modal-lg"
            >
                ${type}
            </a>
            ${isClone ? '(Clone)' : ''}</td>
        <td style="text-align: center;">${btnGroup.outerHTML}</td>
    </tr>
    `;

    return {
        resourceID: name,
        topRow: rowFromHTML(topRow),
        detailRow: expanded ? rowFromHTML(`<tr class="detail-view">${detailRowContent}</tr>`) : null,
    };
}

function populateResourcesTab(crmResources, clones, nodeNames, cibResources){
    let resCount = 0
    const body = document.getElementById("resources-tbody");
    const expandedResourceIDs = getExpandedResourceIDs(body);
    const existingRows = new Map(
        Array.from(body.querySelectorAll("tr.resource-row"))
            .map(row => [row.dataset.resourceId, row])
    );
    const orderedRows = [];
    resourceDetailRows.clear();
    resourceDetailRowID = 0;

    const resourceRows = [];
    for (const resource of crmResources ?? []) {
        resourceRows.push({
            name: resource.ID,
            resource,
            isClone: false,
            cloneResources: []
        });
    }
    for (const clone of clones ?? []) {
        resourceRows.push({
            name: clone.ID,
            resource: clone.Resources[0],
            isClone: (clone.Resources ?? []).length > 0,
            cloneResources: clone.Resources
        });
    }

    resourceRows.sort((a, b) => a.name.localeCompare(b.name));
    for (const { name, resource, isClone, cloneResources } of resourceRows) {
        if (!resource) continue;
        const { ID, ResourceAgent, Role, Active, Maintenance, Managed, Nodes } = resource;

        let location = [];
        let resourceTargetRole = "";
        if (isClone) {
            for (const cloneResource of cloneResources) {
                let bold = false;
                if (resourceTargetRole.toLowerCase() != CLONE_PROMOTED) {
                    if (cloneResource.TargetRole == "") {
                        resourceTargetRole = cloneResource.Role;
                    } else {
                        resourceTargetRole = cloneResource.TargetRole;
                    }
                    if (resourceTargetRole.toLowerCase() == CLONE_PROMOTED)
                        bold = true;
                }
                for (const node of cloneResource.Nodes ?? []) {
                    if (bold) {
                        location.push(`<b>${node.Name}</b>`);
                    } else {
                        location.push(node.Name);
                    }
                }
            }
        } else {
            for (const node of Nodes ?? []) {
                location.push(node.Name);
            }
        }

        const cibResource = getCibResource(ID, cibResources);

        const rendered = generateResourceRow(Role, resourceTargetRole, name, location.join(", "), ResourceAgent,
            Active, Maintenance, Managed, nodeNames, cibResource, isClone, cloneResources[0]?.ID || "",
            expandedResourceIDs.has(name));
        const existingRow = existingRows.get(rendered.resourceID);
        const existingDetailRow = existingRow?.nextElementSibling?.classList.contains("detail-view")
            ? existingRow.nextElementSibling
            : null;

        if (existingRow) {
            patchRowCells(existingRow, rendered.topRow);
            orderedRows.push(existingRow);
        } else {
            orderedRows.push(rendered.topRow);
        }

        if (rendered.detailRow) {
            if (existingDetailRow) {
                patchRowCells(existingDetailRow, rendered.detailRow);
                orderedRows.push(existingDetailRow);
            } else {
                orderedRows.push(rendered.detailRow);
            }
        }

        resCount++;
    }

    showTableRowsOrFallback(body, orderedRows, "resources-tbody-fallback", 6);
    document.getElementById("resources-count").innerText = resCount;
}

function populateNodesTab(nodes, crmResources, crmClones, cibResources){
    const tbody = document.getElementById("nodes-tbody");
    const existingRows = new Map(
        Array.from(tbody.querySelectorAll("tr[data-node-id]"))
            .map(row => [row.dataset.nodeId, row])
    );
    const orderedRows = [];
    let nodesCount = 0

    for (const node of nodes ?? []) {
        const tr = document.createElement("tr");
        tr.dataset.nodeId = node.ID;

        const tdStatus = document.createElement("td");
        tr.appendChild(tdStatus);
        tdStatus.className = "col-sm-1";
        tdStatus.style.textAlign = "center";

        const nodeState = node.Unclean ? "unclean" : (!node.Online || node.Standby ? "offline" : "online");
        const statusIcon = document.createElement("i");
        const statusClasses = {
            online: node.IsDC ? "fa fa-lg fa-home text-success text-online" : "fa fa-lg fa-circle text-success text-online",
            offline: "fa fa-lg fa-minus-circle text-danger",
            unclean: "fa fa-lg fa-plug text-danger",
        };
        statusIcon.className = statusClasses[nodeState];
        statusIcon.title = nodeState + (nodeState === "online" && node.IsDC ? " (DC)" : "");
        tdStatus.appendChild(statusIcon);

        if (node.fenceHistory) {
            const fenceHistoryIcon = document.createElement("i");
            fenceHistoryIcon.className = "fa fa-refresh fa-status-small text-warning";
            fenceHistoryIcon.title = node.fenceHistory;
            tdStatus.append(" ", fenceHistoryIcon);
        }

        if (node.Maintenance) tr.classList.add("info");
        else if (nodeState === "online") tr.classList.add("success");
        else if (nodeState === "unclean") tr.classList.add("danger");

        const tdName = document.createElement("td");
        tr.appendChild(tdName);
        tdName.style.textAlign = "center";
        tdName.innerText = node.Name;

        const tdMaintenance = document.createElement("td");
        tr.appendChild(tdMaintenance);
        tdMaintenance.className = "col-sm-1";
        tdMaintenance.style.textAlign = "center";
        tdMaintenance.innerHTML = `
        <a
            onclick='new NodeMaintenanceOnOffPopup("${node.Name}", ${!node.Maintenance})'
            class="maintenance btn btn-default btn-xs" title="Switch to ${node.Maintenance ? 'ready' : 'maintenance'}"
        >
            <i class="fa fa-toggle-${node.Maintenance ? 'on' : 'off'} text-${node.Maintenance ? 'danger' : 'success'}"></i>
        </a>`;

        const tdStandby = document.createElement("td");
        tr.appendChild(tdStandby);
        tdStandby.className = "col-sm-1";
        tdStandby.style.textAlign = "center";
        tdStandby.innerHTML = `
        <a
            onclick='new NodeStandbyOnOffPopup("${node.Name}", ${!node.Standby})'
            class="standby btn btn-default btn-xs" title="Switch to ${node.Standby ? 'online' : 'standby'}"
        >
            <i class="fa fa-toggle-${node.Standby ? 'on' : 'off'} text-${node.Standby ? 'danger': 'success'}"></i>
        </a>`;

        const tdOperations = document.createElement("td");
        tr.appendChild(tdOperations);
        tdOperations.className = "col-sm-2";
        tdOperations.style.textAlign = "center";
        const btnGroup = generateNodeBtnGroup(node.ID, node.Name, crmResources, crmClones, cibResources, node.fenceHistory);
        tdOperations.appendChild(btnGroup);

        const existingRow = existingRows.get(String(node.ID));
        if (existingRow) {
            patchRowCells(existingRow, tr);
            orderedRows.push(existingRow);
        } else {
            orderedRows.push(tr);
        }

        nodesCount++;
    }

    showTableRowsOrFallback(tbody, orderedRows, "nodes-tbody-fallback", 5);
    document.getElementById("nodes-count").innerText = nodesCount;
}

async function getFenceHistory(nodeName) {
    const res = await fetch("/api/cib/node/fence/history/fetch", {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(nodeName)
    });
    if (!res.ok) throw new Error(`HTTP error: ${res.status}`);
    return res.json();
}

async function updateClusterStatus() {

    // 1.
    const res1 = await fetch("/api/cib/cluster/details/fetch", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ host: window.location.hostname }),
    });
    if (!res1.ok && res1.status !== 503) throw new Error(`Details error: ${res1.status}`);

    const clusterDetails = await res1.json();

    const clusterName = document.getElementById("cluster-name");
    clusterName.innerText = clusterDetails.clusterName || "Unknown";

    setClusterStatusBar("cib-live-cluster-status-alarm", clusterDetails.summary, clusterDetails.status);
    if(clusterDetails.status == CLUSTER_STATUS_OFFLINE) return; // offline --> exit

    // 2. 'crm status --as-xml'
    // TODO?: /api/crm/status/fetch returns the plain 'crm status --as-xml' (in a structure)
    // and we process the WHOLE xml here in JS, I think it's better to
    // prepare data in the backend.
    const res2 = await fetch("/api/crm/status/fetch", {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
    });
    if (!res2.ok) throw new Error(`HTTP error: ${res2.status}`);
    const { XMLName, Nodes, Resources, Clones } = await res2.json();
    const nodeNames = Nodes.map(node => node.Name); // nodeNames are for ResourceMigrationPopup

    // 3. 'cibadmin -Ql' --> resources
    /* I don't like that we call BOTH `crm status` and `cibadmin -Ql`
     * It makes the module UGLY and COMPLICATED.
     * NEW COMMENT: I think the best option is to make up new schema
     * that is easy to consume in the frontend,
     * and simply merge all known cluster data in the new schema in the backend.
     * TODO?: merge
     *   /api/crm/status/fetch
     * and
     *   /api/cib/resources/fetch
     * into one entry-point.
     *
     * OLD COMMENT:
     * We CAN get all the information ONLY from `cibadmin -Ql`.
     * On the other hand
     *  1) `crm status` gives us all the necessary attribues READY TO USE.
     *  2) crmsh should be the prefered interface between hawk and cluster
     * However, `crm status` doesn't return the constraints and `crm constraints`
     * has no `--as-xml` option, so we also have to call `cibadmin -Ql`
     * So, all the status fields (but constraints) come from `crm status`.
     * Constraints come separatelly from `cibadmin -Ql`.
     */
    const res3 = await fetch("/api/cib/resources/fetch", {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
    });
    if (!res3.ok) throw new Error(`HTTP error: ${res3.status}`);
    const CibResources = await res3.json();

    populateResourcesTab(Resources, Clones, nodeNames, CibResources);

    const fenceHistories = await Promise.all(Nodes.map(node => getFenceHistory(node.Name)));
    Nodes.forEach((node, index) => {
        node.fenceHistory = fenceHistories[index];
    });

    populateNodesTab(Nodes, Resources, Clones, CibResources);
}


restoreActiveStatusTab();
updateClusterStatus();  // call it once to initialize
pollClusterStatus(updateClusterStatus);
