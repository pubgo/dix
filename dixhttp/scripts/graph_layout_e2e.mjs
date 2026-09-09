#!/usr/bin/env node
/**
 * Layout E2E against a live dixhttp demo (no browser dependency).
 * Fetches /api/modules, builds the same star positions the UI uses, and asserts readability.
 *
 *   DIX_HTTP_ADDR=127.0.0.1:18099 go run -C example ./http
 *   node dixhttp/scripts/graph_layout_e2e.mjs
 */
import { layoutStarPositions, assessLayoutMetrics, shortGraphLabel, resolveCameraStrategy } from "../static/js/graph_state.mjs";

const BASE = process.env.DIX_E2E_BASE || "http://127.0.0.1:18099";

async function main() {
  const health = await fetch(`${BASE}/`);
  if (!health.ok) throw new Error(`demo server not reachable at ${BASE}/ (${health.status})`);

  const html = await health.text();
  if (!html.includes("graph_state.mjs")) throw new Error("/ HTML missing graph_state.mjs");
  if (!html.includes("legacy/app.js")) throw new Error("/ HTML missing legacy/app.js");
  if (!html.includes("模块地图")) throw new Error("/ HTML missing module map control");

  const modules = await (await fetch(`${BASE}/api/modules`)).json();
  if (!Array.isArray(modules) || modules.length < 3) {
    throw new Error(`expected modules list, got ${JSON.stringify(modules).slice(0, 200)}`);
  }

  const nodes = modules.map((m) => ({
    id: m.name,
    label: shortGraphLabel(m.name) + "\n(" + m.provider_count + "p/" + m.object_count + "o)",
  }));
  const edges = [];
  for (const m of modules) {
    for (const dep of m.depends_on || []) {
      edges.push({ from: m.name, to: dep });
    }
  }

  for (const n of nodes) {
    if (n.label.includes("github.com") || n.label.split("/").length > 3) {
      throw new Error(`label not shortened enough: ${n.label}`);
    }
  }

  const camera = resolveCameraStrategy("modules", nodes.length);
  if (camera !== "fit") throw new Error(`expected fit camera, got ${camera}`);

  const positions = layoutStarPositions(nodes, edges);
  const assessment = assessLayoutMetrics(positions);
  if (!assessment.ok) {
    throw new Error(`module map layout not readable: ${JSON.stringify(assessment)}`);
  }

  // Overlap check: pairwise distance for non-hub nodes should stay healthy.
  const ids = Object.keys(positions);
  const hub = positions[ids.find((id) => positions[id].x === 0 && positions[id].y === 0)];
  let minPeer = Infinity;
  for (let i = 0; i < ids.length; i++) {
    for (let j = i + 1; j < ids.length; j++) {
      const a = positions[ids[i]], b = positions[ids[j]];
      const d = Math.hypot(a.x - b.x, a.y - b.y);
      if (d > 0) minPeer = Math.min(minPeer, d);
    }
  }
  if (!Number.isFinite(minPeer) || minPeer < 80) {
    throw new Error(`nodes too close: minPeer=${minPeer}`);
  }

  console.log(JSON.stringify({
    ok: true,
    moduleCount: modules.length,
    camera,
    assessment,
    minPeer: Math.round(minPeer),
    sampleLabels: nodes.slice(0, 5).map((n) => n.label.replace("\n", " ")),
    hubAtOrigin: !!hub,
  }, null, 2));
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
