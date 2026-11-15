class ResourceDetailsPopup extends Popup {
    #resourceID;

    constructor(name, agent, cibResource) {
        super();

        this.#resourceID = name;

        // Header
        const header = document.createElement("div");
        header.className = "modal-header";
        header.innerHTML = `
        <h4>
            <i class='fa fa-search'></i>${name}<div class='pull-right text-muted small text-uppercase'>${cibResource.ObjectType || 'primitive'}</div>
        </h4>`;

        this._appendModalContentChild(header);

        // Body
        const modalBody = document.createElement("div");
        modalBody.className = "modal-body";

        modalBody.innerHTML = `
        <div class="panel">
            <div class="row">
                <div class="col-md-5"><label>${cibResource.ObjectType === 'group' ? 'Type' : 'Agent'}</label></div>
                <div class="col-md-7">${agent}</div>
            </div>
        </div>`;

        const parameters = [];
        for (const parameter of cibResource.InstanceAttributes ?? [])
            parameters.push({ Name: parameter.name, Value: parameter.value });

        const metaAttributes = [];
        for (const metaAttribute of cibResource.MetaAttributes ?? [])
            metaAttributes.push({ Name: metaAttribute.name, Value: metaAttribute.value });

        const parametersTable = this.#makeKVTable("Parameters", parameters);
        const metaAttributesTable = this.#makeKVTable("Meta Attributes", metaAttributes);
        const instancesTable = this.#makeInstancesTable(cibResource);
        const colocationsTable = this.#makeColocationsTable(cibResource.ID, cibResource.Constraints.Colocations);
        const locationsTable = this.#makeLocationsTable(cibResource.Constraints.Locations);
        const ordersTable = this.#makeOrdersTable(cibResource.Constraints.Orders);

        if (parametersTable)
            modalBody.appendChild(parametersTable);
        if (metaAttributesTable)
            modalBody.appendChild(metaAttributesTable);
        if (instancesTable)
            modalBody.appendChild(instancesTable);
        if (colocationsTable)
            modalBody.appendChild(colocationsTable);
        if (locationsTable)
            modalBody.appendChild(locationsTable);
        if (ordersTable)
            modalBody.appendChild(ordersTable);

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

    #makeInstancesTable(cibResource) {
        if (!cibResource || cibResource.Node == "") return null;
        return this.#makeKVTable("Instances", [{ Name: cibResource.Node, Value: "Started" }]);
    }

    #makeKVTable(name, rows) {
        if (!rows || rows.length === 0) return null;

        const kvTable = document.createElement("div");
        kvTable.className = "panel panel-default";

        const heading = document.createElement("div");
        kvTable.appendChild(heading);
        heading.className = "panel-heading"
        heading.role = "tab";
        heading.id = "r-metacontrol";
        heading.innerHTML = `
        <h4 class="panel-title">
            <a href="#r-metas" data-toggle="collapse" role="button" aria-expanded="true" aria-controls="r-metas">
                ${name}<div class="pull-right"><span class="caret"></span></div>
            </a>
        </h4>`;

        const content = document.createElement("div");
        kvTable.appendChild(content);
        content.className = "panel-collapse collapse in";
        content.role = "tabpanel";
        content.id = "r-metas";

        const table = document.createElement("table")
        table.className = "table table-condensed";
        content.appendChild(table);

        const tbody = document.createElement("tbody");
        table.appendChild(tbody);

        for (const { Name, Value } of rows) {
            const tr = document.createElement("tr");
            tbody.appendChild(tr);

            const tdName = document.createElement("td");
            tr.appendChild(tdName);
            tdName.className = "col-sm-5";
            tdName.style.textAlign = "center";
            tdName.innerText = Name;

            const tdValue = document.createElement("td");
            tr.appendChild(tdValue);
            tdValue.className = "col-sm-7";
            tdValue.style.textAlign = "center";
            tdValue.innerHTML = `<code>${Value}</code>`;
        }

        return kvTable;
    }

    #makeColocationsTable(resourceID, colocations) {
        if (!colocations) return null;

        const resultDiv = document.createElement("div");
        resultDiv.className = "panel panel-default";

        const heading = document.createElement("div");
        resultDiv.appendChild(heading);
        heading.className = "panel-heading"
        heading.role = "tab";
        heading.id = "r-colocations";
        heading.innerHTML = `
        <div class="panel-heading" id="r-colocations" role="tab">
            <h4 class="panel-title">
                <a
                    href="#r-colocations-body"
                    data-toggle="collapse"
                    role="button"
                    aria-expanded="true"
                    aria-controls="r-colocations-body"
                >
                    Colocations<div class="pull-right"><span class="caret"></span></div>
                </a>
            </h4>
        </div>`;

        const content = document.createElement("div");
        resultDiv.appendChild(content);
        content.className = "panel-collapse collapse in";
        content.role = "tabpanel";
        content.id = "r-colocations-body";

        const table = document.createElement("table")
        table.className = "table table-condensed";
        content.appendChild(table);

        const tbody = document.createElement("tbody");
        table.appendChild(tbody);

        const trHeading = document.createElement("tr");
        tbody.appendChild(trHeading);
        trHeading.innerHTML = `
        <th style="text-align: center">Role</th>
        <th style="text-align: center">Score</th>
        <th style="text-align: center">With Resource</th>
        <th style="text-align: center">With Resource's Role</th>`;

        for (const { ID, Rsc, RscRole, Score, WithRsc, WithRscRole } of colocations) {
            const tr = document.createElement("tr");
            tbody.appendChild(tr);
            if (WithRsc == resourceID) { // switch Rsc and WithRsc
                tr.innerHTML = `
                <td style="text-align: center">${WithRscRole}</td>
                <td style="text-align: center">${Score}</td>
                <td style="text-align: center">${Rsc}</td>
                <td style="text-align: center">${RscRole}</td>`;
            } else {
                tr.innerHTML = `
                <td style="text-align: center">${RscRole}</td>
                <td style="text-align: center">${Score}</td>
                <td style="text-align: center">${WithRsc}</td>
                <td style="text-align: center">${WithRscRole}</td>`;
            }
        }

        return resultDiv;
    }

    #makeLocationsTable(locations) {
        if (!locations) return null;

        const resultDiv = document.createElement("div");
        resultDiv.className = "panel panel-default";

        const heading = document.createElement("div");
        resultDiv.appendChild(heading);
        heading.className = "panel-heading"
        heading.role = "tab";
        heading.id = "r-locations";
        heading.innerHTML = `
        <div class="panel-heading" id="r-locations" role="tab">
            <h4 class="panel-title">
                <a
                    href="#r-locations-body"
                    data-toggle="collapse"
                    role="button"
                    aria-expanded="true"
                    aria-controls="r-locations-body"
                >
                    Locations<div class="pull-right"><span class="caret"></span></div>
                </a>
            </h4>
        </div>`;

        const content = document.createElement("div");
        resultDiv.appendChild(content);
        content.className = "panel-collapse collapse in";
        content.role = "tabpanel";
        content.id = "r-locations-body";

        const table = document.createElement("table")
        table.className = "table table-condensed";
        content.appendChild(table);

        const tbody = document.createElement("tbody");
        table.appendChild(tbody);

        const trHeading = document.createElement("tr");
        tbody.appendChild(trHeading);
        trHeading.innerHTML = `
        <th style="text-align: center">Node</th>
        <th style="text-align: center">Score</th>`;

        for (const { ID, Node, Rsc, Score } of locations) {
            const tr = document.createElement("tr");
            tbody.appendChild(tr);
            tr.innerHTML = `
            <td style="text-align: center">${Node}</td>
            <td style="text-align: center">${Score}</td>`;
        }

        return resultDiv;
    }

    #makeOrdersTable(orders) {
        if (!orders) return null;

        const resultDiv = document.createElement("div");
        resultDiv.className = "panel panel-default";

        const heading = document.createElement("div");
        resultDiv.appendChild(heading);
        heading.className = "panel-heading"
        heading.role = "tab";
        heading.id = "r-orders";
        heading.innerHTML = `
        <div class="panel-heading" id="r-orders" role="tab">
            <h4 class="panel-title">
                <a
                    href="#r-orders-body"
                    data-toggle="collapse"
                    role="button"
                    aria-expanded="true"
                    aria-controls="r-orders-body"
                >
                    Orders<div class="pull-right"><span class="caret"></span></div>
                </a>
            </h4>
        </div>`;

        const content = document.createElement("div");
        resultDiv.appendChild(content);
        content.className = "panel-collapse collapse in";
        content.role = "tabpanel";
        content.id = "r-orders-body";

        const table = document.createElement("table")
        table.className = "table table-condensed";
        content.appendChild(table);

        const tbody = document.createElement("tbody");
        table.appendChild(tbody);

        const trHeading = document.createElement("tr");
        tbody.appendChild(trHeading);
        trHeading.innerHTML = `
        <th style="text-align: center">Kind</th>
        <th style="text-align: center">First Resource</th>
        <th style="text-align: center">First Action</th>
        <th style="text-align: center">Then Resource</th>
        <th style="text-align: center">Then Action</th>`;

        for (const { First, FirstAction, ID, Kind, Then, ThenAction } of orders) {
            const tr = document.createElement("tr");
            tbody.appendChild(tr);
            tr.innerHTML = `
            <td style="text-align: center">${Kind}</td>
            <td style="text-align: center">${First}</td>
            <td style="text-align: center">${FirstAction}</td>
            <td style="text-align: center">${Then}</td>
            <td style="text-align: center">${ThenAction}</td>`;
        }

        return resultDiv;
    }
}

window.ResourceDetailsPopup = ResourceDetailsPopup;
