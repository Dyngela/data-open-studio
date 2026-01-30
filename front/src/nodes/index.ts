// ── Re-export every definition module ───────────────────
export {isStartConfig, startDefinition} from './start/definition';
export type { StartNodeConfig } from './start/definition';
export {
  isDbInputConfig,
  dbInputDefinition
} from './db-input/definition';
export type {
    DbConnectionConfig,
    DbInputNodeConfig
} from './db-input/definition';
export {
  isMapConfig,
  transformDefinition
} from './transform/definition';
export type {
    InputFlow,
    MapFuncType,
    MapOutputCol,
    OutputFlow,
    JoinConfig,
    MapNodeConfig
} from './transform/definition';
export {isLogConfig, logDefinition} from './log/definition';
export type { LogConfig } from './log/definition';
export {isOutputConfig, outputDefinition} from './output/definition';
export type { OutputNodeConfig } from './output/definition';

// ── Re-export shared types ──────────────────────────────
export type { NodeDefinition } from './node-definition.type';
export type { ApiNodeType } from './node-definition.type';

// ── NodeConfig union (single source of truth) ───────────
import { StartNodeConfig } from './start/definition';
import { DbInputNodeConfig } from './db-input/definition';
import { MapNodeConfig } from './transform/definition';
import { LogConfig } from './log/definition';
import { OutputNodeConfig } from './output/definition';

export type NodeConfig =
  | StartNodeConfig
  | DbInputNodeConfig
  | MapNodeConfig
  | LogConfig
  | OutputNodeConfig;

import { startDefinition } from './start/definition';
import { dbInputDefinition } from './db-input/definition';
import { transformDefinition } from './transform/definition';
import { logDefinition } from './log/definition';
import { outputDefinition } from './output/definition';
import { NodeDefinition } from './node-definition.type';

export const ALL_NODE_DEFINITIONS: NodeDefinition<any>[] = [
  startDefinition,
  dbInputDefinition,
  transformDefinition,
  outputDefinition,
  logDefinition,
];
