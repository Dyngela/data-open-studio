import { Component, computed, input, inject } from '@angular/core';
import { LayoutService } from '../../core/services/layout-service';
import { KuiModalHeader } from '../../ui/modal/kui-modal-header/kui-modal-header';
import { NodeInstance } from '../../core/nodes-services/node.type';
import { NodeGraphService } from '../../core/nodes-services/node-graph.service';

@Component({
  selector: 'app-start-modal',
  standalone: true,
  imports: [KuiModalHeader],
  templateUrl: './start.modal.html',
  styleUrl: './start.modal.css',
})
export class StartModal {
  private layout = inject(LayoutService);
  private nodeGraph = inject(NodeGraphService);
  node = input.required<NodeInstance>();
  modalTitle = computed(() => this.node().name ?? this.node().type.label);

  onCancel() {
    this.layout.closeModal();
  }

  onTitleChange(value: string) {
    const trimmed = value.trim();
    this.nodeGraph.renameNode(this.node().id, trimmed || this.node().type.label);
  }
}
