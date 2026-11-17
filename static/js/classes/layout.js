async function updateClusterStatusIndicator() {

  const statusRes = await statusFetch("/api/cib/cluster/details/fetch", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ host: window.location.hostname }),
  });

  if (!statusRes.ok) throw new Error(`Details error: ${statusRes.status}`);
  const data = await statusRes.json();

  setClusterStatusIndicator(data.summary, data.status);
}

if (!window.batchConfig) {
  updateClusterStatusIndicator();
  pollClusterStatus(updateClusterStatusIndicator);
}
