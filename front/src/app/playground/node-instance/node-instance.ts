import { Component, input, output, ElementRef, AfterViewInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NodeInstance } from '../models/node.model';

@Component({
  selector: 'app-node-instance',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './node-instance.html',
  styleUrl: './node-instance.css',
})
export class NodeInstanceComponent {
  node = input.required<NodeInstance>();
  panOffset = input({ x: 0, y: 0 });
  isPanning = input(false);
  outputPortClick = output<{ nodeId: string; portIndex: number; portType: 'data' | 'flow' }>();
  inputPortClick = output<{ nodeId: string; portIndex: number; portType: 'data' | 'flow' }>();

  constructor(private elementRef: ElementRef) {}

  onOutputPortClick(portIndex: number, portType: 'data' | 'flow', event: MouseEvent) {
    event.stopPropagation();
    this.outputPortClick.emit({ nodeId: this.node().id, portIndex, portType });
  }

  onInputPortClick(portIndex: number, portType: 'data' | 'flow', event: MouseEvent) {
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

  getDbName(): string | null {
    const cfg = (this.node().config as Record<string, any>) || {};
    if (cfg['database']) return String(cfg['database']);
    const cs = cfg['connectionString'];
    if (!cs || typeof cs !== 'string') return null;
    try {
      if (cs.includes('://')) {
        const u = new URL(cs);
        const path = (u.pathname || '').replace(/^\//, '');
        if (path) return path.split('/')[0];
        const qp = u.searchParams.get('database') || u.searchParams.get('db');
        if (qp) return qp;
      }
      const m =
        cs.match(/(?:^|;)\s*Database\s*=\s*([^;]+)/i) ||
        cs.match(/(?:^|;)\s*Initial\s*Catalog\s*=\s*([^;]+)/i) ||
        cs.match(/(?:^|;)\s*db\s*=\s*([^;]+)/i);
      if (m) return m[1].trim();
    } catch {}
    return null;
  }

  getDbTypeIcon(): string {
    const cfg = (this.node().config as Record<string, any>) || {};
    const dbType = cfg['dbType'] || 'postgresql';
    
    switch (dbType) {
      case 'postgresql':
        return 'pi pi-database';
      case 'sqlserver':
        return 'pi pi-table';
      case 'mysql':
        return 'pi pi-box';
      default:
        return 'pi pi-database';
    }
  }
  
  getDbQuery(): string {
    const query = this.node().config?.['query'];
    if (Array.isArray(query)) return query.join('\n');
    return query || '';
  }
}
