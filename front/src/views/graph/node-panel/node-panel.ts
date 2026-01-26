import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { CdkDrag, CdkDropList } from '@angular/cdk/drag-drop';
import { NodeRegistryService } from '../../../core/services/node-registry.service';
import { NodeType } from '../../../core/services/node.type';

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
