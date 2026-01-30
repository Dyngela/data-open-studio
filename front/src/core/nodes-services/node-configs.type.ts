import { DataModel, DbType } from '../api/metadata.type';

// ============ DB Input Node ============

export interface DbConnectionConfig {
  type: DbType;
  host: string;
  port: number;
  database: string;
  username: string;
  password: string;
  sslMode: string;
  dsn?: string;
}

export interface DbInputNodeConfig {
  kind: 'db-input';
  query: string;
  dbSchema?: string;
  connection: DbConnectionConfig;
  connectionId?: string;
  dataModels: DataModel[];
}

// ============ Map / Transform Node ============

export interface InputFlow {
  name: string;       // "A", "B", ...
  portId: number;
  schema: DataModel[];
}

export type MapFuncType = 'direct' | 'library' | 'custom';

export interface MapOutputCol {
  name: string;
  dataType: string;
  funcType: MapFuncType;
  inputRef?: string;    // "A.column_name"
  libFunc?: string;
  args?: string[];
  expression?: string;
}

export interface OutputFlow {
  name: string;
  portId: number;
  columns: MapOutputCol[];
}

export interface JoinConfig {
  type: 'inner' | 'left' | 'right' | 'full' | 'cross';
  leftInput: string;
  rightInput: string;
  leftKeys: string[];
  rightKeys: string[];
}

export interface MapNodeConfig {
  kind: 'map';
  inputs: InputFlow[];
  outputs: OutputFlow[];
  join?: JoinConfig;
}

// ============ Start Node ============

export interface StartNodeConfig {
  kind: 'start';
}

// ============ Union & Guards ============

export type NodeConfig = DbInputNodeConfig | MapNodeConfig | StartNodeConfig;

export function isDbInputConfig(config: NodeConfig | undefined): config is DbInputNodeConfig {
  return config?.kind === 'db-input';
}

export function isMapConfig(config: NodeConfig | undefined): config is MapNodeConfig {
  return config?.kind === 'map';
}

export function isStartConfig(config: NodeConfig | undefined): config is StartNodeConfig {
  return config?.kind === 'start';
}
