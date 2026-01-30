import { Component, computed, inject, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NodeInstance } from '../../../../core/nodes-services/node.type';
import { JobStateService } from '../../../../core/nodes-services/job-state.service';
import { isDbInputConfig } from '../../../../core/nodes-services/node-configs.type';

@Component({
  selector: 'app-db-input-canvas',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './db-input-canvas.html',
  styleUrl: './db-input-canvas.css',
})
export class DbInputCanvasComponent {
  node = input.required<NodeInstance>();

  private jobState = inject(JobStateService);

  protected dbName = computed(() => {
    const config = this.jobState.getNodeConfig(this.node().id);
    if (isDbInputConfig(config)) {
      return config.connection?.database || null;
    }
    // Fallback to untyped config for backward compat
    const cfg = this.node().config as Record<string, any> | undefined;
    if (!cfg) return null;
    if (cfg['database']) return String(cfg['database']);
    return null;
  });

  protected dbTypeIcon = computed(() => {
    const config = this.jobState.getNodeConfig(this.node().id);
    if (isDbInputConfig(config)) {
      switch (config.connection?.type) {
        case 'postgres': return 'pi pi-database';
        case 'sqlserver': return 'pi pi-table';
        case 'mysql': return 'pi pi-box';
        default: return 'pi pi-database';
      }
    }
    // Fallback
    const cfg = this.node().config as Record<string, any> | undefined;
    const dbType = cfg?.['dbType'] || 'postgresql';
    switch (dbType) {
      case 'postgresql': return 'pi pi-database';
      case 'sqlserver': return 'pi pi-table';
      case 'mysql': return 'pi pi-box';
      default: return 'pi pi-database';
    }
  });

  protected query = computed(() => {
    const config = this.jobState.getNodeConfig(this.node().id);
    if (isDbInputConfig(config)) {
      return config.query || '';
    }
    const cfg = this.node().config as Record<string, any> | undefined;
    const q = cfg?.['query'];
    if (Array.isArray(q)) return q.join('\n');
    return q || '';
  });

  protected columnCount = computed(() => {
    this.jobState.schemaVersion();
    const config = this.jobState.getNodeConfig(this.node().id);
    if (isDbInputConfig(config) && config.dataModels?.length) {
      return config.dataModels.length;
    }
    return 0;
  });
}
