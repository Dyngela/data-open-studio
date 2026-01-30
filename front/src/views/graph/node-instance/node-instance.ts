import { Component, computed, inject, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NodeInstance, PortType } from '../../../core/nodes-services/node.type';
import { JobStateService } from '../../../core/nodes-services/job-state.service';
import { isDbInputConfig } from '../../../nodes/db-input/definition';
import { DbInputCanvasComponent } from '../../../nodes/db-input/db-input-canvas';
import { MapCanvasComponent } from '../../../nodes/transform/map-canvas';
import { LogCanvasComponent } from '../../../nodes/log/log-canvas';

@Component({
  selector: 'app-node-instance',
  standalone: true,
  imports: [CommonModule, DbInputCanvasComponent, MapCanvasComponent, LogCanvasComponent],
  templateUrl: './node-instance.html',
  styleUrl: './node-instance.css',
})
export class NodeInstanceComponent {
  node = input.required<NodeInstance>();
  panOffset = input({ x: 0, y: 0 });
  isPanning = input(false);
  outputPortClick = output<{ nodeId: number; portIndex: number; portType: PortType }>();
  inputPortClick = output<{ nodeId: number; portIndex: number; portType: PortType }>();

  private jobState = inject(JobStateService);

  onOutputPortClick(portIndex: number, portType: PortType, event: MouseEvent) {
    event.stopPropagation();
    this.outputPortClick.emit({ nodeId: this.node().id, portIndex, portType });
  }

  onInputPortClick(portIndex: number, portType: PortType, event: MouseEvent) {
    event.stopPropagation();
    this.inputPortClick.emit({ nodeId: this.node().id, portIndex, portType });
  }

  getDataInputPorts(): number[] {
    return this.node().type.hasDataInput ? [0] : [];
  }

  getDataOutputPorts(): number[] {
    return this.node().type.hasDataOutput ? [0] : [];
  }

  getFlowInputPorts(): number[] {
    return this.node().type.hasFlowInput ? [0] : [];
  }

  getFlowOutputPorts(): number[] {
    return this.node().type.hasFlowOutput ? [0] : [];
  }

  /** Header icon — typed config aware for db-input */
  protected headerIcon = computed(() => {
    const n = this.node();
    if (n.type.id === 'db-input') {
      const config = this.jobState.getNodeConfig(n.id);
      if (isDbInputConfig(config)) {
        switch (config.connection?.type) {
          case 'postgres': return 'pi pi-database';
          case 'sqlserver': return 'pi pi-table';
          case 'mysql': return 'pi pi-box';
        }
      }
      // Fallback to untyped
      const cfg = n.config as Record<string, any> | undefined;
      const dbType = cfg?.['dbType'] || 'postgresql';
      switch (dbType) {
        case 'postgresql': return 'pi pi-database';
        case 'sqlserver': return 'pi pi-table';
        case 'mysql': return 'pi pi-box';
        default: return 'pi pi-database';
      }
    }
    return n.type.icon || 'pi pi-box';
  });

  /** Header DB badge — typed config aware */
  protected dbName = computed(() => {
    const n = this.node();
    if (n.type.id !== 'db-input') return null;
    const config = this.jobState.getNodeConfig(n.id);
    if (isDbInputConfig(config)) {
      return config.connection?.database || null;
    }
    const cfg = n.config as Record<string, any> | undefined;
    return cfg?.['database'] ? String(cfg['database']) : null;
  });

  protected readonly PortType = PortType;
}
