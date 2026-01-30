import { Component, computed, inject, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NodeInstance } from '../../core/nodes-services/node.type';
import { JobStateService } from '../../core/nodes-services/job-state.service';
import { isMapConfig } from './definition';

@Component({
  selector: 'app-map-canvas',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './map-canvas.html',
  styleUrl: './map-canvas.css',
})
export class MapCanvasComponent {
  node = input.required<NodeInstance>();

  private jobState = inject(JobStateService);

  protected inputCount = computed(() => {
    this.jobState.schemaVersion();
    const upstream = this.jobState.getUpstreamSchemas(this.node().id);
    return upstream.length;
  });

  protected outputColumnCount = computed(() => {
    this.jobState.schemaVersion();
    const config = this.jobState.getNodeConfig(this.node().id);
    if (isMapConfig(config)) {
      return config.outputs.reduce((sum, out) => sum + out.columns.length, 0);
    }
    return 0;
  });

  protected joinType = computed(() => {
    const config = this.jobState.getNodeConfig(this.node().id);
    if (isMapConfig(config) && config.join) {
      return config.join.type;
    }
    return null;
  });
}
