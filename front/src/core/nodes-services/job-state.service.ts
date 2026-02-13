import { Injectable, signal } from '@angular/core';
import { DataModel } from '../api/metadata.type';
import type { NodeConfig } from '../../nodes/node-definition.type';
import { type DbInputNodeConfig, isDbInputConfig } from '../../nodes/db-input/definition';
import { type InputFlow, type MapNodeConfig, isMapConfig } from '../../nodes/transform/definition';
import { isOutputConfig } from '../../nodes/output/definition';
import { Connection, NodeInstance, PortType } from './node.type';
import { NodeGraphService } from './node-graph.service';
import { inject } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class JobStateService {
  private nodeGraph = inject(NodeGraphService);

  private configs = signal<Map<number, NodeConfig>>(new Map());

  /** Bumped on every config change so computed() consumers re-evaluate. */
  readonly schemaVersion = signal(0);

  setNodeConfig(nodeId: number, config: NodeConfig): void {
    this.configs.update(map => {
      const next = new Map(map);
      next.set(nodeId, config);
      return next;
    });
    this.nodeGraph.updateNodeConfig(nodeId, config);
    this.schemaVersion.update(v => v + 1);
  }

  getNodeConfig<T extends NodeConfig = NodeConfig>(nodeId: number): T | undefined {
    return this.configs().get(nodeId) as T | undefined;
  }

  /**
   * Returns the output schema (DataModel[]) for a given node based on its type.
   *  - db-input: returns config.dataModels
   *  - map: converts output columns to DataModel[]
   */
  getOutputSchema(nodeId: number): DataModel[] {
    // Read schemaVersion so callers in computed() re-trigger
    this.schemaVersion();

    const config = this.configs().get(nodeId);
    if (!config) return [];

    if (isDbInputConfig(config)) {
      return config.dataModels || [];
    }

    if (isMapConfig(config)) {
      if (!config.outputs || config.outputs.length === 0) return [];
      // Flatten all output columns into DataModel[]
      return config.outputs.flatMap(out =>
        out.columns.map(col => ({
          name: col.name,
          type: col.dataType,
          goType: '',
          nullable: false,
        })),
      );
    }

    return [];
  }

  /**
   * Traces DATA connections backward from a node and returns InputFlow[]
   * with upstream output schemas, named "A", "B", ...
   */
  getUpstreamSchemas(nodeId: number): InputFlow[] {
    // Read schemaVersion so callers in computed() re-trigger
    this.schemaVersion();

    const connections = this.nodeGraph.connections();
    const dataInputs = connections.filter(
      c => c.targetNodeId === nodeId && c.targetPortType === PortType.DATA,
    );

    const inputNames = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';

    return dataInputs.map((conn, index) => ({
      name: inputNames[index] || `Input${index}`,
      portId: conn.targetPort,
      schema: this.getOutputSchema(conn.sourceNodeId),
    }));
  }

  /**
   * Traces DATA connections forward from a node and returns the expected
   * input schema of the first connected downstream node that defines one
   * (e.g. a db-output node with dataModels).
   */
  getDownstreamExpectedSchema(sourceNodeId: number): DataModel[] {
    this.schemaVersion();

    const connections = this.nodeGraph.connections();
    const dataOutputs = connections.filter(
      c => c.sourceNodeId === sourceNodeId && c.sourcePortType === PortType.DATA,
    );

    for (const conn of dataOutputs) {
      const config = this.configs().get(conn.targetNodeId);
      if (isOutputConfig(config) && config.dataModels?.length) {
        return config.dataModels;
      }
    }

    return [];
  }

  /**
   * Initialize configs from loaded nodes (e.g. after loading a job).
   */
  loadFromNodes(nodes: NodeInstance[]): void {
    const map = new Map<number, NodeConfig>();
    for (const node of nodes) {
      const cfg = node.config;
      if (cfg && typeof cfg === 'object' && 'kind' in cfg) {
        map.set(node.id, cfg as NodeConfig);
      }
    }
    this.configs.set(map);
    this.schemaVersion.update(v => v + 1);
  }

  reset(): void {
    this.configs.set(new Map());
    this.schemaVersion.set(0);
  }
}
