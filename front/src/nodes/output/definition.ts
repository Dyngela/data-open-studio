import { NodeDefinition } from '../node-definition.type';

// ── Config (placeholder — no UI yet) ────────────────────
export interface OutputNodeConfig {
  kind: 'output';
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
