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

    const os = window.hawkConfig?.os || {};
    const majorVersion = (os.version || "").split(".")[0];
    const isSLES = os.id === "sles" && ["15", "16"].includes(majorVersion);
    const docs = [
      {
        icon: "bug", style: "danger", title: "File a Bug Report",
        desc: "Report a bug or request a feature!",
        url: isSLES
          ? "https://bugzilla.suse.com/enter_bug.cgi?classification=SUSE%20Linux%20Enterprise%20High%20Availability%20Extension"
          : "https://github.com/ClusterLabs/hawk/issues/new"
      },
      {
        icon: "newspaper", style: "success",
        title: isSLES ? "Release Notes" : "News",
        desc: isSLES
          ? "Read the release notes for the SUSE High Availability Extension."
          : "Visit the Hawk website for information about the latest release.",
        url: isSLES
          ? `https://www.suse.com/releasenotes/x86_64/SLE-HA/${majorVersion}/`
          : "http://hawk-ui.github.io/"
      },
      {
        icon: "book", style: "success", title: "Online Documentation",
        desc: isSLES
          ? "Read the most up-to-date documentation at the suse.com website."
          : "Getting Started using Hawk",
        url: isSLES
          ? `https://documentation.suse.com/sle-ha/${majorVersion}/`
          : "http://hawk-guide.readthedocs.io/en/latest/"
      }
    ];

    modalBody.innerHTML = `
      <ul class="media-list">
        ${docs.map(doc => `
          <li class="media">
            <div class="media-body">
              <h4 class="media-heading">
                <div class="btn-group pull-right">
                  <a target="_blank" rel="noopener noreferrer" class="btn btn-${doc.style} btn-lg" title="${doc.title}" href="${doc.url}">
                    <i class="fas fa-${doc.icon} fa-fw fa-lg"></i>
                  </a>
                </div>
                ${doc.title}
              </h4>
              ${doc.desc}
            </div>
          </li>
        `).join("")}
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
