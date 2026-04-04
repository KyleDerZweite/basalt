export type Seed = { type: string; value: string };

export type ModuleStatus = {
  name: string;
  description: string;
  status: string;
  message: string;
};

export type TargetAlias = {
  id: string;
  target_id: string;
  seed_type: string;
  seed_value: string;
  label?: string;
  is_primary?: boolean;
};

export type Target = {
  id: string;
  slug: string;
  display_name: string;
  notes?: string;
  aliases?: TargetAlias[];
};

export type RawNode = {
  id: string;
  type: string;
  label: string;
  source_module: string;
  confidence: number;
  properties?: Record<string, unknown>;
};

export type RawEdge = {
  id: string;
  source: string;
  target: string;
  type: string;
  source_module: string;
};

export type RawGraph = { nodes: RawNode[]; edges: RawEdge[] };

export type InsightFinding = {
  title: string;
  summary: string;
  node_ids?: string[];
  profile_url?: string;
  confidence?: number;
  category?: string;
  source_label?: string;
};

export type ScanInsights = {
  headline: string;
  top_findings?: InsightFinding[];
  high_confidence_accounts?: InsightFinding[];
  identity_signals?: string[];
  infrastructure_summary?: string[];
  warnings?: string[];
};

export type ScanRecord = {
  id: string;
  target_id?: string;
  status: string;
  started_at: string;
  completed_at?: string;
  updated_at: string;
  seeds: Seed[];
  health?: ModuleStatus[];
  insights?: ScanInsights;
  node_count: number;
  edge_count: number;
  error_message?: string;
  graph?: RawGraph;
};

export type WorkspaceNode = {
  id: string;
  label: string;
  type: string;
  category: string;
  raw_node_ids?: string[];
  raw_edge_ids?: string[];
  profile_url?: string;
  confidence?: number;
  collapsed_count?: number;
};

export type WorkspaceEdge = {
  id: string;
  source: string;
  target: string;
  type: string;
  raw_edge_ids?: string[];
};

export type ScanWorkspace = {
  record: ScanRecord;
  target?: Target;
  insights?: ScanInsights;
  graph: { layout: string; nodes: WorkspaceNode[]; edges: WorkspaceEdge[] };
  raw_graph_available: boolean;
  raw_node_count: number;
  raw_edge_count: number;
};

export type Bootstrap = {
  version: string;
  data_dir: string;
  default_config_path: string;
  api_base_path: string;
  base_url: string;
};

export type Settings = {
  strict_mode: boolean;
  disabled_modules?: string[];
  legal_accepted_at?: string | null;
};

export type ScanEvent = {
  sequence: number;
  time: string;
  type: string;
  module?: string;
  message?: string;
  data?: Record<string, unknown>;
};

export type ThemeMode = "dark" | "light";

export type GraphNodeData = {
  label: string;
  category: string;
  type: string;
  confidence?: number;
  collapsedCount?: number;
  profileURL?: string;
};
