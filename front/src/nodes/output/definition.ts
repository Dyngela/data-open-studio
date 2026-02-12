import { DataModel, DbType } from '../../core/api/metadata.type';
import { NodeDefinition } from '../node-definition.type';
import { DbConnectionConfig } from '../db-input/definition';

// ── Config types ────────────────────────────────────────
export type DbOutputMode = 'insert' | 'update' | 'merge' | 'delete' | 'truncate';

export interface OutputNodeConfig {
  kind: 'output';
  table: string;
  mode: DbOutputMode;
  batchSize: number;
  dbSchema: string;
  connection: DbConnectionConfig;
  connectionId?: string;
  dataModels: DataModel[];
  keyColumns: string[];
}

// ── Guard ───────────────────────────────────────────────
export function isOutputConfig(config: unknown): config is OutputNodeConfig {
  return (config as OutputNodeConfig)?.kind === 'output';
}

// ── Definition ──────────────────────────────────────────
export const outputDefinition: NodeDefinition<OutputNodeConfig> = {
  id: 'output',
  apiType: 'db_output',
  label: 'Output',
  icon: 'pi pi-save',
  color: '#F44336',
  hasDataInput: true,
  hasDataOutput: false,
  hasFlowInput: true,
  hasFlowOutput: false,
  type: 'output',
  configKind: 'output',
  isConfig: isOutputConfig,
};
