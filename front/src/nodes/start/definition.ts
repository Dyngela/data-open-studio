import { NodeDefinition } from '../node-definition.type';

// ── Config ──────────────────────────────────────────────
export interface StartNodeConfig {
  kind: 'start';
}

// ── Guard ───────────────────────────────────────────────
export function isStartConfig(config: unknown): config is StartNodeConfig {
  return (config as StartNodeConfig)?.kind === 'start';
}

// ── Definition ──────────────────────────────────────────
export const startDefinition: NodeDefinition<StartNodeConfig> = {
  id: 'start',
  apiType: 'start',
  label: 'Start',
  icon: 'pi pi-play',
  color: '#4CAF50',
  hasDataInput: false,
  hasDataOutput: false,
  hasFlowInput: false,
  hasFlowOutput: true,
  type: 'start',
  configKind: 'start',
  isConfig: isStartConfig,
};
