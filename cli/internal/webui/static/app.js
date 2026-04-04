const state = {
  bootstrap: null,
  settings: null,
  scans: [],
  health: [],
  currentScan: null,
  currentEvents: [],
  currentSelection: null,
  route: { name: "home", scanID: "" },
  error: "",
  eventSource: null,
  refreshTimer: null,
};

const appRoot = document.getElementById("app");
const eventTypes = [
  "scan_queued",
  "scan_status",
  "module_verified",
  "verify_complete",
  "module_started",
  "module_finished",
  "node_discovered",
  "edge_discovered",
  "module_error",
  "scan_finished",
  "scan_failed",
];

document.addEventListener("DOMContentLoaded", init);

async function init() {
  bindNavigation();
  try {
    await Promise.all([loadBootstrap(), loadSettings(), loadScans()]);
    await routeToCurrentPath();
  } catch (error) {
    state.error = error.message || String(error);
    render();
  }
}

function bindNavigation() {
  window.addEventListener("popstate", () => {
    void routeToCurrentPath();
  });

  document.addEventListener("click", (event) => {
    const link = event.target.closest("[data-nav]");
    if (!link) {
      return;
    }
    event.preventDefault();
    navigate(link.getAttribute("href"));
  });
}

function navigate(path) {
  window.history.pushState({}, "", path);
  void routeToCurrentPath();
}

async function routeToCurrentPath() {
  closeEventStream();
  const path = window.location.pathname;
  if (path === "/" || path === "") {
    state.route = { name: "home", scanID: "" };
    state.currentScan = null;
    state.currentEvents = [];
    state.currentSelection = null;
    await loadHomeData();
    render();
    return;
  }
  if (path === "/new") {
    state.route = { name: "new", scanID: "" };
    state.currentScan = null;
    state.currentEvents = [];
    state.currentSelection = null;
    render();
    bindNewScanPage();
    return;
  }
  if (path === "/settings") {
    state.route = { name: "settings", scanID: "" };
    state.currentScan = null;
    state.currentEvents = [];
    state.currentSelection = null;
    await loadSettings();
    render();
    bindSettingsPage();
    return;
  }
  if (path.startsWith("/scans/")) {
    const scanID = decodeURIComponent(path.slice("/scans/".length));
    state.route = { name: "scan", scanID };
    await loadScanWorkspace(scanID);
    render();
    bindScanPage();
    attachEventStream(scanID);
    return;
  }

  state.route = { name: "home", scanID: "" };
  state.error = `Unknown route: ${path}`;
  render();
}

async function loadHomeData() {
  const [health, scans] = await Promise.all([getModuleHealth(), getScans()]);
  state.health = health;
  state.scans = scans;
}

async function loadBootstrap() {
  state.bootstrap = await api("/app/bootstrap");
}

async function loadSettings() {
  state.settings = await api("/api/settings");
}

async function loadScans() {
  state.scans = await getScans();
}

async function loadScanWorkspace(scanID) {
  const [record, events] = await Promise.all([
    api(`/api/scans/${encodeURIComponent(scanID)}`),
    api(`/api/scans/${encodeURIComponent(scanID)}/events`),
  ]);
  state.currentScan = record;
  state.currentEvents = Array.isArray(events.events) ? events.events : [];
  ensureSelectionExists();
}

function ensureSelectionExists() {
  if (!state.currentScan || !state.currentScan.graph) {
    state.currentSelection = null;
    return;
  }

  const nodes = state.currentScan.graph.nodes || [];
  const edges = state.currentScan.graph.edges || [];
  if (!state.currentSelection) {
    if (nodes.length > 0) {
      state.currentSelection = { kind: "node", id: nodes[0].id };
    } else if (edges.length > 0) {
      state.currentSelection = { kind: "edge", id: edges[0].id };
    }
    return;
  }

  const exists =
    state.currentSelection.kind === "node"
      ? nodes.some((node) => node.id === state.currentSelection.id)
      : edges.some((edge) => edge.id === state.currentSelection.id);
  if (!exists) {
    state.currentSelection = null;
    ensureSelectionExists();
  }
}

function attachEventStream(scanID) {
  if (!state.currentScan) {
    return;
  }
  if (!["queued", "verifying", "running"].includes(state.currentScan.status)) {
    return;
  }

  const after = state.currentEvents.length > 0 ? state.currentEvents[state.currentEvents.length - 1].sequence : 0;
  const source = new EventSource(`/api/scans/${encodeURIComponent(scanID)}/events?stream=1&after=${after}`);
  state.eventSource = source;

  for (const type of eventTypes) {
    source.addEventListener(type, (event) => {
      try {
        const payload = JSON.parse(event.data);
        mergeEvent(payload);
      } catch (error) {
        console.error("failed to parse event", error);
      }
    });
  }

  source.onerror = () => {
    if (source.readyState === EventSource.CLOSED) {
      return;
    }
    scheduleScanRefresh(1200);
  };
}

function closeEventStream() {
  if (state.eventSource) {
    state.eventSource.close();
    state.eventSource = null;
  }
  if (state.refreshTimer) {
    window.clearTimeout(state.refreshTimer);
    state.refreshTimer = null;
  }
}

function mergeEvent(event) {
  if (!state.currentEvents.some((item) => item.sequence === event.sequence)) {
    state.currentEvents.push(event);
    state.currentEvents.sort((a, b) => a.sequence - b.sequence);
  }

  if (event.type === "scan_status" && event.data && event.data.status && state.currentScan) {
    state.currentScan.status = event.data.status;
  }
  if (event.type === "scan_failed" && state.currentScan && event.message) {
    state.currentScan.error_message = event.message;
    state.currentScan.status = "failed";
  }
  if (event.type === "scan_finished" && state.currentScan && event.data && event.data.status) {
    state.currentScan.status = event.data.status;
  }

  render();
  scheduleScanRefresh(350);
}

function scheduleScanRefresh(delay) {
  if (state.refreshTimer) {
    return;
  }
  state.refreshTimer = window.setTimeout(async () => {
    state.refreshTimer = null;
    if (state.route.name !== "scan" || !state.route.scanID) {
      return;
    }
    try {
      await loadScanWorkspace(state.route.scanID);
      render();
      bindScanPage();
      if (state.currentScan && !["queued", "verifying", "running"].includes(state.currentScan.status)) {
        closeEventStream();
      }
      void loadScans();
    } catch (error) {
      state.error = error.message || String(error);
      render();
    }
  }, delay);
}

function render() {
  appRoot.innerHTML = `
    <div class="shell">
      <aside class="sidebar">
        <h1>Basalt</h1>
        <p>Local OSINT workspace</p>
        <nav class="nav">
          ${navLink("/", "Home", state.route.name === "home")}
          ${navLink("/new", "New Scan", state.route.name === "new")}
          ${navLink("/settings", "Settings", state.route.name === "settings")}
        </nav>
      </aside>
      <main class="content">
        ${state.error ? `<div class="error-box">${escapeHTML(state.error)}</div>` : ""}
        ${renderRoute()}
      </main>
    </div>
  `;
}

function renderRoute() {
  switch (state.route.name) {
    case "new":
      return renderNewScan();
    case "scan":
      return renderScanWorkspace();
    case "settings":
      return renderSettings();
    case "home":
    default:
      return renderHome();
  }
}

function renderHome() {
  const healthCounts = summarizeHealth(state.health);
  return `
    <section class="page-header">
      <div>
        <h2>Recent Scans</h2>
        <p>Start scans, inspect persisted results, and monitor module health.</p>
      </div>
      <div class="actions">
        <a class="button" href="/new" data-nav>Start scan</a>
      </div>
    </section>
    <section class="panel-grid">
      <div class="panel">
        <h3>Runtime</h3>
        <div class="stats">
          <div class="stat"><span>Version</span><strong>${escapeHTML(state.bootstrap?.version || "unknown")}</strong></div>
          <div class="stat"><span>Data directory</span><strong>${escapeHTML(state.bootstrap?.data_dir || "unknown")}</strong></div>
          <div class="stat"><span>Config path</span><strong>${escapeHTML(state.bootstrap?.default_config_path || "unknown")}</strong></div>
        </div>
      </div>
      <div class="panel">
        <h3>Module Health</h3>
        <div class="stats">
          <div class="stat"><span>Healthy</span><strong>${healthCounts.healthy}</strong></div>
          <div class="stat"><span>Degraded</span><strong>${healthCounts.degraded}</strong></div>
          <div class="stat"><span>Offline</span><strong>${healthCounts.offline}</strong></div>
        </div>
      </div>
      <div class="panel">
        <h3>Settings Summary</h3>
        <div class="list">
          <div class="list-item"><strong>Strict mode</strong><div class="muted">${state.settings?.strict_mode ? "Enabled" : "Disabled"}</div></div>
          <div class="list-item"><strong>Disabled modules</strong><div class="muted">${escapeHTML((state.settings?.disabled_modules || []).join(", ") || "None")}</div></div>
          <div class="list-item"><strong>Legal notice accepted</strong><div class="muted">${formatDate(state.settings?.legal_accepted_at) || "Not recorded"}</div></div>
        </div>
      </div>
    </section>
    <section class="panel" style="margin-top: 20px;">
      <h3>Recent Scan History</h3>
      ${renderScanList(state.scans)}
    </section>
  `;
}

function renderScanList(scans) {
  if (!Array.isArray(scans) || scans.length === 0) {
    return `<div class="empty">No persisted scans yet.</div>`;
  }
  return `
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Started</th>
            <th>Status</th>
            <th>Seeds</th>
            <th>Nodes</th>
            <th>Edges</th>
          </tr>
        </thead>
        <tbody>
          ${scans
            .map(
              (scan) => `
                <tr>
                  <td><a href="/scans/${encodeURIComponent(scan.id)}" data-nav>${escapeHTML(formatDate(scan.started_at))}</a></td>
                  <td>${statusPill(scan.status)}</td>
                  <td>${escapeHTML(formatSeeds(scan.seeds))}</td>
                  <td>${escapeHTML(String(scan.node_count || 0))}</td>
                  <td>${escapeHTML(String(scan.edge_count || 0))}</td>
                </tr>`
            )
            .join("")}
        </tbody>
      </table>
    </div>
  `;
}

function renderNewScan() {
  return `
    <section class="page-header">
      <div>
        <h2>New Scan</h2>
        <p>Create a new local scan job and follow it live in the workspace.</p>
      </div>
    </section>
    <section class="panel">
      <form id="scan-form">
        <div class="stack" id="seed-list">
          ${renderSeedRow(0, "username", "")}
        </div>
        <div class="actions" style="margin-bottom: 16px;">
          <button type="button" class="button secondary" id="add-seed">Add seed</button>
        </div>
        <div class="panel-grid">
          <div>
            <div class="field">
              <label for="depth">Depth</label>
              <input id="depth" name="depth" type="number" min="1" value="2" />
            </div>
            <div class="field">
              <label for="concurrency">Concurrency</label>
              <input id="concurrency" name="concurrency" type="number" min="1" value="5" />
            </div>
            <div class="field">
              <label for="timeout">Timeout Seconds</label>
              <input id="timeout" name="timeout" type="number" min="1" value="10" />
            </div>
          </div>
          <div>
            <div class="field">
              <label for="disabled-modules">Disabled Modules</label>
              <textarea id="disabled-modules" name="disabled-modules" placeholder="github, reddit"></textarea>
            </div>
            <label class="checkbox">
              <input id="strict-mode" name="strict-mode" type="checkbox" />
              <span>Strict mode</span>
            </label>
          </div>
        </div>
        <div class="actions" style="margin-top: 16px;">
          <button type="submit" class="button">Start scan</button>
          <a class="button secondary" href="/" data-nav>Cancel</a>
        </div>
      </form>
    </section>
  `;
}

function renderSeedRow(index, type, value) {
  return `
    <div class="seed-row" data-seed-row="${index}">
      <div class="field">
        <label>Seed Type</label>
        <select name="seed-type">
          <option value="username" ${type === "username" ? "selected" : ""}>username</option>
          <option value="email" ${type === "email" ? "selected" : ""}>email</option>
          <option value="domain" ${type === "domain" ? "selected" : ""}>domain</option>
        </select>
      </div>
      <div class="field">
        <label>Seed Value</label>
        <input name="seed-value" value="${escapeAttribute(value)}" />
      </div>
      <div class="actions">
        <button type="button" class="button secondary seed-remove">Remove</button>
      </div>
    </div>
  `;
}

function renderScanWorkspace() {
  if (!state.currentScan) {
    return `<div class="empty">Loading scan workspace...</div>`;
  }
  const graph = state.currentScan.graph || { nodes: [], edges: [], meta: {} };
  const nodes = graph.nodes || [];
  const edges = graph.edges || [];
  const selection = getSelectedItem(nodes, edges);
  const moduleErrors = state.currentEvents.filter((event) => event.type === "module_error");

  return `
    <section class="page-header">
      <div>
        <h2>Scan Workspace</h2>
        <p>Scan ${escapeHTML(state.currentScan.id)}</p>
      </div>
      <div class="actions">
        <a class="button secondary" href="/api/scans/${encodeURIComponent(state.currentScan.id)}/export?format=json">Export JSON</a>
        <a class="button secondary" href="/api/scans/${encodeURIComponent(state.currentScan.id)}/export?format=csv">Export CSV</a>
        ${["queued", "verifying", "running"].includes(state.currentScan.status) ? '<button type="button" class="button danger" id="cancel-scan">Cancel scan</button>' : ""}
      </div>
    </section>
    <section class="panel-grid">
      <div class="panel">
        <h3>Status</h3>
        <div class="stats">
          <div class="stat"><span>State</span><strong>${escapeHTML(state.currentScan.status)}</strong></div>
          <div class="stat"><span>Nodes</span><strong>${escapeHTML(String(state.currentScan.node_count || 0))}</strong></div>
          <div class="stat"><span>Edges</span><strong>${escapeHTML(String(state.currentScan.edge_count || 0))}</strong></div>
        </div>
        <div class="list" style="margin-top: 14px;">
          <div class="list-item"><strong>Started</strong><div class="muted">${escapeHTML(formatDate(state.currentScan.started_at))}</div></div>
          <div class="list-item"><strong>Completed</strong><div class="muted">${escapeHTML(formatDate(state.currentScan.completed_at) || "Still running")}</div></div>
          <div class="list-item"><strong>Seeds</strong><div class="muted">${escapeHTML(formatSeeds(state.currentScan.seeds))}</div></div>
        </div>
      </div>
      <div class="panel">
        <h3>Health</h3>
        ${renderHealthTable(state.currentScan.health || [])}
      </div>
      <div class="panel">
        <h3>Evidence</h3>
        ${
          selection
            ? `<div class="list-item"><strong>${escapeHTML(selection.kind)}</strong><pre>${escapeHTML(JSON.stringify(selection.item, null, 2))}</pre></div>`
            : '<div class="empty">Select a node or edge to inspect its evidence.</div>'
        }
      </div>
    </section>
    <section class="panel-grid" style="margin-top: 20px;">
      <div class="panel">
        <h3>Nodes</h3>
        ${renderNodesTable(nodes)}
      </div>
      <div class="panel">
        <h3>Edges</h3>
        ${renderEdgesTable(edges)}
      </div>
    </section>
    <section class="panel-grid" style="margin-top: 20px;">
      <div class="panel">
        <h3>Timeline</h3>
        ${renderEventTimeline(state.currentEvents)}
      </div>
      <div class="panel">
        <h3>Errors and Warnings</h3>
        ${
          state.currentScan.error_message || moduleErrors.length > 0
            ? `<div class="list">
                ${
                  state.currentScan.error_message
                    ? `<div class="list-item"><strong>Scan failure</strong><div class="muted">${escapeHTML(state.currentScan.error_message)}</div></div>`
                    : ""
                }
                ${moduleErrors
                  .map(
                    (event) => `
                      <div class="list-item">
                        <strong>${escapeHTML(event.module || "module")}</strong>
                        <div class="muted">${escapeHTML(event.message || "module error")}</div>
                      </div>`
                  )
                  .join("")}
              </div>`
            : '<div class="empty">No module errors recorded for this scan.</div>'
        }
      </div>
    </section>
  `;
}

function renderHealthTable(health) {
  if (!Array.isArray(health) || health.length === 0) {
    return '<div class="empty">Health information is not available yet.</div>';
  }
  return `
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Module</th>
            <th>Status</th>
            <th>Message</th>
          </tr>
        </thead>
        <tbody>
          ${health
            .map(
              (item) => `
                <tr>
                  <td>${escapeHTML(item.name)}</td>
                  <td>${statusPill(item.status)}</td>
                  <td>${escapeHTML(item.message || "")}</td>
                </tr>`
            )
            .join("")}
        </tbody>
      </table>
    </div>
  `;
}

function renderNodesTable(nodes) {
  if (!nodes.length) {
    return '<div class="empty">No nodes persisted for this scan yet.</div>';
  }
  return `
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Label</th>
            <th>Type</th>
            <th>Module</th>
            <th>Confidence</th>
          </tr>
        </thead>
        <tbody>
          ${nodes
            .map((node) => {
              const selected = state.currentSelection?.kind === "node" && state.currentSelection.id === node.id;
              return `
                <tr class="${selected ? "selected" : ""}" data-select-kind="node" data-select-id="${escapeAttribute(node.id)}">
                  <td>${escapeHTML(node.label)}</td>
                  <td>${escapeHTML(node.type)}</td>
                  <td>${escapeHTML(node.source_module || "")}</td>
                  <td>${escapeHTML(formatConfidence(node.confidence))}</td>
                </tr>`;
            })
            .join("")}
        </tbody>
      </table>
    </div>
  `;
}

function renderEdgesTable(edges) {
  if (!edges.length) {
    return '<div class="empty">No edges persisted for this scan yet.</div>';
  }
  return `
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Type</th>
            <th>Source</th>
            <th>Target</th>
            <th>Module</th>
          </tr>
        </thead>
        <tbody>
          ${edges
            .map((edge) => {
              const selected = state.currentSelection?.kind === "edge" && state.currentSelection.id === edge.id;
              return `
                <tr class="${selected ? "selected" : ""}" data-select-kind="edge" data-select-id="${escapeAttribute(edge.id)}">
                  <td>${escapeHTML(edge.type)}</td>
                  <td>${escapeHTML(edge.source)}</td>
                  <td>${escapeHTML(edge.target)}</td>
                  <td>${escapeHTML(edge.source_module || "")}</td>
                </tr>`;
            })
            .join("")}
        </tbody>
      </table>
    </div>
  `;
}

function renderEventTimeline(events) {
  if (!events.length) {
    return '<div class="empty">No events recorded yet.</div>';
  }
  return `
    <div class="event-log">
      ${events
        .slice(-40)
        .reverse()
        .map(
          (event) => `
            <div class="event">
              <div class="row" style="justify-content: space-between;">
                <strong>${escapeHTML(event.type)}</strong>
                <span class="muted">${escapeHTML(formatDate(event.time))}</span>
              </div>
              <div class="muted">${escapeHTML(event.module || event.message || "")}</div>
            </div>`
        )
        .join("")}
    </div>
  `;
}

function renderSettings() {
  return `
    <section class="page-header">
      <div>
        <h2>Settings</h2>
        <p>Persist local defaults used by future scans.</p>
      </div>
    </section>
    <section class="panel">
      <form id="settings-form">
        <label class="checkbox" style="margin-bottom: 16px;">
          <input id="settings-strict-mode" type="checkbox" ${state.settings?.strict_mode ? "checked" : ""} />
          <span>Strict mode</span>
        </label>
        <div class="field">
          <label for="settings-disabled-modules">Disabled Modules</label>
          <textarea id="settings-disabled-modules">${escapeHTML((state.settings?.disabled_modules || []).join(", "))}</textarea>
        </div>
        <div class="field">
          <label for="settings-legal-accepted-at">Legal Accepted At</label>
          <input id="settings-legal-accepted-at" value="${escapeAttribute(state.settings?.legal_accepted_at || "")}" placeholder="2026-04-04T12:00:00Z" />
        </div>
        <div class="actions">
          <button type="submit" class="button">Save settings</button>
          <button type="button" class="button secondary" id="legal-now">Accept legal notice now</button>
        </div>
      </form>
    </section>
    <section class="panel" style="margin-top: 20px;">
      <h3>Runtime</h3>
      <div class="list">
        <div class="list-item"><strong>Version</strong><div class="muted">${escapeHTML(state.bootstrap?.version || "unknown")}</div></div>
        <div class="list-item"><strong>Data directory</strong><div class="muted">${escapeHTML(state.bootstrap?.data_dir || "unknown")}</div></div>
        <div class="list-item"><strong>Config path</strong><div class="muted">${escapeHTML(state.bootstrap?.default_config_path || "unknown")}</div></div>
      </div>
    </section>
  `;
}

function bindNewScanPage() {
  const seedList = document.getElementById("seed-list");
  const addSeedButton = document.getElementById("add-seed");
  const form = document.getElementById("scan-form");
  if (!seedList || !addSeedButton || !form) {
    return;
  }

  addSeedButton.addEventListener("click", () => {
    const nextIndex = seedList.querySelectorAll("[data-seed-row]").length;
    seedList.insertAdjacentHTML("beforeend", renderSeedRow(nextIndex, "username", ""));
  });

  seedList.addEventListener("click", (event) => {
    const button = event.target.closest(".seed-remove");
    if (!button) {
      return;
    }
    const rows = seedList.querySelectorAll("[data-seed-row]");
    if (rows.length === 1) {
      return;
    }
    button.closest("[data-seed-row]").remove();
  });

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const seeds = Array.from(seedList.querySelectorAll("[data-seed-row]"))
      .map((row) => ({
        type: row.querySelector('select[name="seed-type"]').value.trim(),
        value: row.querySelector('input[name="seed-value"]').value.trim(),
      }))
      .filter((seed) => seed.value !== "");

    if (seeds.length === 0) {
      state.error = "At least one seed is required.";
      render();
      bindNewScanPage();
      return;
    }

    const payload = {
      seeds,
      depth: numberValue(document.getElementById("depth").value, 2),
      concurrency: numberValue(document.getElementById("concurrency").value, 5),
      timeout_seconds: numberValue(document.getElementById("timeout").value, 10),
      strict_mode: document.getElementById("strict-mode").checked,
      disabled_modules: splitList(document.getElementById("disabled-modules").value),
    };

    try {
      state.error = "";
      const record = await api("/api/scans", {
        method: "POST",
        body: JSON.stringify(payload),
      });
      await loadScans();
      navigate(`/scans/${encodeURIComponent(record.id)}`);
    } catch (error) {
      state.error = error.message || String(error);
      render();
      bindNewScanPage();
    }
  });
}

function bindScanPage() {
  const cancelButton = document.getElementById("cancel-scan");
  if (cancelButton && state.currentScan) {
    cancelButton.addEventListener("click", async () => {
      try {
        await api(`/api/scans/${encodeURIComponent(state.currentScan.id)}/cancel`, { method: "POST" });
      } catch (error) {
        state.error = error.message || String(error);
        render();
      }
    });
  }

  document.querySelectorAll("[data-select-kind]").forEach((row) => {
    row.addEventListener("click", () => {
      state.currentSelection = {
        kind: row.getAttribute("data-select-kind"),
        id: row.getAttribute("data-select-id"),
      };
      render();
      bindScanPage();
    });
  });
}

function bindSettingsPage() {
  const form = document.getElementById("settings-form");
  const legalNow = document.getElementById("legal-now");
  if (!form) {
    return;
  }

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const acceptedAt = document.getElementById("settings-legal-accepted-at").value.trim();
    const payload = {
      strict_mode: document.getElementById("settings-strict-mode").checked,
      disabled_modules: splitList(document.getElementById("settings-disabled-modules").value),
      legal_accepted_at: acceptedAt || null,
    };
    try {
      state.settings = await api("/api/settings", {
        method: "PUT",
        body: JSON.stringify(payload),
      });
      state.error = "";
      render();
      bindSettingsPage();
    } catch (error) {
      state.error = error.message || String(error);
      render();
      bindSettingsPage();
    }
  });

  if (legalNow) {
    legalNow.addEventListener("click", () => {
      document.getElementById("settings-legal-accepted-at").value = new Date().toISOString();
    });
  }
}

function getSelectedItem(nodes, edges) {
  if (!state.currentSelection) {
    return null;
  }
  if (state.currentSelection.kind === "node") {
    const node = nodes.find((item) => item.id === state.currentSelection.id);
    return node ? { kind: "node", item: node } : null;
  }
  const edge = edges.find((item) => item.id === state.currentSelection.id);
  return edge ? { kind: "edge", item: edge } : null;
}

function summarizeHealth(health) {
  const summary = { healthy: 0, degraded: 0, offline: 0 };
  for (const item of health || []) {
    if (item.status === "healthy") {
      summary.healthy += 1;
    } else if (item.status === "degraded") {
      summary.degraded += 1;
    } else {
      summary.offline += 1;
    }
  }
  return summary;
}

async function getScans() {
  const payload = await api("/api/scans");
  return Array.isArray(payload.scans) ? payload.scans : [];
}

async function getModuleHealth() {
  const payload = await api("/api/modules/health");
  return Array.isArray(payload.modules) ? payload.modules : [];
}

async function api(url, options = {}) {
  const response = await fetch(url, {
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
    ...options,
  });

  const contentType = response.headers.get("Content-Type") || "";
  const isJSON = contentType.includes("application/json");
  const payload = isJSON ? await response.json() : await response.text();
  if (!response.ok) {
    const message = isJSON && payload.error ? payload.error : String(payload || response.statusText);
    throw new Error(message);
  }
  return payload;
}

function navLink(href, label, active) {
  return `<a href="${href}" data-nav class="${active ? "active" : ""}">${label}</a>`;
}

function statusPill(status) {
  return `<span class="pill status-${escapeAttribute(status || "")}">${escapeHTML(status || "unknown")}</span>`;
}

function formatSeeds(seeds) {
  return (seeds || []).map((seed) => `${seed.type}:${seed.value}`).join(", ");
}

function formatDate(value) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

function formatConfidence(value) {
  const number = Number(value || 0);
  return number.toFixed(2);
}

function splitList(value) {
  return value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter((item) => item !== "");
}

function numberValue(value, fallback) {
  const parsed = Number.parseInt(value, 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

function escapeHTML(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function escapeAttribute(value) {
  return escapeHTML(value).replaceAll("`", "&#96;");
}
