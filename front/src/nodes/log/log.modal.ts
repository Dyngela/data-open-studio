import { Component, computed, inject, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NodeInstance } from '../../core/nodes-services/node.type';
import { JobStateService } from '../../core/nodes-services/job-state.service';
import { LogConfig } from './definition';
import {LayoutService} from '../../core/services/layout-service';

@Component({
  selector: 'app-log-modal',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './log.modal.html',
  styleUrl: './log.modal.css',
})
export class LogModal {
  private jobState = inject(JobStateService);
  private layoutService = inject(LayoutService);
  node = input.required<NodeInstance>();

  upstreamSchema = computed(() => {
    const schemas = this.jobState.getUpstreamSchemas(this.node().id);
    if (schemas.length === 0) return [];
    return schemas[0].schema;
  });

  onSave() {
    const config: LogConfig = {
      kind: 'log',
      input: this.upstreamSchema(),
    };
    this.jobState.setNodeConfig(this.node().id, config);
    this.layoutService.closeModal();
  }

  onCancel() {
    this.layoutService.closeModal();
  }
}
