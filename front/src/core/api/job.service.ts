import { BaseWebSocketService, WsError, WsMutation } from './base-ws.service';
import { inject, Injectable } from '@angular/core';
import { Job } from './job.model';
import {
  DbNodeGuessDataModelRequest,
  DbNodeGuessDataModelResponse,
  DbType,
  MessageType,
} from './ws.types';

@Injectable({ providedIn: 'root' })
export class JobsWsService {
  private ws = inject(BaseWebSocketService);

  /**
   * Create a mutation to guess data model from a DB query
   * Returns signals for data, isLoading, error, success
   *
   * Usage:
   * ```ts
   * const guessDataModel = jobsWsService.guessDataModel(
   *   (response) => console.log('Data models:', response.dataModels),
   *   (error) => console.error('Error:', error)
   * );
   *
   * // In template: guessDataModel.isLoading(), guessDataModel.data()
   *
   * // Execute the request
   * guessDataModel.execute({
   *   nodeId: 1,
   *   jobId: 123,
   *   query: 'SELECT * FROM employees',
   *   dbType: DbType.Postgres,
   *   ...
   * });
   * ```
   */
  guessDataModel(
    onSuccess?: (data: DbNodeGuessDataModelResponse) => void,
    onError?: (error: WsError) => void
  ): WsMutation<DbNodeGuessDataModelResponse, DbNodeGuessDataModelRequest> {
    return this.ws.mutation<DbNodeGuessDataModelResponse, DbNodeGuessDataModelRequest>(
      MessageType.DbNodeGuessDataModel,
      MessageType.DbNodeGuessDataModelResponse,
      onSuccess,
      onError,
      { timeout: 60000 } // 60s for potentially slow queries
    );
  }

  /**
   * Create a mutation with pre-filled connection details
   * Useful when you have a saved DB metadata and just need to provide the query
   */
  guessDataModelFromConnection(
    connection: {
      dbType: DbType;
      host: string;
      port: number;
      database: string;
      username: string;
      password: string;
      dbSchema?: string;
      sslMode?: string;
    },
    onSuccess?: (data: DbNodeGuessDataModelResponse) => void,
    onError?: (error: WsError) => void
  ): WsMutation<DbNodeGuessDataModelResponse, { nodeId: number; jobId: number; query: string }> {
    const mutation = this.ws.mutation<DbNodeGuessDataModelResponse, DbNodeGuessDataModelRequest>(
      MessageType.DbNodeGuessDataModel,
      MessageType.DbNodeGuessDataModelResponse,
      onSuccess,
      onError,
      { timeout: 60000 }
    );

    // Wrap execute to merge connection details
    const originalExecute = mutation.execute;

    return {
      ...mutation,
      execute: (params: { nodeId: number; jobId: number; query: string }) => {
        const fullRequest: DbNodeGuessDataModelRequest = {
          nodeId: params.nodeId,
          jobId: params.jobId,
          query: params.query,
          dbType: connection.dbType,
          dbSchema: connection.dbSchema ?? "",
          host: connection.host,
          port: connection.port,
          database: connection.database,
          username: connection.username,
          password: connection.password,
          sslMode: connection.sslMode ?? 'disable',
        };
        originalExecute(fullRequest);
      }
    };
  }
}
