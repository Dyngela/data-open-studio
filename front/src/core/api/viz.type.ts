// ---------------------------------------------------------------------------
// Workspaces
// ---------------------------------------------------------------------------

export interface Workspace {
  id: string;
  name: string;
  created_at: string;
}

export interface CreateWorkspaceRequest {
  name: string;
}

export interface UpdateWorkspaceRequest {
  name: string;
}

export interface WorkspaceListResponse {
  workspaces: Workspace[];
}

// ---------------------------------------------------------------------------
// Sources
// ---------------------------------------------------------------------------

export type SourceType = 'csv' | 'postgres';

export interface CsvSourceConfig {
  path: string;
}

export interface PostgresSourceConfig {
  connection_string: string;
  query: string;
}

export interface Source {
  id: string;
  workspace_id: string;
  name: string;
  source_type: SourceType;
  config: CsvSourceConfig | PostgresSourceConfig | Record<string, unknown>;
  created_at: string;
}

export interface CreateSourceRequest {
  name: string;
  source_type: SourceType;
  config: CsvSourceConfig | PostgresSourceConfig;
}

export interface SourceListResponse {
  sources: Source[];
}

// ---------------------------------------------------------------------------
// Frames
// ---------------------------------------------------------------------------

export type FrameDtype =
  | 'Boolean'
  | 'UInt8' | 'UInt16' | 'UInt32' | 'UInt64'
  | 'Int8'  | 'Int16'  | 'Int32'  | 'Int64'
  | 'Float32' | 'Float64'
  | 'String' | 'Binary'
  | 'Date' | 'Time' | 'Datetime' | 'Duration'
  | 'List' | 'Struct' | 'Null';

export interface FrameColumn {
  name: string;
  dtype: FrameDtype;
  len: number;
  null_count: number;
}

export interface FrameSchema {
  name: string;
  row_count: number;
  columns: FrameColumn[];
}

export interface FrameListResponse {
  frames: FrameSchema[];
}

/** Column-oriented paginated frame data returned by preview and execute */
export interface FrameData {
  name: string;
  offset: number;
  limit: number;
  row_count: number;
  columns: Record<string, unknown[]>;
}

export interface LoadSourceResponse {
  loaded: FrameSchema;
}

// ---------------------------------------------------------------------------
// Execute
// ---------------------------------------------------------------------------

export interface ExecuteRequest {
  script: string;
  /** Frame name whose data should be included in the response */
  result?: string;
  /** Max rows to return (default 100) */
  limit?: number;
}

export interface ExecuteResponse {
  materialized: FrameSchema[];
  result: FrameData | null;
}
