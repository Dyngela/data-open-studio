// Database types enum
export enum DbType {
  Postgres = 'postgres',
  SQLServer = 'sqlserver',
  MySQL = 'mysql',
}

export interface DataModel {
  name: string;
  type: string;
  goType: string;
  nullable: boolean;
  length?: number;
  precision?: number;
  scale?: number;
}

// ============ DB Metadata ============

// Response: DB Metadata
export interface DbMetadata {
  id: number;
  host: string;
  port: number;
  databaseType: DbType;
  user: string;
  password: string;
  databaseName: string;
  sslMode: string;
  extra: string;
}

// Request: Create DB Metadata
export interface CreateDbMetadataRequest {
  host: string;
  port: number;
  user: string;
  password: string;
  databaseName: string;
  sslMode?: string;
  databaseType: DbType;
}

// Request: Update DB Metadata
export interface UpdateDbMetadataRequest {
  host?: string;
  port?: number;
  user?: string;
  password?: string;
  databaseName?: string;
  sslMode?: string;
  databaseType?: DbType;
}

// ============ SFTP Metadata ============

// Response: SFTP Metadata
export interface SftpMetadata {
  id: number;
  host: string;
  port: number;
  user: string;
  password: string;
  privateKey: string;
  basePath: string;
  extra: string;
}

// Request: Create SFTP Metadata
export interface CreateSftpMetadataRequest {
  host: string;
  port: number;
  user: string;
  password?: string;
  privateKey?: string;
  basePath?: string;
  extra?: string;
}

// Request: Update SFTP Metadata
export interface UpdateSftpMetadataRequest {
  host?: string;
  port?: number;
  user?: string;
  password?: string;
  privateKey?: string;
  basePath?: string;
  extra?: string;
}

// ============ Common ============

// Response: Test connection result
export interface TestConnectionResult {
  success: boolean;
  message: string;
  version?: string;
}

// Response: Delete operation
export interface DeleteResponse {
  id: number;
  deleted: boolean;
}
