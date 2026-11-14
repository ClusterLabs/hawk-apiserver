"use strict";

document.querySelectorAll("pre code").forEach(block => hljs.highlightBlock(block));
const graph = document.querySelector(".graph-image img");
if (graph) {
    const ready = () => {
        document.querySelector(".graph-placeholder").hidden = true;
        if (!graph.naturalWidth) {
            graph.hidden = true;
            document.querySelector(".graph-error").hidden = false;
        }
    };
    graph.addEventListener("load", ready);
    graph.addEventListener("error", ready);
    if (graph.complete) ready();
}
document.getElementById("monitoringMenu").classList.remove("in");
document.getElementById("configurationMenu").classList.add("in");
document.querySelector('a[href="/cib/live/config/edit"]').setAttribute("aria-current", "page");
