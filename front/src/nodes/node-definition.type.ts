/** API-side node type identifiers (snake_case, match backend enum) */
export type ApiNodeType = 'start' | 'db_input' | 'db_output' | 'map' | 'log';

/**
 * Every node type must satisfy this interface.
 * TConfig is the node's specific config shape (e.g. DbInputNodeConfig).
 */
export interface NodeDefinition<TConfig> {
  /** Unique frontend id used in switch/case and registry lookups (e.g. 'db-input') */
  id: string;
  /** Corresponding backend API type (e.g. 'db_input') */
  apiType: ApiNodeType;
  /** Human-readable label shown in the UI */
  label: string;
  /** PrimeNG icon class (e.g. 'pi pi-database') */
  icon: string;
  /** Header colour hex / rgba */
  color: string;
  /** Port capabilities */
  hasDataInput: boolean;
  hasDataOutput: boolean;
  hasFlowInput: boolean;
  hasFlowOutput: boolean;
  type: 'start' | 'input' | 'output' | 'process' | 'log';
  /** The `kind` discriminator value stored in the config (e.g. 'db-input') */
  configKind: TConfig extends { kind: infer K } ? K : never;
  /** Type guard that narrows a generic NodeConfig to TConfig */
  isConfig: (config: unknown) => config is TConfig;
}
