import { Component, input, output, signal, inject, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { NodeInstance } from '../../../core/services/node.type';
import { ConnectionService, DbConnection } from '../../../core/services/connection.service';
import { BaseWebSocketService } from '../../../core/services/base-ws.service';
import {DataModel, DbType} from '../../../core/api/metadata.type';

@Component({
  selector: 'app-db-input-modal',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './db-input.modal.html',
  styleUrl: './db-input.modal.css',
})
export class DbInputModal {
  node = input.required<NodeInstance>();
  close = output<void>();
  save = output<{ connectionString: string; table: string; query: string; database?: string; connectionId?: string; dbType?: DbType; host?: string; port?: string; username?: string; password?: string; sslMode?: string }>();

  private connectionService = inject(ConnectionService);
  private wsService = inject(BaseWebSocketService);

  mode = signal<'select' | 'new' | 'edit'>('select');
  selectedConnectionId = signal<string>('');

  // Schema guessing state
  isGuessingSchema = signal(false);
  guessedSchema = signal<DataModel[]>([]);
  guessError = signal<string | null>(null);

  formState = {
    connectionString: '',
    table: '',
    query: '',
    database: '',
    connectionName: '',
    host: '',
    port: '',
    username: '',
    password: '',
    sslMode: 'disable',
    dbType: DbType.Postgres,
  };

  connections = signal<DbConnection[]>([]);

  constructor() {
    // Listen for guess schema responses
  }

  ngOnInit() {
    this.connections.set(this.connectionService.getConnections());

    const cfg = this.node().config || {};
    this.formState = {
      connectionString: cfg['connectionString'] ?? '',
      table: cfg['table'] ?? '',
      query: cfg['query'] ?? '',
      database: cfg['database'] ?? '',
      connectionName: '',
      host: cfg['host'] ?? '',
      port: cfg['port'] ?? '',
      username: cfg['username'] ?? '',
      password: cfg['password'] ?? '',
      sslMode: cfg['sslMode'] ?? 'disable',
      dbType: cfg['dbType'] ?? 'postgresql',
    };

    const savedConnId = cfg['connectionId'];
    if (savedConnId) {
      this.selectedConnectionId.set(savedConnId);
      this.mode.set('select');
      const conn = this.connectionService.getConnectionById(savedConnId);
      if (conn) {
        this.formState.connectionString = conn.connectionString;
        this.formState.database = conn.database ?? '';
      }
    }
  }

  switchMode(newMode: 'select' | 'new' | 'edit') {
    this.mode.set(newMode);
    if (newMode === 'new') {
      this.formState.connectionString = '';
      this.formState.host = '';
      this.formState.port = '';
      this.formState.username = '';
      this.formState.password = '';
      this.formState.database = '';
      this.formState.sslMode = 'disable';
      this.formState.dbType = <DbType>'postgresql';
      this.formState.connectionName = '';
      this.selectedConnectionId.set('');
    } else if (newMode === 'select' && this.selectedConnectionId()) {
      const conn = this.connectionService.getConnectionById(this.selectedConnectionId());
      if (conn) {
        this.formState.connectionString = conn.connectionString;
        this.formState.database = conn.database ?? '';
        this.formState.host = conn.host ?? '';
        this.formState.port = conn.port ?? '';
        this.formState.username = conn.username ?? '';
        this.formState.password = conn.password ?? '';
        this.formState.sslMode = conn.sslMode ?? 'disable';
        this.formState.dbType = conn.type;
      }
    }
  }

  onConnectionSelected(connId: string) {
    this.selectedConnectionId.set(connId);
    const conn = this.connectionService.getConnectionById(connId);
    if (conn) {
      this.formState.connectionString = conn.connectionString;
      this.formState.database = conn.database ?? '';
      this.formState.host = conn.host ?? '';
      this.formState.port = conn.port ?? '';
      this.formState.username = conn.username ?? '';
      this.formState.password = conn.password ?? '';
      this.formState.sslMode = conn.sslMode ?? 'disable';
    }
  }

  onSave() {
    // Validation pour mode 'new'
    if (this.mode() === 'new') {
      if (!this.formState.connectionName?.trim()) {
        alert('Le nom de la connexion est obligatoire');
        return;
      }
      if (!this.formState.dbType) {
        alert('Le type de base de données est obligatoire');
        return;
      }
      if (!this.formState.connectionString && !this.formState.host) {
        alert('Remplissez soit la Connection DNS, soit les détails séparés (Host, Port, Base de données, Utilisateur, Mot de passe)');
        return;
      }
      if (this.formState.connectionString) {
        if (!this.formState.connectionString.trim()) {
          alert('La Connection DNS est obligatoire si vous utilisez l\'option 1');
          return;
        }
      } else {
        if (!this.formState.host?.trim()) {
          alert('Host est obligatoire');
          return;
        }
        if (!this.formState.port?.toString().trim()) {
          alert('Port est obligatoire');
          return;
        }
        if (!this.formState.database?.trim()) {
          alert('Base de données est obligatoire');
          return;
        }
        if (!this.formState.username?.trim()) {
          alert('Utilisateur est obligatoire');
          return;
        }
        if (!this.formState.password?.toString().trim()) {
          alert('Mot de passe est obligatoire');
          return;
        }
      }
    }

    if (this.mode() === 'new') {
      const newConn = this.connectionService.addConnection(
        this.formState.connectionName || 'Custom Connection',
        this.formState.dbType,
        this.formState.connectionString,
        this.formState.database,
        this.formState.host,
        this.formState.port,
        this.formState.username,
        this.formState.password,
        this.formState.sslMode
      );
      this.save.emit({
        connectionString: this.formState.connectionString,
        table: this.formState.table,
        query: this.formState.query,
        database: this.formState.database,
        connectionId: newConn.id,
        dbType: this.formState.dbType,
        host: this.formState.host,
        port: this.formState.port,
        username: this.formState.username,
        password: this.formState.password,
        sslMode: this.formState.sslMode,
      });
    } else {
      this.save.emit({
        connectionString: this.formState.connectionString,
        table: this.formState.table,
        query: this.formState.query,
        database: this.formState.database,
        connectionId: this.selectedConnectionId(),
        dbType: this.formState.dbType,
        host: this.formState.host,
        port: this.formState.port,
        username: this.formState.username,
        password: this.formState.password,
        sslMode: this.formState.sslMode,
      });
    }
  }

  onCancel() {
    this.close.emit();
  }

  getConnectionDbIcon(dbType: string): string {
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

  onSaveExistingConnection() {
    const connId = this.selectedConnectionId();
    if (connId) {
      this.connectionService.updateConnection(
        connId,
        this.formState.connectionName,
        this.formState.dbType,
        this.formState.connectionString,
        this.formState.database,
        this.formState.host,
        this.formState.port,
        this.formState.username,
        this.formState.password,
        this.formState.sslMode
      );
      this.switchMode('select');
    }
  }

  /**
   * Guess the schema/data model from the query
   */
  guessSchema() {
    if (!this.formState.query?.trim()) {
      this.guessError.set('Please enter a query first');
      return;
    }


  }
}
