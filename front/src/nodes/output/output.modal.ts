import { Component, input, signal, inject, OnInit, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormControl, FormsModule } from '@angular/forms';
import { NodeInstance } from '../../core/nodes-services/node.type';
import { DataModel, DbMetadata, DbType } from '../../core/api/metadata.type';
import { MetadataLocalService } from '../../core/services/metadata.local.service';
import { SqlService } from '../../core/api/sql.service';
import { DatabaseColumn } from '../../core/api/sql.type';
import { DbOutputMode, OutputNodeConfig, isOutputConfig } from './definition';
import { KuiSelect, SelectOption } from '../../ui/form/select/kui-select/kui-select';
import { JobStateService } from '../../core/nodes-services/job-state.service';
import { LayoutService } from '../../core/services/layout-service';

@Component({
  selector: 'app-output-modal',
  standalone: true,
  imports: [CommonModule, FormsModule, ReactiveFormsModule, KuiSelect],
  templateUrl: './output.modal.html',
  styleUrl: './output.modal.css',
})
export class OutputModal implements OnInit {
  private metadata = inject(MetadataLocalService);
  private sqlService = inject(SqlService);
  private layoutService = inject(LayoutService);
  private jobState = inject(JobStateService);

  node = input.required<NodeInstance>();

  connectionsOptions = computed<SelectOption[]>(() => {
    return this.metadata.db.data()?.map(conn => ({
      label: `${conn.databaseName} (${conn.host}:${conn.port})`,
      value: conn.id
    })) ?? [];
  });

  connectionControl = new FormControl<number | null>(null);
  tableControl = new FormControl<string | null>(null);

  selectedConnection = signal<DbMetadata | null>(null);
  selectedMode = signal<DbOutputMode>('insert');
  selectedTable = signal<string>('');
  batchSize = signal<number>(500);
  keyColumns = signal<Set<string>>(new Set());

  // Introspection state
  tables = signal<{ schema: string; name: string }[]>([]);
  columns = signal<DatabaseColumn[]>([]);
  isLoadingTables = signal(false);
  isLoadingColumns = signal(false);

  tableOptions = computed<SelectOption[]>(() => {
    return this.tables().map(t => ({
      label: t.schema ? `${t.schema}.${t.name}` : t.name,
      value: t.schema ? `${t.schema}.${t.name}` : t.name,
    }));
  });

  needsKeyColumns = computed(() => {
    const m = this.selectedMode();
    return m === 'update' || m === 'merge' || m === 'delete';
  });

  readonly modes: { value: DbOutputMode; label: string; icon: string }[] = [
    { value: 'insert', label: 'Insert', icon: 'pi pi-plus' },
    { value: 'update', label: 'Update', icon: 'pi pi-pencil' },
    { value: 'merge', label: 'Merge', icon: 'pi pi-sync' },
    { value: 'delete', label: 'Delete', icon: 'pi pi-trash' },
    { value: 'truncate', label: 'Truncate', icon: 'pi pi-eraser' },
  ];

  ngOnInit() {
    const cfg = this.node().config;
    if (cfg && typeof cfg === 'object' && 'kind' in cfg && isOutputConfig(cfg as any)) {
      const typed = cfg as OutputNodeConfig;
      this.selectedMode.set(typed.mode ?? 'insert');
      this.selectedTable.set(typed.table ?? '');
      this.batchSize.set(typed.batchSize ?? 500);
      this.keyColumns.set(new Set(typed.keyColumns ?? []));

      if (typed.connectionId) {
        const connId = Number(typed.connectionId);
        this.connectionControl.setValue(connId);
        const conn = this.metadata.db.data()?.find(c => c.id === connId) ?? null;
        this.selectedConnection.set(conn);

        if (conn) {
          this.loadTables(connId);
          if (typed.table) {
            const fullTable = typed.dbSchema ? `${typed.dbSchema}.${typed.table}` : typed.table;
            this.tableControl.setValue(fullTable);
            this.loadColumns(connId, typed.table);
          }
        }
      }

      if (typed.dataModels?.length) {
        // Rebuild columns from saved dataModels for display
        this.columns.set(typed.dataModels.map(dm => ({
          name: dm.name,
          dataType: dm.type,
          isNullable: dm.nullable,
          isPrimary: typed.keyColumns?.includes(dm.name) ?? false,
        })));
      }
    }
  }

  onConnectionSelected(connId: number) {
    const conn = this.metadata.db.data()?.find(c => c.id === connId) ?? null;
    this.selectedConnection.set(conn);
    this.selectedTable.set('');
    this.columns.set([]);
    this.keyColumns.set(new Set());

    if (conn) {
      this.loadTables(connId);
    }
  }

  onTableSelected(tableFullName: string) {
    this.selectedTable.set(tableFullName);
    this.columns.set([]);
    this.keyColumns.set(new Set());

    const conn = this.selectedConnection();
    if (conn) {
      // Extract just the table name (strip schema if present)
      const parts = tableFullName.split('.');
      const tableName = parts.length > 1 ? parts[parts.length - 1] : tableFullName;
      this.loadColumns(conn.id, tableName);
    }
  }

  onModeSelected(mode: DbOutputMode) {
    this.selectedMode.set(mode);
  }

  toggleKeyColumn(colName: string) {
    const current = new Set(this.keyColumns());
    if (current.has(colName)) {
      current.delete(colName);
    } else {
      current.add(colName);
    }
    this.keyColumns.set(current);
  }

  isKeyColumn(colName: string): boolean {
    return this.keyColumns().has(colName);
  }

  getConnectionDbIcon(conn: DbMetadata): string {
    if (conn.port === 1433) return 'pi pi-table';
    if (conn.port === 3306) return 'pi pi-box';
    return 'pi pi-database';
  }

  onSave() {
    const conn = this.selectedConnection();
    if (!conn || !this.selectedTable()) return;

    const tableFullName = this.selectedTable();
    const parts = tableFullName.split('.');
    const dbSchema = parts.length > 1 ? parts[0] : '';
    const tableName = parts.length > 1 ? parts[1] : parts[0];

    const dataModels: DataModel[] = this.columns().map(col => ({
      name: col.name,
      type: col.dataType,
      goType: '',
      nullable: col.isNullable,
    }));

    const config: OutputNodeConfig = {
      kind: 'output',
      table: tableName,
      mode: this.selectedMode(),
      batchSize: this.batchSize(),
      dbSchema,
      connection: {
        type: conn.databaseType || DbType.Postgres,
        host: conn.host,
        port: conn.port,
        database: conn.databaseName,
        username: conn.user,
        password: conn.password,
        sslMode: conn.sslMode || 'disable',
      },
      connectionId: String(conn.id),
      dataModels,
      keyColumns: Array.from(this.keyColumns()),
    };

    this.jobState.setNodeConfig(this.node().id, config);
    this.layoutService.closeModal();
  }

  onCancel() {
    this.layoutService.closeModal();
  }

  private loadTables(connectionId: number) {
    this.isLoadingTables.set(true);
    const mutation = this.sqlService.getTables(
      (response) => {
        this.tables.set(response.tables ?? []);
        this.isLoadingTables.set(false);
      },
      () => {
        this.tables.set([]);
        this.isLoadingTables.set(false);
      },
    );
    mutation.execute({ metadataDatabaseId: connectionId });
  }

  private loadColumns(connectionId: number, tableName: string) {
    this.isLoadingColumns.set(true);
    const mutation = this.sqlService.getColumns(
      (response) => {
        const cols = response.columns ?? [];
        this.columns.set(cols);
        // Auto-select primary keys as key columns if none selected yet
        if (this.keyColumns().size === 0) {
          const primaryKeys = new Set(cols.filter(c => c.isPrimary).map(c => c.name));
          this.keyColumns.set(primaryKeys);
        }
        this.isLoadingColumns.set(false);
      },
      () => {
        this.columns.set([]);
        this.isLoadingColumns.set(false);
      },
    );
    mutation.execute({ metadataDatabaseId: connectionId, tableName });
  }
}
