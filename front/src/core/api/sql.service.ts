import {inject, Injectable} from '@angular/core';
import {BaseApiService} from '../services/base-api.service';
import {ApiMutation} from '../services/base-api.type';
import {
  GuessQueryRequest,
  GuessQueryResponse,
  OptimizeQueryRequest,
  OptimizeQueryResponse,
  TestConnectionRequest,
  TestConnectionResult,
  IntrospectDatabaseRequest,
  DatabaseIntrospection,
} from './sql.type';

@Injectable({
  providedIn: 'root'
})
export class SqlService {
  private api = inject(BaseApiService);

  guessQuery(
    onSuccess?: (data: GuessQueryResponse) => void,
    onError?: (error: any) => void
  ): ApiMutation<GuessQueryResponse, GuessQueryRequest> {
    return this.api.post<GuessQueryResponse, GuessQueryRequest>(
      '/sql/guess-query',
      onSuccess,
      onError
    );
  }

  optimizeQuery(
    onSuccess?: (data: OptimizeQueryResponse) => void,
    onError?: (error: any) => void
  ): ApiMutation<OptimizeQueryResponse, OptimizeQueryRequest> {
    return this.api.post<OptimizeQueryResponse, OptimizeQueryRequest>(
      '/sql/optimize-query',
      onSuccess,
      onError
    );
  }

  testConnection(
    onSuccess?: (data: TestConnectionResult) => void,
    onError?: (error: any) => void
  ): ApiMutation<TestConnectionResult, TestConnectionRequest> {
    return this.api.post<TestConnectionResult, TestConnectionRequest>(
      '/sql/introspect/test-connection',
      onSuccess,
      onError
    );
  }

  getTables(
    onSuccess?: (data: DatabaseIntrospection) => void,
    onError?: (error: any) => void
  ): ApiMutation<DatabaseIntrospection, IntrospectDatabaseRequest> {
    return this.api.post<DatabaseIntrospection, IntrospectDatabaseRequest>(
      '/sql/introspect/tables',
      onSuccess,
      onError
    );
  }

  getColumns(
    onSuccess?: (data: DatabaseIntrospection) => void,
    onError?: (error: any) => void
  ): ApiMutation<DatabaseIntrospection, IntrospectDatabaseRequest> {
    return this.api.post<DatabaseIntrospection, IntrospectDatabaseRequest>(
      '/sql/introspect/columns',
      onSuccess,
      onError
    );
  }
}
