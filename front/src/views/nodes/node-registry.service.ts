import { Injectable } from '@angular/core';
import { NodeType } from './node.type';

@Injectable({
  providedIn: 'root',
})
export class NodeRegistryService {
  private nodeTypes: NodeType[] = [
    {
      id: 'start',
      label: 'Start',
      icon: 'pi pi-play',
      color: '#4CAF50',
      hasDataInput: false,
      hasDataOutput: false,
      hasFlowInput: false,
      hasFlowOutput: true,
      type: 'start',
    },
    {
      id: 'db-input',
      label: 'Database Input',
      icon: 'pi pi-database',
      color: '#2196F3',
      hasDataInput: false,
      hasDataOutput: true,
      hasFlowInput: true,
      hasFlowOutput: true,
      type: 'input',
    },
    {
      id: 'transform',
      label: 'Transform',
      icon: 'pi pi-sync',
      color: '#FF9800',
      hasDataInput: true,
      hasDataOutput: true,
      hasFlowInput: true,
      hasFlowOutput: true,
      type: 'process',
    },
    {
      id: 'output',
      label: 'Output',
      icon: 'pi pi-save',
      color: '#F44336',
      hasDataInput: true,
      hasDataOutput: false,
      hasFlowInput: true,
      hasFlowOutput: false,
      type: 'output',
    },
  ];

  getNodeTypes(): NodeType[] {
    return this.nodeTypes;
  }

  getNodeTypeById(id: string): NodeType | undefined {
    return this.nodeTypes.find((type) => type.id === id);
  }
}
