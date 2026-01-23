import { Signal } from '@angular/core';

export type WsStatus =
  | 'connecting'
  | 'connected'
  | 'disconnected'
  | 'error';

export interface WsConnection<T = any> {
  status: Signal<WsStatus>;
  error: Signal<any>;
  send: (message: WsMessage) => void;
  close: () => void;
}

// MessageType enum matching backend websocket message types
export enum MessageType {
  // Job operations
  JobUpdate = 'job_update',
  JobDelete = 'job_delete',
  JobCreate = 'job_create',
  JobExecute = 'job_execute',
  JobGet = 'job_get',
  JobProgress = 'job_progress',

  // User interactions
  CursorMove = 'cursor_move',
  Chat = 'chat',
  UserJoin = 'user_join',
  UserLeave = 'user_leave',

  // System messages
  Error = 'error',
  Ping = 'ping',
  Pong = 'pong',

  // DB Metadata operations
  DbMetadataCreate = 'db_metadata_create',
  DbMetadataCreateResponse = 'response_db_metadata_create',
  DbMetadataUpdate = 'db_metadata_update',
  DbMetadataUpdateResponse = 'response_db_metadata_update',
  DbMetadataDelete = 'db_metadata_delete',
  DbMetadataDeleteResponse = 'response_db_metadata_delete',
  DbMetadataGet = 'db_metadata_get',
  DbMetadataGetResponse = 'response_db_metadata_get',
  DbMetadataGetAll = 'db_metadata_get_all',
  DbMetadataGetAllResponse = 'response_db_metadata_get_all',

  // SFTP Metadata operations
  SftpMetadataCreate = 'sftp_metadata_create',
  SftpMetadataCreateResponse = 'response_sftp_metadata_create',
  SftpMetadataUpdate = 'sftp_metadata_update',
  SftpMetadataUpdateResponse = 'response_sftp_metadata_update',
  SftpMetadataDelete = 'sftp_metadata_delete',
  SftpMetadataDeleteResponse = 'response_sftp_metadata_delete',
  SftpMetadataGet = 'sftp_metadata_get',
  SftpMetadataGetResponse = 'response_sftp_metadata_get',
  SftpMetadataGetAll = 'sftp_metadata_get_all',
  SftpMetadataGetAllResponse = 'response_sftp_metadata_get_all',

  // DB Node operations
  DbNodeGuessDataModel = 'db_node_guess_data_model',
  DbNodeGuessDataModelResponse = 'response_db_node_guess_data_model',
}

export interface WsMessage<T = any> {
  type: MessageType;
  jobId?: number;
  userId: number;
  username: string;
  timestamp: string; // ISO string
  data: T;
}

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
  nodeId: number;
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

// DB Metadata types
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

// SFTP Metadata types
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
