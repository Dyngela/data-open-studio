import { Component, computed, inject, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NodeInstance } from '../../core/nodes-services/node.type';
import { JobStateService } from '../../core/nodes-services/job-state.service';

@Component({
  selector: 'app-log-canvas',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './log-canvas.html',
  styleUrl: './log-canvas.css',
})
export class LogCanvasComponent {
  node = input.required<NodeInstance>();
  private jobState = inject(JobStateService);

  protected columnCount = computed(() => {
    this.jobState.schemaVersion();
    const schemas = this.jobState.getUpstreamSchemas(this.node().id);
    if (schemas.length === 0) return 0;
    return schemas[0].schema.length;
  });
}
