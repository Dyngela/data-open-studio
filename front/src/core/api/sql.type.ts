export interface GuessQueryRequest {
  prompt: string;
  connectionId: number;
  schemaOptimizationNeeded?: boolean;
  previousMessages?: string[];
}

export interface GuessQueryResponse {
  query: string;
}

export interface OptimizeQueryRequest {
  query: string;
  connectionId: number;
}

export interface OptimizeQueryResponse {
  optimizedQuery: string;
  explanation: string;
}

// Database introspection types
export type { DBConnectionConfig } from './trigger.type';

export interface DatabaseTable {
  schema: string;
  name: string;
}

export interface DatabaseColumn {
  name: string;
  dataType: string;
  isNullable: boolean;
  isPrimary: boolean;
}

export interface DatabaseIntrospection {
  tables?: DatabaseTable[];
  columns?: DatabaseColumn[];
}

export interface TestConnectionResult {
  success: boolean;
  message: string;
  version?: string;
}

export interface TestConnectionRequest {
  connection: import('./trigger.type').DBConnectionConfig;
}

export interface IntrospectDatabaseRequest {
  metadataDatabaseId?: number;
  connection?: import('./trigger.type').DBConnectionConfig;
  tableName?: string;
}
