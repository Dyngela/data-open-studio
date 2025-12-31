import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { CdkDrag, CdkDropList } from '@angular/cdk/drag-drop';
import { NodeRegistryService } from '../services/node-registry.service';
import { NodeType } from '../models/node.model';

@Component({
  selector: 'app-node-panel',
  standalone: true,
  imports: [CommonModule, CdkDrag, CdkDropList],
  templateUrl: './node-panel.html',
  styleUrl: './node-panel.css',
})
export class NodePanel {
  nodeTypes = signal<NodeType[]>([]);

  constructor(private nodeRegistry: NodeRegistryService) {
    this.nodeTypes.set(this.nodeRegistry.getNodeTypes());
  }
}
