import { Injectable } from '@angular/core';
import { NodeType } from '../core/nodes-services/node.type';
import { ALL_NODE_DEFINITIONS, ApiNodeType } from './index';

@Injectable({
  providedIn: 'root',
})
export class NodeRegistryService {
  private nodeTypes: NodeType[] = ALL_NODE_DEFINITIONS.map(def => ({
    id: def.id,
    label: def.label,
    icon: def.icon,
    color: def.color,
    hasDataInput: def.hasDataInput,
    hasDataOutput: def.hasDataOutput,
    hasFlowInput: def.hasFlowInput,
    hasFlowOutput: def.hasFlowOutput,
    type: def.type,
  }));

  /** Map from frontend id → API type (e.g. 'db-input' → 'db_input') */
  private idToApi = new Map<string, ApiNodeType>(
    ALL_NODE_DEFINITIONS.map(d => [d.id, d.apiType]),
  );

  /** Map from API type → NodeType (e.g. 'db_input' → NodeType) */
  private apiToNodeType = new Map<string, NodeType>(
    ALL_NODE_DEFINITIONS.map(d => [
      d.apiType,
      this.nodeTypes.find(n => n.id === d.id)!,
    ]),
  );

  getNodeTypes(): NodeType[] {
    return this.nodeTypes;
  }

  getNodeTypeById(id: string): NodeType | undefined {
    return this.nodeTypes.find(type => type.id === id);
  }

  /** Returns the API type string for a frontend node id. */
  getApiType(nodeTypeId: string): ApiNodeType {
    return this.idToApi.get(nodeTypeId) ?? 'start';
  }

  /** Returns the frontend NodeType for a given API type string. */
  getNodeTypeFromApiType(apiType: string): NodeType {
    return this.apiToNodeType.get(apiType) ?? this.nodeTypes[0];
  }
}
