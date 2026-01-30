import { DataModel, DbType } from '../../core/api/metadata.type';
import { NodeDefinition } from '../node-definition.type';

// ── Config types ────────────────────────────────────────
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

// ── Guard ───────────────────────────────────────────────
export function isDbInputConfig(config: unknown): config is DbInputNodeConfig {
  return (config as DbInputNodeConfig)?.kind === 'db-input';
}

// ── Definition ──────────────────────────────────────────
export const dbInputDefinition: NodeDefinition<DbInputNodeConfig> = {
  id: 'db-input',
  apiType: 'db_input',
  label: 'Database Input',
  icon: 'pi pi-database',
  color: '#2196F3',
  hasDataInput: false,
  hasDataOutput: true,
  hasFlowInput: true,
  hasFlowOutput: true,
  type: 'input',
  configKind: 'db-input',
  isConfig: isDbInputConfig,
};
