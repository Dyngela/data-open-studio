# Node Types

Each node type lives in its own directory under `front/src/nodes/<type>/`.

## Directory structure

```
nodes/
  node-definition.type.ts   # NodeDefinition<T> interface + ApiNodeType union
  node-registry.service.ts  # Injectable service: getNodeTypes(), getApiType(), etc.
  index.ts                  # Barrel: NodeConfig union, ALL_NODE_DEFINITIONS, re-exports

  start/
    definition.ts            # StartNodeConfig + guard + startDefinition
    start.modal.ts/html/css  # Modal component

  db-input/
    definition.ts            # DbConnectionConfig, DbInputNodeConfig + guard + dbInputDefinition
    db-input.modal.ts/html/css
    db-input-canvas.ts/html/css

  transform/
    definition.ts            # InputFlow, OutputFlow, MapNodeConfig + guard + transformDefinition
    transform.modal.ts/html/css
    map-canvas.ts/html/css

  log/
    definition.ts            # LogConfig + guard + logDefinition
    log.modal.ts/html/css
    log-canvas.ts/html/css

  output/
    definition.ts            # OutputNodeConfig (placeholder, no UI yet)
```

## Adding a new node type

### 1. Create the directory and definition

Create `nodes/<my-node>/definition.ts`:

```typescript
import { NodeDefinition } from '../node-definition.type';

export interface MyNodeConfig {
  kind: 'my-node';
  // ... your config fields
}

export function isMyNodeConfig(config: unknown): config is MyNodeConfig {
  return (config as MyNodeConfig)?.kind === 'my-node';
}

export const myNodeDefinition: NodeDefinition<MyNodeConfig> = {
  id: 'my-node',
  apiType: 'my_node',          // must match backend enum
  label: 'My Node',
  icon: 'pi pi-box',
  color: '#9C27B0',
  hasDataInput: true,
  hasDataOutput: true,
  hasFlowInput: true,
  hasFlowOutput: true,
  type: 'process',
  configKind: 'my-node',
  isConfig: isMyNodeConfig,
};
```

### 2. Register in the barrel

In `nodes/index.ts`:

- Add re-exports for your config, guard, and definition
- Add your config to the `NodeConfig` union
- Add your definition to `ALL_NODE_DEFINITIONS`

### 3. Update `ApiNodeType`

In `nodes/node-definition.type.ts`, add `'my_node'` to the `ApiNodeType` union.

### 4. Create UI components (optional)

- **Modal** (`my-node.modal.ts/html/css`): The configuration dialog opened on double-click.
- **Canvas** (`my-node-canvas.ts/html/css`): The inline preview rendered inside the node body on the playground.

### 5. Wire up in playground views

- `views/graph/playground/playground.ts`: import the modal, add to `imports` array
- `views/graph/playground/playground.html`: add an `@if` block inside the modal overlay
- `views/graph/node-instance/node-instance.ts`: import the canvas, add to `imports` array
- `views/graph/node-instance/node-instance.html`: add a `@case` inside the `@switch` block

### 6. Handle in `JobStateService` (if the node produces output schema)

If your node has data output, update `getOutputSchema()` in
`core/nodes-services/job-state.service.ts` to handle your config type.

### 7. Add backend support

Add the matching API node type in the Go backend (`api/internal/api/models/`).
