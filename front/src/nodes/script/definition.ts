import { NodeDefinition } from '../node-definition.type';

// ── Config ──────────────────────────────────────────────
export interface ScriptNodeConfig {
  kind: 'script';
}

// ── Guard ───────────────────────────────────────────────
export function isScriptConfig(config: unknown): config is ScriptNodeConfig {
  return (config as ScriptNodeConfig)?.kind === 'script';
}

// ── Definition ──────────────────────────────────────────
export const scriptDefinition: NodeDefinition<ScriptNodeConfig> = {
  id: 'script',
  apiType: 'script',
  label: 'Script',
  icon: 'pi pi-code',
  color: '#6f42c1',
  hasDataInput: true,
  hasDataOutput: true,
  hasFlowInput: true,
  hasFlowOutput: true,
  type: 'script',
  configKind: 'script',
  isConfig: isScriptConfig,
};
