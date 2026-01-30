import { DataModel } from '../../core/api/metadata.type';
import { NodeDefinition } from '../node-definition.type';

// ── Config types ────────────────────────────────────────
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

// ── Guard ───────────────────────────────────────────────
export function isMapConfig(config: unknown): config is MapNodeConfig {
  return (config as MapNodeConfig)?.kind === 'map';
}

// ── Definition ──────────────────────────────────────────
export const transformDefinition: NodeDefinition<MapNodeConfig> = {
  id: 'transform',
  apiType: 'map',
  label: 'Transform',
  icon: 'pi pi-sync',
  color: '#FF9800',
  hasDataInput: true,
  hasDataOutput: true,
  hasFlowInput: true,
  hasFlowOutput: true,
  type: 'process',
  configKind: 'map',
  isConfig: isMapConfig,
};
