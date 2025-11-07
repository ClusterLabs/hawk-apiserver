class HelpPopup extends Popup {
  constructor() {
    super();


    // Header
    const header = document.createElement("div");
    header.className = "modal-header";

    const closeBtn = document.createElement("button");
    closeBtn.className = "close";
    closeBtn.type = "button";
    closeBtn.innerHTML = `<span aria-hidden="true">&times;</span><span class="sr-only">Close</span>`;
    closeBtn.onclick = () => this.close();
    header.appendChild(closeBtn);

    const h3 = document.createElement("h3");
    h3.className = "modal-title";
    h3.innerHTML = `<i class="fas fa-question page"></i> Help`;
    header.appendChild(h3);

    this._appendModalContentChild(header);

    // Body
    const modalBody = document.createElement("div");
    modalBody.className = "modal-body help";

    modalBody.innerHTML = `
      <ul class="media-list">
        <li class="media">
          <div class="media-body">
            <h4 class="media-heading">
              <div class="btn-group pull-right">
                <a target="_blank" class="btn btn-danger btn-lg" title="File a Bug Report" href="https://github.com/ClusterLabs/hawk/issues/new">
                  <i class="fas fa-bug fa-fw fa-lg"></i>
                </a>
              </div>
              File a Bug Report
            </h4>
            Report a bug or request a feature!
          </div>
        </li>
        <li class="media">
          <div class="media-body">
            <h4 class="media-heading">
              <div class="btn-group pull-right">
                <a target="_blank" class="btn btn-success btn-lg" title="News" href="http://hawk-ui.github.io/">
                  <i class="fas fa-newspaper fa-fw fa-lg"></i>
                </a>
              </div>
              News
            </h4>
            Visit the Hawk website for information about the latest release.
          </div>
        </li>
        <li class="media">
          <div class="media-body">
            <h4 class="media-heading">
              <div class="btn-group pull-right">
                <a target="_blank" class="btn btn-success btn-lg" title="Online Documentation" href="http://hawk-guide.readthedocs.io/en/latest/">
                  <i class="fas fa-book fa-fw fa-lg"></i>
                </a>
              </div>
              Online Documentation
            </h4>
            Getting Started using Hawk
          </div>
        </li>
      </ul>
    `;

    this._appendModalContentChild(modalBody);

    // Footer
    const footer = document.createElement("div");
    footer.className = "modal-footer";

    const close = document.createElement("button");
    close.className = "btn btn-default";
    close.type = "button";
    close.setAttribute("data-dismiss", "modal");
    close.textContent = "Close";
    close.onclick = () => this.close();

    footer.appendChild(close);
    this._appendModalContentChild(footer);
  }

  close() {
    super.close();
  }
}

window.HelpPopup = HelpPopup;
