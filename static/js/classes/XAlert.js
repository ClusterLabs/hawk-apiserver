class XAlert extends HTMLElement {
  constructor() {
    super();
    this.attachShadow({ mode: "open" });
  }

  connectedCallback() {
    const type = this.getAttribute("type") || "info";
    const message = this.getAttribute("message") || "";
    const visible = this.getAttribute("visible") !== "false";

    /* In the end we will get
    <x-alert type="success" message="Primitive created successfully" visible="true">
      #shadow-root
      <style>...</style>
      <div role="alert" class="alert alert-dismissible alert-success">
        <button class="close" type="button" data-dismiss="alert">
          <span aria-hidden="true">×</span>
          <span class="sr-only">Close</span>
        </button>
        <p>Primitive created successfully</p>
      </div>
    </x-alert>
    The hawk selenuim test can't see the <div class="allert-success"...>
    because it's inside the #shadow-root,
    To make it visible for the selenium test here is a small tweak:
    give this class to the <x-alert class="allert-success" ...>
    */
    this.classList.add(`alert-${type}`);

    const wrapper = document.createElement("div");
    wrapper.setAttribute("role", "alert");
    wrapper.classList.add("alert", "alert-dismissible", `alert-${type}`);
    if (!visible) wrapper.style.display = "none";

    // Close button
    const closeBtn = document.createElement("button");
    closeBtn.className = "close";
    closeBtn.setAttribute("type", "button");
    closeBtn.setAttribute("data-dismiss", "alert");

    const spanX = document.createElement("span");
    spanX.setAttribute("aria-hidden", "true");
    spanX.textContent = "×";

    const spanSR = document.createElement("span");
    spanSR.className = "sr-only";
    spanSR.textContent = "Close";

    closeBtn.appendChild(spanX);
    closeBtn.appendChild(spanSR);

    // Message wrapped in <p> like Rails simple_format
    const p = document.createElement("p");
    p.textContent = message;

    // Append all
    wrapper.appendChild(closeBtn);
    wrapper.appendChild(p);

    // Styles (shadow DOM)
    const style = document.createElement("style");
    style.textContent = `
      .alert {
        padding: 15px;
        margin-bottom: 20px;
        border: 1px solid transparent;
        border-radius: 4px;
      }

      /* bootstrap-like color sets */
      .alert-success { color:#3c763d; background:#dff0d8; border-color:#d6e9c6; }
      .alert-danger  { color:#a94442; background:#f2dede; border-color:#ebccd1; }
      .alert-warning { color:#8a6d3b; background:#fcf8e3; border-color:#faebcc; }
      .alert-info    { color:#31708f; background:#d9edf7; border-color:#bce8f1; }

      .close {
        float: right;
        font-size: 21px;
        font-weight: 700;
        line-height: 1;
        color: #000;
        opacity: .2;
        background: none;
        border: 0;
        cursor: pointer;
        position: relative;
        /* This pixel shifting is ugly,
         but I couldn't make it looking
         like the old hawk in Ruby */
        top: -2px;
        right: -8px;
      }
      .close:hover { opacity: .5; }
      .sr-only {
        position:absolute;
        width:1px;height:1px;
        padding:0;margin:-1px;
        overflow:hidden;
        clip:rect(0,0,0,0);
        border:0;
      }
      p { margin: 0; }
    `;

    this.shadowRoot.appendChild(style);
    this.shadowRoot.appendChild(wrapper);

    // Dismiss logic (Bootstrap-like)
    closeBtn.addEventListener("click", () => {
      wrapper.style.display = "none";
    });
  }
}

customElements.define("x-alert", XAlert);
