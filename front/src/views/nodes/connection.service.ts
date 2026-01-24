import { Injectable } from '@angular/core';
import { signal } from '@angular/core';

export type DbType = 'postgresql' | 'sqlserver' | 'mysql';

export interface DbConnection {
  id: string;
  name: string;
  type: DbType;
  connectionString: string;
  database?: string;
  host?: string;
  port?: string;
  username?: string;
  password?: string;
  sslMode?: string;
  createdAt: Date;
}

@Injectable({
  providedIn: 'root',
})
export class ConnectionService {
  private connections = signal<DbConnection[]>([]);

  getConnections() {
    return this.connections();
  }

  getConnectionById(id: string): DbConnection | undefined {
    return this.connections().find((c) => c.id === id);
  }

  addConnection(name: string, type: DbType, connectionString: string, database?: string, host?: string, port?: string, username?: string, password?: string, sslMode?: string): DbConnection {
    const id = `conn-${Date.now()}`;
    const newConnection: DbConnection = {
      id,
      name,
      type,
      connectionString,
      database,
      host,
      port,
      username,
      password,
      sslMode,
      createdAt: new Date(),
    };
    this.connections.update((conns) => [...conns, newConnection]);
    return newConnection;
  }

  deleteConnection(id: string) {
    if (id === 'default') return; // Ne pas supprimer la connexion par défaut
    this.connections.update((conns) => conns.filter((c) => c.id !== id));
  }

  updateConnection(id: string, name: string, type: DbType, connectionString: string, database?: string, host?: string, port?: string, username?: string, password?: string, sslMode?: string) {
    this.connections.update((conns) =>
      conns.map((c) =>
        c.id === id
          ? {
              ...c,
              name,
              type,
              connectionString,
              database,
              host,
              port,
              username,
              password,
              sslMode,
            }
          : c
      )
    );
  }
}
