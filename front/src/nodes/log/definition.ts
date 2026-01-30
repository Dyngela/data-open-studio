import { DataModel } from '../../core/api/metadata.type';
import { NodeDefinition } from '../node-definition.type';

// ── Config ──────────────────────────────────────────────
export interface LogConfig {
  kind: 'log';
  input?: DataModel[];
}

// ── Guard ───────────────────────────────────────────────
export function isLogConfig(config: unknown): config is LogConfig {
  return (config as LogConfig)?.kind === 'log';
}

// ── Definition ──────────────────────────────────────────
export const logDefinition: NodeDefinition<LogConfig> = {
  id: 'log',
  apiType: 'log',
  label: 'Log',
  icon: 'pi pi-terminal',
  color: 'rgba(165,165,158,0.93)',
  hasDataInput: true,
  hasDataOutput: false,
  hasFlowInput: true,
  hasFlowOutput: false,
  type: 'output',
  configKind: 'log',
  isConfig: isLogConfig,
};
