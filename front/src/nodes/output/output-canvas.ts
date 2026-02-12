import { Component, computed, inject, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NodeInstance } from '../../core/nodes-services/node.type';
import { JobStateService } from '../../core/nodes-services/job-state.service';
import { isOutputConfig } from './definition';

@Component({
  selector: 'app-output-canvas',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './output-canvas.html',
  styleUrl: './output-canvas.css',
})
export class OutputCanvasComponent {
  node = input.required<NodeInstance>();

  private jobState = inject(JobStateService);

  protected mode = computed(() => {
    this.jobState.schemaVersion();
    const config = this.jobState.getNodeConfig(this.node().id);
    if (isOutputConfig(config)) {
      return config.mode || null;
    }
    return null;
  });

  protected tableName = computed(() => {
    this.jobState.schemaVersion();
    const config = this.jobState.getNodeConfig(this.node().id);
    if (isOutputConfig(config)) {
      return config.table || null;
    }
    return null;
  });

  protected columnCount = computed(() => {
    this.jobState.schemaVersion();
    const config = this.jobState.getNodeConfig(this.node().id);
    if (isOutputConfig(config) && config.dataModels?.length) {
      return config.dataModels.length;
    }
    return 0;
  });
}
