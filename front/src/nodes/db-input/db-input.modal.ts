import {Component, input, output, signal, inject, OnInit, computed, Signal} from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormControl, FormGroup } from '@angular/forms';
import { NodeInstance } from '../../core/nodes-services/node.type';
import { DataModel, DbMetadata, DbType } from '../../core/api/metadata.type';
import { MetadataLocalService } from '../../core/services/metadata.local.service';
import { DbNodeService } from '../../core/api/db-node.service';
import { DbInputNodeConfig, isDbInputConfig } from './definition';
import {KuiSelect, SelectOption} from '../../ui/form/select/kui-select/kui-select';
import {JobStateService} from '../../core/nodes-services/job-state.service';
import {LayoutService} from '../../core/services/layout-service';

@Component({
  selector: 'app-db-input-modal',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, KuiSelect],
  templateUrl: './db-input.modal.html',
  styleUrl: './db-input.modal.css',
})
export class DbInputModal implements OnInit {
  private metadata = inject(MetadataLocalService);
  private dbNodeService = inject(DbNodeService);
  private layoutService = inject(LayoutService);
  private jobState = inject(JobStateService);

  node = input.required<NodeInstance>();
  connectionsOptions = computed<SelectOption[]>(() => {
    return this.metadata.db.data()?.map(conn => ({
      label: `${conn.databaseName} (${conn.host}:${conn.port})`,
      value: conn.id
    })) ?? [];
  });

  form = new FormGroup({
    connectionId: new FormControl<number | null>(null),
    query: new FormControl(''),
  });

  selectedConnection = signal<DbMetadata | null>(null);

  // Schema state
  isGuessingSchema = signal(false);
  guessedSchema = signal<DataModel[]>([]);
  guessError = signal<string | null>(null);

  ngOnInit() {
    const cfg = this.node().config;
    console.log(cfg)
    if (cfg && typeof cfg === 'object' && 'kind' in cfg && isDbInputConfig(cfg as any)) {
      const typed = cfg as DbInputNodeConfig;
      this.form.patchValue({ query: typed.query ?? '' });

      if (typed.connectionId) {
        console.log(typed.connectionId)
        const connId = Number(typed.connectionId);
        this.form.patchValue({ connectionId: connId });
        const conn = this.metadata.db.data()?.find(c => c.id === connId) ?? null;
        console.log(conn)
        this.selectedConnection.set(conn);
      }

      if (typed.dataModels?.length) {
        this.guessedSchema.set(typed.dataModels);
      }
      return;
    }
  }

  onConnectionSelected(connId: number) {
    this.form.patchValue({ connectionId: connId });
    const conn = this.metadata.db.data()?.find(c => c.id === connId) ?? null;
    this.selectedConnection.set(conn);
  }

  getConnectionDbIcon(conn: DbMetadata): string {
    // Port-based heuristic since DbMetadata doesn't store dbType
    if (conn.port === 1433) return 'pi pi-table';
    if (conn.port === 3306) return 'pi pi-box';
    return 'pi pi-database';
  }

  guessSchema() {
    const query = this.form.value.query?.trim();
    if (!query) {
      this.guessError.set('Entrez une requête SQL d\'abord');
      return;
    }

    const conn = this.selectedConnection();
    if (!conn) {
      this.guessError.set('Sélectionnez une connexion d\'abord');
      return;
    }

    this.isGuessingSchema.set(true);
    this.guessError.set(null);

    const mutation = this.dbNodeService.guessSchema(
      (response) => {
        this.guessedSchema.set(response.dataModels || []);
        this.isGuessingSchema.set(false);
      },
      (error) => {
        this.guessError.set(error?.message || 'Impossible de détecter le schéma');
        this.isGuessingSchema.set(false);
      },
    );

    mutation.execute({
      nodeId: String(this.node().id),
      query,
      connectionId: conn.id,
    });
  }

  onSave() {
    const conn = this.selectedConnection();
    if (!conn) return;

    const config: DbInputNodeConfig = {
      kind: 'db-input',
      query: this.form.value.query ?? '',
      connectionId: String(conn.id),
      connection: {
        type: conn.databaseType || DbType.Postgres,
        host: conn.host,
        port: conn.port,
        database: conn.databaseName,
        username: conn.user,
        password: conn.password,
        sslMode: conn.sslMode || 'disable',
      },
      dataModels: this.guessedSchema(),
    };
    this.jobState.setNodeConfig(this.node().id, config);
    this.layoutService.closeModal();
  }

  onCancel() {
    this.layoutService.closeModal()
  }
}
