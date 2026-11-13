// static/js/XCSS.js
(function () {
  const sheet = new CSSStyleSheet();

  sheet.replaceSync(`
    .kvsection-row {
      display: flex;
      align-items: center;
      gap: 6px;
      margin-bottom: 8px;
      justify-content: flex-end;
    }

    .form-control, input, select {
      width: 100%;
      padding: 6px 10px;
      border: 1px solid #ccc;
      border-radius: 4px;
      box-sizing: border-box;
      background: #fff;
    }

    /* copy-paste from XSelect */
    .form-control:focus { border-color: #66afe9; outline: 0; box-shadow: 0 0 8px rgba(102, 175, 233, 0.6); }

    .kv-control {
      width: 60%;
      display: flex;
      align-items: center;
      gap: 0;
    }

    .kv-control .form-control,
    .kv-control input,
    .kv-control select {
      flex: 1 1 auto;
      width: auto;
      min-width: 0;
      border-top-right-radius: 0;
      border-bottom-right-radius: 0;
    }

    /* FIX: prevent empty labels (like in the "+" row) from affecting row height */
    label:empty {
      display: none;
    }

    button.btn {
      flex: 0 0 auto;
      width: 30px;
      display: flex;
      align-items: center;
      justify-content: center;
      border: 1px solid #ccc;
      background: white;
      border-radius: 4px;
      cursor: pointer;
      padding: 6px;
    }

    .kv-control button.btn {
      align-self: stretch;
      margin-left: -1px;
      border-top-left-radius: 0;
      border-bottom-left-radius: 0;
      border-top-right-radius: 4px;
      border-bottom-right-radius: 4px;
    }

    button.btn i {
      pointer-events: none;
      margin: auto;
    }

    label {
      flex: 0 0 var(--kv-label-width, 120px);
      white-space: var(--kv-label-white-space, normal);
      text-align: right;
      padding-right: 10px;
    }

    /* Opt in to the legacy horizontal form geometry without changing editors. */
    :host([ruby-layout]) .kvsection-row {
      display: grid;
      grid-template-columns: 5fr 7fr;
      gap: 0;
      margin: 0 -15px 15px;
    }
    :host([ruby-layout]) label {
      box-sizing: border-box;
      width: auto;
      padding: 0 15px;
      font-size: 14px;
      font-weight: 400;
      white-space: nowrap;
    }
    :host([ruby-layout]) .kv-control {
      grid-column: 2;
      width: auto;
      min-width: 0;
      margin: 0 15px;
      height: 34px;
    }
    :host([ruby-layout]) .form-control {
      height: 34px;
      padding: 6px 12px;
      color: #555;
      font: inherit;
      font-size: 14px;
      line-height: 20px;
    }
    :host([ruby-layout]) button.btn {
      width: 44px;
      height: 34px;
      padding: 6px 12px;
      font-size: 14px;
      color: #444;
    }
    :host([ruby-layout]) .kvsection-row:last-child { margin-bottom: 0; }

    /* THIS IS FOR DEBUGGING */
    /*
    label {
      background: rgba(0, 255, 0, 0.15);
    }
    .kvsection-row {
      outline: 3px solid blue !important;
      outline-offset: 3px !important;
    }
    .kv-control {
      outline: 3px solid red !important;
      outline-offset: 2px !important;
      background: blue;
    }
    */
  `);

  window.XCSS = window.XCSS || {};
  window.XCSS.debugSheet = sheet;
})();
