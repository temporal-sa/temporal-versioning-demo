"use strict";

// Pizza Tracker SPA — vanilla JS, no build step.
// Consumes the backend's SSE stream of DashboardState snapshots and renders
// the full dashboard on every frame (state is small: tens of orders), then
// wires the rollout action buttons to the REST endpoints.

const VERSION_CLASS = { v1: "b-v1", v2: "b-v2", v3: "b-v3" };

const els = {
  dep: document.getElementById("dep"),
  inflight: document.getElementById("kpi-inflight"),
  current: document.getElementById("kpi-current"),
  ramping: document.getElementById("kpi-ramping"),
  ordersCount: document.getElementById("orders-count"),
  orders: document.getElementById("orders"),
  versions: document.getElementById("versions"),
  promote: document.getElementById("btn-promote"),
  rollback: document.getElementById("btn-rollback"),
  recover: document.getElementById("btn-recover"),
  toast: document.getElementById("toast"),
};

const rampButtons = Array.from(document.querySelectorAll("[data-ramp]"));

// Latest state, used by button handlers (e.g. highlight the active ramp %).
let lastState = null;

function versionClass(version) {
  return VERSION_CLASS[version] || "b-v1";
}

function badge(version) {
  return `<span class="vb ${versionClass(version)}">${version}</span>`;
}

function formatElapsed(seconds) {
  const total = Math.max(0, Math.floor(seconds || 0));
  const m = Math.floor(total / 60);
  const s = total % 60;
  return `${m}:${String(s).padStart(2, "0")}`;
}

function escapeHtml(value) {
  return String(value).replace(
    /[&<>"']/g,
    (c) =>
      ({
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        '"': "&quot;",
        "'": "&#39;",
      })[c],
  );
}

function renderKpis(state) {
  const k = state.kpis || {};
  els.dep.textContent = state.deploymentName || "—";
  els.inflight.textContent = k.inFlight != null ? k.inFlight : "—";
  els.current.innerHTML = k.currentVersion ? badge(k.currentVersion) : "—";

  if (k.rampingVersion) {
    const pct = k.rampingPct != null ? ` <small>${k.rampingPct}%</small>` : "";
    els.ramping.innerHTML = badge(k.rampingVersion) + pct;
  } else {
    els.ramping.textContent = "—";
  }
}

function renderStepper(order) {
  const steps = order.steps || [];
  const nodes = steps.map((label, i) => {
    let nodeClass = "";
    let glyph = "";

    if (order.done || i < order.currentStep) {
      nodeClass = "done";
      glyph = "✓";
    } else if (i === order.currentStep) {
      if (order.failing) {
        nodeClass = "err";
        glyph = "✕";
      } else {
        nodeClass = "cur";
      }
    }

    const node = `<div class="node ${nodeClass}"><div class="dot">${glyph}</div><div class="lbl">${escapeHtml(label)}</div></div>`;

    // Connector after every node except the last; filled up to the current step.
    if (i < steps.length - 1) {
      const fill = order.done || i < order.currentStep ? " fill" : "";
      return node + `<div class="conn${fill}"></div>`;
    }
    return node;
  });

  return `<div class="stepper">${nodes.join("")}</div>`;
}

function renderOrder(order) {
  const failClass = order.failing ? " fail" : "";
  const name = order.pizza
    ? `<span class="nm">${escapeHtml(order.pizza)}</span>`
    : "";
  const retry = order.failing
    ? `<span class="retry">⟳ retry ${order.retryCount || 0}×</span>`
    : "";
  const elapsed = `<span class="el">${formatElapsed(order.elapsedSec)}</span>`;

  return `<div class="order${failClass}">
      <div class="oh">${badge(order.version)}<b>#${escapeHtml(order.id)}</b>${name}${retry}${elapsed}</div>
      ${renderStepper(order)}
    </div>`;
}

function renderOrders(state) {
  const orders = state.orders || [];
  els.ordersCount.textContent = orders.length;
  els.orders.innerHTML = orders.map(renderOrder).join("");
}

function statusChip(version) {
  switch (version.status) {
    case "CURRENT":
      return '<span class="chip c-cur">CURRENT</span>';
    case "RAMPING":
      return `<span class="chip c-ramp">RAMPING ${version.trafficPct || 0}%</span>`;
    case "DRAINING":
      return '<span class="chip c-drain">DRAINING</span>';
    default:
      return '<span class="chip c-inact">INACTIVE</span>';
  }
}

function renderVersion(version, failingByVersion) {
  const inactiveClass = version.status === "INACTIVE" ? " inactive" : "";
  const cssVar = `var(--${version.version === "v1" ? "blue" : version.version === "v2" ? "green" : "amber"})`;
  const width = Math.max(0, Math.min(100, version.trafficPct || 0));

  const failing = failingByVersion[version.version] || 0;
  const pin =
    failing > 0
      ? `<div class="pin warn">⚠ ${failing} failing</div>`
      : `<div class="pin">🔒 ${version.pinnedCount || 0}</div>`;

  return `<div class="ver${inactiveClass}">
      <div class="vrow">${badge(version.version)}${statusChip(version)}</div>
      <div class="bar"><span style="width:${width}%;background:${cssVar}"></span></div>
      ${pin}
    </div>`;
}

function renderVersions(state) {
  const versions = state.versions || [];
  const orders = state.orders || [];

  // Count failing orders per version so a version card can flag its slice.
  const failingByVersion = {};
  for (const o of orders) {
    if (o.failing) {
      failingByVersion[o.version] = (failingByVersion[o.version] || 0) + 1;
    }
  }

  els.versions.innerHTML = versions
    .map((v) => renderVersion(v, failingByVersion))
    .join("");
}

function renderControls(state) {
  const pct = state.kpis ? state.kpis.rampingPct : null;
  for (const btn of rampButtons) {
    btn.classList.toggle("active", Number(btn.dataset.ramp) === pct);
  }
}

function render(state) {
  lastState = state;
  renderKpis(state);
  renderOrders(state);
  renderVersions(state);
  renderControls(state);
}

let toastTimer = null;

function showToast(message) {
  els.toast.textContent = message;
  els.toast.classList.add("show");
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => els.toast.classList.remove("show"), 4000);
}

// POST to an action endpoint, briefly disabling the button and surfacing any
// failure as a toast. Returns the parsed JSON body (or null).
async function postAction(button, path, body) {
  button.disabled = true;
  try {
    const res = await fetch(path, {
      method: "POST",
      headers: body ? { "Content-Type": "application/json" } : undefined,
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) {
      throw new Error(`${res.status} ${res.statusText}`);
    }
    const text = await res.text();
    return text ? JSON.parse(text) : null;
  } catch (err) {
    showToast(`Action failed: ${err.message}`);
    return null;
  } finally {
    // Re-enable shortly after; the next SSE frame refreshes button state.
    setTimeout(() => {
      button.disabled = false;
    }, 600);
  }
}

function wireButtons() {
  for (const btn of rampButtons) {
    btn.addEventListener("click", () => {
      postAction(btn, "/api/ramp", { percentage: Number(btn.dataset.ramp) });
    });
  }
  els.promote.addEventListener("click", () => {
    postAction(els.promote, "/api/promote");
  });
  els.rollback.addEventListener("click", () => {
    postAction(els.rollback, "/api/rollback");
  });
  els.recover.addEventListener("click", async () => {
    const result = await postAction(els.recover, "/api/recover");
    if (result && result.recovered != null) {
      showToast(`Recovered ${result.recovered} stuck order(s)`);
    }
  });
}

function connect() {
  const es = new EventSource("/events");
  es.onmessage = (e) => {
    try {
      render(JSON.parse(e.data));
    } catch (err) {
      console.error("Failed to parse state", err);
    }
  };
  // EventSource auto-reconnects on error; no manual handling needed.
}

wireButtons();
connect();
