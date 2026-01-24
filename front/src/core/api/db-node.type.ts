// Request payload for guessing schema
import {DataModel, DbType} from './metadata.type';

export interface GuessSchemaRequest {
  nodeId: string;
  query: string;
  dbType: DbType;
  dbSchema?: string;
  host: string;
  port: number;
  database: string;
  username: string;
  password: string;
  sslMode?: string;
  extra?: Record<string, string>;
  dsn?: string;
}

// Response from guess schema endpoint
export interface GuessSchemaResponse {
  nodeId: string;
  dataModels: DataModel[];
}
