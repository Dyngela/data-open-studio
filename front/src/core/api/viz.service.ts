import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { inject, Injectable, signal } from '@angular/core';
import { catchError, finalize, Observable, of } from 'rxjs';
import { MessageService } from 'primeng/api';
import { environment } from '../../environments/environment';
import { ApiError, ApiMutation, ApiResult, MutationSuccess, SearchCriteria } from '../services/base-api.type';
import {
  CreateWorkspaceRequest,
  UpdateWorkspaceRequest,
  CreateSourceRequest,
  ExecuteRequest,
  ExecuteResponse,
  FrameData,
  FrameListResponse,
  LoadSourceResponse,
  Source,
  SourceListResponse,
  Workspace,
  WorkspaceListResponse,
} from './viz.type';

@Injectable({ providedIn: 'root' })
export class VizService {
  private http = inject(HttpClient);
  private messageService = inject(MessageService);
  private readonly baseUrl = environment.vizBaseUrl;

  // ---------------------------------------------------------------------------
  // Workspaces
  // ---------------------------------------------------------------------------

  listWorkspaces(onSuccess?: (data: WorkspaceListResponse) => void): ApiResult<WorkspaceListResponse> {
    return this.fetch<WorkspaceListResponse>('/workspaces', undefined, onSuccess);
  }

  getWorkspace(id: string, onSuccess?: (data: Workspace) => void): ApiResult<Workspace> {
    return this.fetch<Workspace>(`/workspaces/${id}`, undefined, onSuccess);
  }

  createWorkspace(
    onSuccess?: (data: Workspace) => void,
    onError?: (error: ApiError) => void,
  ): ApiMutation<Workspace, CreateWorkspaceRequest> {
    return this.mutate<Workspace, CreateWorkspaceRequest>('POST', '/workspaces', onSuccess, onError);
  }

  updateWorkspace(
    id: string,
    onSuccess?: (data: Workspace) => void,
    onError?: (error: ApiError) => void,
  ): ApiMutation<Workspace, UpdateWorkspaceRequest> {
    return this.mutate<Workspace, UpdateWorkspaceRequest>('PATCH', `/workspaces/${id}`, onSuccess, onError);
  }

  deleteWorkspace(
    id: string,
    onSuccess?: () => void,
    onError?: (error: ApiError) => void,
  ): ApiMutation<null, void> {
    return this.mutate<null, void>('DELETE', `/workspaces/${id}`, onSuccess as any, onError);
  }

  // ---------------------------------------------------------------------------
  // Sources
  // ---------------------------------------------------------------------------

  listSources(workspaceId: string, onSuccess?: (data: SourceListResponse) => void): ApiResult<SourceListResponse> {
    return this.fetch<SourceListResponse>(`/workspaces/${workspaceId}/sources`, undefined, onSuccess);
  }

  createSource(
    workspaceId: string,
    onSuccess?: (data: Source) => void,
    onError?: (error: ApiError) => void,
  ): ApiMutation<Source, CreateSourceRequest> {
    return this.mutate<Source, CreateSourceRequest>('POST', `/workspaces/${workspaceId}/sources`, onSuccess, onError);
  }

  deleteSource(
    workspaceId: string,
    sourceId: string,
    onSuccess?: () => void,
    onError?: (error: ApiError) => void,
  ): ApiMutation<null, void> {
    return this.mutate<null, void>('DELETE', `/workspaces/${workspaceId}/sources/${sourceId}`, onSuccess as any, onError);
  }

  loadSource(
    workspaceId: string,
    sourceId: string,
    onSuccess?: (data: LoadSourceResponse) => void,
    onError?: (error: ApiError) => void,
  ): ApiMutation<LoadSourceResponse, void> {
    return this.mutate<LoadSourceResponse, void>('POST', `/workspaces/${workspaceId}/sources/${sourceId}/load`, onSuccess, onError);
  }

  // ---------------------------------------------------------------------------
  // Frames
  // ---------------------------------------------------------------------------

  listFrames(workspaceId: string, onSuccess?: (data: FrameListResponse) => void): ApiResult<FrameListResponse> {
    return this.fetch<FrameListResponse>(`/workspaces/${workspaceId}/frames`, undefined, onSuccess);
  }

  deleteFrame(
    workspaceId: string,
    frameName: string,
    onSuccess?: () => void,
    onError?: (error: ApiError) => void,
  ): ApiMutation<null, void> {
    return this.mutate<null, void>('DELETE', `/workspaces/${workspaceId}/frames/${encodeURIComponent(frameName)}`, onSuccess as any, onError);
  }

  previewFrame(
    workspaceId: string,
    frameName: string,
    offset = 0,
    limit = 100,
    onSuccess?: (data: FrameData) => void,
  ): ApiResult<FrameData> {
    return this.fetch<FrameData>(
      `/workspaces/${workspaceId}/frames/${encodeURIComponent(frameName)}/preview`,
      [{ name: 'offset', value: offset }, { name: 'limit', value: limit }],
      onSuccess,
    );
  }

  // ---------------------------------------------------------------------------
  // Execute
  // ---------------------------------------------------------------------------

  execute(
    workspaceId: string,
    onSuccess?: (data: ExecuteResponse) => void,
    onError?: (error: ApiError) => void,
  ): ApiMutation<ExecuteResponse, ExecuteRequest> {
    return this.mutate<ExecuteResponse, ExecuteRequest>('POST', `/workspaces/${workspaceId}/execute`, onSuccess, onError);
  }

  // ---------------------------------------------------------------------------
  // Private HTTP helpers
  // ---------------------------------------------------------------------------

  private fetch<T>(
    path: string,
    criteria?: SearchCriteria[],
    onSuccess?: (data: T) => void,
    onError?: (error: ApiError) => void,
  ): ApiResult<T> {
    const params = new HttpParams({
      fromObject: Object.fromEntries(criteria?.map(c => [c.name, String(c.value)]) ?? []),
    });

    const data = signal<T | null>(null);
    const error = signal<ApiError | null>(null);
    const isLoading = signal(true);

    const run = () => {
      isLoading.set(true);
      this.http.get<T>(`${this.baseUrl}${path}`, { params }).pipe(
        catchError(err => {
          const apiError = this.toApiError(err);
          error.set(apiError);
          if (onError) onError(apiError);
          else this.showError(apiError, path);
          return of(null);
        }),
        finalize(() => isLoading.set(false)),
      ).subscribe(res => {
        if (res !== null) {
          data.set(res);
          if (onSuccess) onSuccess(res);
        }
      });
    };

    run();
    return { data, error, isLoading, refresh: run };
  }

  private mutate<TResponse, TBody = void>(
    method: 'POST' | 'PUT' | 'PATCH' | 'DELETE',
    path: string,
    onSuccess?: (data: TResponse) => void,
    onError?: (error: ApiError) => void,
  ): ApiMutation<TResponse, TBody> {
    const data = signal<TResponse | null>(null);
    const isLoading = signal(false);
    const error = signal<ApiError | null>(null);
    const success = signal<MutationSuccess<TResponse> | null>(null);

    return {
      data,
      isLoading,
      error,
      success,
      reset: () => { data.set(null); error.set(null); success.set(null); },
      execute: ((body?: TBody) => {
        isLoading.set(true);
        error.set(null);

        this.http.request<TResponse>(method, `${this.baseUrl}${path}`, {
          body,
          headers: new HttpHeaders({ 'Content-Type': 'application/json' }),
          observe: 'body',
          responseType: 'json',
        }).pipe(
          catchError(err => {
            const apiError = this.toApiError(err);
            error.set(apiError);
            if (onError) onError(apiError);
            else this.showError(apiError, path);
            return of(null);
          }),
          finalize(() => isLoading.set(false)),
        ).subscribe(res => {
          if (res !== null) {
            data.set(res);
            success.set({ data: res, timestamp: Date.now() });
            if (onSuccess) onSuccess(res);
          }
        });
      }) as TBody extends void ? () => void : (body: TBody) => void,
    };
  }

  private toApiError(err: any): ApiError {
    return {
      status: err.status,
      // api API returns { error: "..." }, pipeline API returns { message: "..." }
      message: err.error?.error ?? err.error?.message ?? 'Unknown error',
      raw: err,
    };
  }

  private showError(error: ApiError, path: string): void {
    console.error(`Viz API error on ${path}`, error);
    this.messageService.add({ severity: 'error', summary: 'Error', detail: error.message });
  }
}
