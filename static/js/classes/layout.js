const POLL_INTERVAL_MS = 5000;

function updateClusterStatusIndicator(data) {
  const summary = data.Summary;
  const status = data.NameValues[0]?.Value || "unknown";

  const circle = document.getElementById("cluster-status-indicator");
  if (!circle) return;

  const icon = circle.querySelector("i");

  // Update tooltip content
  circle.setAttribute("title", summary);
  circle.setAttribute("data-original-title", summary);
  $(circle).tooltip("fixTitle");

  // Update visual appearance #TODO: make them like in hawk
  if (status?.toLowerCase().includes("error")) {
    icon.className = "fas fa-times text-danger";
    circle.style.backgroundColor = "#dc3545";
  } else if (status?.toLowerCase().includes("warn")) {
    icon.className = "fas fa-exclamation text-warning";
    circle.style.backgroundColor = "#ffc107";
  } else if (status?.toLowerCase().includes("unknown")) {
    icon.className = "fas fa-question text-muted";
    circle.style.backgroundColor = "#6c757d"; // muted gray
  } else {
    icon.className = "fas fa-check text-success";
    circle.style.backgroundColor = "#28a745";
  }
}

async function pollClusterStatus() {
  while (true) {
    try {
      const lastEpoch = sessionStorage.getItem("cibEpoch") || "";
      const res = await fetch(`/monitor?${lastEpoch}`); // long-polling, returns when cib.xml is changed
      if (!res.ok) throw new Error(`Monitor error: ${res.status}`);
      const { epoch: currentEpoch } = await res.json();

      if (currentEpoch !== lastEpoch) {
        sessionStorage.setItem("cibEpoch", currentEpoch);

        const statusRes = await fetch("/api/data-interface/fetch-cluster-details", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ host: window.location.hostname }),
        });

        if (!statusRes.ok) throw new Error(`Details error: ${statusRes.status}`);
        const data = await statusRes.json();

        updateClusterStatusIndicator(data);
      }
    } catch (err) {
      console.error("[Cluster Status Poll] Failed:", err);
    }

    /* The long polling doesn't require the delay
     * but just in case, let's add at least one second */
    await new Promise(resolve => setTimeout(resolve, 1000));
  }
}

pollClusterStatus();

