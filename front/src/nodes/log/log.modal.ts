import { Component, computed, inject, input, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NodeInstance } from '../../core/nodes-services/node.type';
import { JobStateService } from '../../core/nodes-services/job-state.service';
import { isLogConfig, LogConfig } from './definition';
import { LayoutService } from '../../core/services/layout-service';
import { KuiModalHeader } from '../../ui/modal/kui-modal-header/kui-modal-header';
import { NodeGraphService } from '../../core/nodes-services/node-graph.service';

@Component({
  selector: 'app-log-modal',
  standalone: true,
  imports: [CommonModule, KuiModalHeader],
  templateUrl: './log.modal.html',
  styleUrl: './log.modal.css',
})
export class LogModal implements OnInit {
  private jobState = inject(JobStateService);
  private layoutService = inject(LayoutService);
  private nodeGraph = inject(NodeGraphService);
  node = input.required<NodeInstance>();
  modalTitle = computed(() => this.node().name ?? this.node().type.label);
  separator = signal(' | ');

  upstreamSchema = computed(() => {
    const schemas = this.jobState.getUpstreamSchemas(this.node().id);
    if (schemas.length === 0) return [];
    return schemas[0].schema;
  });

  ngOnInit() {
    const cfg = this.node().config;
    if (cfg && typeof cfg === 'object' && 'kind' in cfg && isLogConfig(cfg as any)) {
      const typed = cfg as LogConfig;
      if (typed.separator !== undefined) {
        this.separator.set(typed.separator);
      }
    }
  }

  onSave() {
    const separator = this.separator();
    const config: LogConfig = {
      kind: 'log',
      input: this.upstreamSchema(),
      separator: separator === '' ? ' | ' : separator,
    };
    this.jobState.setNodeConfig(this.node().id, config);
    this.layoutService.closeModal();
  }

  onCancel() {
    this.layoutService.closeModal();
  }

  onTitleChange(value: string) {
    const trimmed = value.trim();
    this.nodeGraph.renameNode(this.node().id, trimmed || this.node().type.label);
  }
}
