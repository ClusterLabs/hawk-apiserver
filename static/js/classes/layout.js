async function updateClusterStatusIndicator() {

  const statusRes = await fetch("/api/cib/cluster/details/fetch", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ host: window.location.hostname }),
  });

  if (!statusRes.ok) throw new Error(`Details error: ${statusRes.status}`);
  const data = await statusRes.json();

  setClusterStatusIndicator(data.summary, data.status);
}

updateClusterStatusIndicator(); // call it once to initialize
pollClusterStatus(updateClusterStatusIndicator);
