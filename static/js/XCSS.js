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
      gap: 2px;
    }

    .kv-control .form-control,
    .kv-control input,
    .kv-control select {
      flex: 1 1 auto;
      width: auto;
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

    button.btn i {
      pointer-events: none;
      margin: auto;
    }

    label {
      flex: 0 0 120px;
      text-align: right;
      padding-right: 10px;
    }

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
