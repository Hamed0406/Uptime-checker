const API_BASE = (window.API_BASE ?? "") || ""; // allow override if you serve behind a path

const rowsEl = document.getElementById("rows");
const addForm = document.getElementById("add-form");
const addMsg = document.getElementById("add-msg");
const urlInput = document.getElementById("url-input");
const refreshBtn = document.getElementById("refresh");

// Hook for future auth
function authHeaders() {
  const headers = { "Content-Type": "application/json" };
  const token = localStorage.getItem("api_token");
  if (token) headers["Authorization"] = `Bearer ${token}`;
  return headers;
}

async function listTargets() {
  const [targetsRes, latestRes] = await Promise.all([
    fetch(`${API_BASE}/api/targets`),
    fetch(`${API_BASE}/api/results/latest`),
  ]);

  if (!targetsRes.ok || !latestRes.ok) {
    rowsEl.innerHTML = `<tr><td colspan="5" class="py-3 text-red-600">Failed to load data.</td></tr>`;
    return;
  }

  const targets = await targetsRes.json();
  const latest = await latestRes.json(); // array of {target_id,url,up,http_status,latency_ms,reason,checked_at}

  const latestById = new Map(latest.map(r => [r.target_id, r]));
  rowsEl.innerHTML = "";

  (targets || []).forEach(t => {
    const r = latestById.get(t.id) || {};
    const up = r.up === true ? "✅" : (r.up === false ? "❌" : "—");
    const http = r.http_status ?? "—";
    const lat = r.latency_ms ?? "—";
    const at  = r.checked_at ? new Date(r.checked_at).toLocaleString() : "—";

    const tr = document.createElement("tr");
    tr.innerHTML = `
      <td class="py-2 pr-4">${escapeHtml(t.url)}</td>
      <td class="py-2 pr-4">${up}</td>
      <td class="py-2 pr-4">${http}</td>
      <td class="py-2 pr-4">${lat}</td>
      <td class="py-2">${at}</td>
    `;
    rowsEl.appendChild(tr);
  });

  if ((targets || []).length === 0) {
    rowsEl.innerHTML = `<tr><td colspan="5" class="py-3 text-slate-500">No targets yet. Add one above.</td></tr>`;
  }
}

addForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  addMsg.textContent = "";
  const url = urlInput.value.trim();
  if (!url) return;

  try {
    const res = await fetch(`${API_BASE}/api/targets`, {
      method: "POST",
      headers: authHeaders(),
      body: JSON.stringify({ url }),
    });
    if (!res.ok) {
      const txt = await res.text();
      addMsg.textContent = `Add failed: ${txt}`;
      addMsg.className = "text-sm mt-2 text-red-600";
      return;
    }
    addMsg.textContent = `Added: ${url}`;
    addMsg.className = "text-sm mt-2 text-green-600";
    urlInput.value = "";
    await listTargets();
  } catch (err) {
    addMsg.textContent = `Error: ${err.message}`;
    addMsg.className = "text-sm mt-2 text-red-600";
  }
});

refreshBtn.addEventListener("click", listTargets);

function escapeHtml(s) {
  return s.replace(/[&<>"']/g, c => (
    { "&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;","'":"&#39;" }[c]
  ));
}

// initial load
listTargets();
