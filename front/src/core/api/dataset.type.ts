// Dataset types matching backend models

export type DatasetStatus = 'draft' | 'ready' | 'error';
export type QueryFilterOperator = 'eq' | 'neq' | 'gt' | 'lt' | 'gte' | 'lte' | 'like';

export interface DatasetColumn {
  name: string;
  dataType: 'string' | 'integer' | 'float' | 'date' | 'datetime' | 'boolean';
  nullable: boolean;
}

export interface DatasetSchema {
  columns: DatasetColumn[];
}

// Summary (list view)
export interface Dataset {
  id: number;
  name: string;
  description: string;
  creatorId: number;
  metadataDatabaseId: number;
  status: DatasetStatus;
  columnCount: number;
  lastRefreshedAt?: string;
  lastError?: string;
  createdAt: string;
  updatedAt: string;
}

// Full detail (editor view)
export interface DatasetWithDetails {
  id: number;
  name: string;
  description: string;
  creatorId: number;
  metadataDatabaseId: number;
  query: string;
  schema: DatasetSchema;
  status: DatasetStatus;
  lastRefreshedAt?: string;
  lastError?: string;
  createdAt: string;
  updatedAt: string;
}

// Requests
export interface CreateDatasetRequest {
  name: string;
  description?: string;
  metadataDatabaseId: number;
  query: string;
}

export interface UpdateDatasetRequest {
  name?: string;
  description?: string;
  metadataDatabaseId?: number;
  query?: string;
}

export interface DatasetPreviewRequest {
  limit?: number;
}

export interface DatasetQueryFilter {
  column: string;
  operator: QueryFilterOperator;
  value: string | number | boolean;
}

export interface DatasetQueryRequest {
  filters?: DatasetQueryFilter[];
  limit?: number;
}

// Responses
export interface DatasetPreviewResult {
  columns: string[];
  rows: Record<string, unknown>[];
  rowCount: number;
}

export interface DatasetQueryResult {
  columns: string[];
  rows: Record<string, unknown>[];
  rowCount: number;
}

export interface DeleteResponse {
  id: number;
  deleted: boolean;
}
