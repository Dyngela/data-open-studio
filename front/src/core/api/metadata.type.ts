// Database types enum
export enum DbType {
  Postgres = 'postgres',
  SQLServer = 'sqlserver',
  MySQL = 'mysql',
}

// Data model returned from DB schema introspection
export interface DataModel {
  name: string;
  type: string;
  goType: string;
  nullable: boolean;
  length?: number;
  precision?: number;
  scale?: number;
}

// Request payload for guessing data model
export interface DbNodeGuessDataModelRequest {
  nodeId: string;
  jobId: number;
  query: string;
  dbType: DbType;
  dbSchema: string;
  host: string;
  port: number;
  database: string;
  username: string;
  password: string;
  sslMode: string;
  extra?: Record<string, string>;
  dsn?: string;
}

// Response payload for guessed data model
export interface DbNodeGuessDataModelResponse {
  nodeId: number;
  jobId: number;
  dataModels: DataModel[];
}

// DB MetadataService types
export interface DbMetadata {
  id: number;
  host: string;
  port: string;
  user: string;
  password: string;
  databaseName: string;
  sslMode: string;
  extra: string;
}

// SFTP MetadataService types
export interface SftpMetadata {
  id: number;
  host: string;
  port: string;
  user: string;
  password: string;
  privateKey: string;
  basePath: string;
  extra: string;
}
