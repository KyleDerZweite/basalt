import assert from "node:assert/strict";
import { test } from "node:test";
import { buildEvidenceGraph } from "../src/lib/evidence.ts";

test("the inspector retains real connections and neighbors omitted from display categories", () => {
  const graph = buildEvidenceGraph({
    nodes: [
      { id: "account:alice", label: "Alice", type: "account", source_module: "github", confidence: 0.9, wave: 1, pivot: true },
      { id: "username:friend", label: "Friend", type: "username", source_module: "github", confidence: 0.7, properties: { relationship: "following" } },
      { id: "email:alice@example.com", label: "alice@example.com", type: "email", source_module: "github", confidence: 0.8 },
    ],
    edges: [
      { id: "e1", source: "account:alice", target: "username:friend", type: "follows", source_module: "github" },
      { id: "e2", source: "account:alice", target: "email:alice@example.com", type: "has_email", source_module: "github" },
    ],
  }, [{ id: "raw:account:alice", label: "Alice", type: "account", category: "accounts", raw_node_ids: ["account:alice"] }]);
  assert.deepEqual(graph.edges.map((edge) => edge.type), ["follows", "has_email"]);
  assert.deepEqual(graph.edges.map((edge) => edge.target), ["username:friend", "email:alice@example.com"]);
  assert.equal(graph.nodes.length, 3);
  assert.equal(graph.nodes[0].category, "accounts");
  assert.equal(graph.nodes[0].wave, 1);
  assert.equal(graph.nodes[0].pivot, true);
  assert.equal(graph.nodes[1].properties?.relationship, "following");
});

test("the inspector handles an unavailable graph without category connections", () => {
  assert.deepEqual(buildEvidenceGraph(undefined), { nodes: [], edges: [] });
});
