import { Injectable } from '@angular/core';
import { NodeType } from '../models/node.model';

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
      inputs: 0,
      outputs: 1,
    },
    {
      id: 'db-input',
      label: 'Database Input',
      icon: 'pi pi-database',
      color: '#2196F3',
      inputs: 1,
      outputs: 1,
    },
    {
      id: 'transform',
      label: 'Transform',
      icon: 'pi pi-sync',
      color: '#FF9800',
      inputs: 1,
      outputs: 1,
    },
    {
      id: 'filter',
      label: 'Filter',
      icon: 'pi pi-filter',
      color: '#9C27B0',
      inputs: 1,
      outputs: 1,
    },
    {
      id: 'output',
      label: 'Output',
      icon: 'pi pi-save',
      color: '#F44336',
      inputs: 1,
      outputs: 0,
    },
  ];

  getNodeTypes(): NodeType[] {
    return this.nodeTypes;
  }

  getNodeTypeById(id: string): NodeType | undefined {
    return this.nodeTypes.find((type) => type.id === id);
  }
}
