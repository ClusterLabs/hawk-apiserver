console.log("node_edit.js loaded");

//const nodeAttributes = document.getElementById('details-node-attributes');
//nodeAttributes.innerHTML = '<summary><strong>Attributes</strong></summary>';

const fieldShortdesc = document.getElementById('field-shortdesc');
const fieldLongdesc = document.getElementById('field-longdesc');

/* custom elements x-select and x-operations-kvgroup
 * already have their own listeners.
 * input-resource-id is only left w/o it's own listener. */
const inputNodeID = document.getElementById("input-node-id");
inputNodeID.addEventListener("mouseenter", () => {
  fieldShortdesc.textContent = "Node ID";
  fieldLongdesc.textContent = "Unique identifier for the node.";
});

const inputNodeName = document.getElementById("input-node-name");
inputNodeName.addEventListener("mouseenter", () => {
  fieldShortdesc.textContent = "Node Name";
  fieldLongdesc.textContent = "Name used to refer to the node in the cluster.";
});

function clickApplyButton() {

  const attributes = document.getElementById("kvgroup-node_attributes");
  const utilizations = document.getElementById("kvgroup-node_utilizations");

  let result = attributes.submitSanityCheck();
  if (!result.ok) {
    showFlash("danger", result.message);
    return;
  }

  result = utilizations.submitSanityCheck();
  if (!result.ok) {
    showFlash("danger", result.message);
    return;
  }

  Promise.all([
    attributes.submit(),
    utilizations.submit(),
  ])
    .then(() => {
      // refresh the page after submitting all 3
      const url = new URL(window.location.href);
      url.searchParams.set("flash", "updated");

      // wait 1 sec and update the page.
      setTimeout(() => {
        window.location.href = url.toString();
      }, 1000);
    })
    .catch(err => {
      // TODO?: refresh the page even after fail
      console.error("Apply failed:", err);
      showFlash("danger", `There was a problem updating the node:\n${err.message}`);
    });
  return;
}