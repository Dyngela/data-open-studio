import {HttpClient, HttpHeaders, HttpParams} from '@angular/common/http';
import {inject, Injectable, signal, Signal} from '@angular/core';
import {finalize, Observable, of} from 'rxjs';
import {catchError} from 'rxjs/operators';
import {environment} from '../../environments/environment';
import {MessageService} from 'primeng/api';
import {ApiError, ApiMutation, ApiOptions, ApiResult, MutationSuccess, SearchCriteria} from './base-api.type';

@Injectable({
  providedIn: 'root'
})
export class BaseApiService {
  protected http = inject(HttpClient);
  protected messageService = inject(MessageService);

  private defaultOptions = {
    get: { showSuccessMessage: false, showErrorMessage: true },
    post: { showSuccessMessage: false, showErrorMessage: true },
    put: { showSuccessMessage: false, showErrorMessage: true },
    delete: { showSuccessMessage: false, showErrorMessage: true },
  };

  public request$<T>(
    method: 'POST' | 'PUT' | 'DELETE',
    path: string,
    body?: any
  ): Observable<T> {
    return this.http.request<T>(method, `${environment.baseUrl}${path}`, {
      body,
      headers: new HttpHeaders({ 'Content-Type': 'application/json' }),
      observe: 'body',
      responseType: 'json'
    });
  }

  private execute<T>(
    method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH',
    path: string,
    request$: () => Observable<T>,
    options: ApiOptions
  ): ApiResult<T> {
    const data = signal<T | null>(null);
    const error = signal<ApiError | null>(null);
    const isLoading = signal(true);

    const run = () => {
      isLoading.set(true);
      request$()
        .pipe(
          catchError(err => {
            const apiError: ApiError = {
              status: err.status,
              message: err.error?.message ?? 'Erreur inconnue',
              raw: err
            };

            error.set(apiError);

            if (options.showErrorMessage !== false) {
              this.handleError(apiError, path);
            }

            return of(null);
          }),
          finalize(() => isLoading.set(false))
        )
        .subscribe(res => {
          if (res !== null) {
            data.set(res);

            if (options.showSuccessMessage) {
              this.handleSuccess(method, options.successMessage);
            }
          }
        });
    };

    run();

    return { data, error, isLoading, refresh: run };
  }

  public get<T>(
    path: string,
    criteria?: SearchCriteria[],
    options: ApiOptions = {}
  ): ApiResult<T> {
    const params = new HttpParams({
      fromObject: Object.fromEntries(
        criteria?.map(c => [c.name, c.value]) ?? []
      )
    });

    return this.execute(
      'GET',
      path,
      () => this.http.get<T>(`${environment.baseUrl}${path}`, { params }),
      { ...this.defaultOptions.get, ...options }
    );
  }


  public mutation<TResponse, TBody = void>(
    method: 'POST' | 'PUT' | 'DELETE',
    path: string,
    onSuccess?: (data: TResponse) => void | Promise<void>,
    onError?: (error: ApiError) => void | Promise<void>,
    options: ApiOptions = {}
  ): ApiMutation<TResponse, TBody> {
    const data = signal<TResponse | null>(null);
    const isLoading = signal(false);
    const error = signal<ApiError | null>(null);
    const success = signal<MutationSuccess<TResponse> | null>(null);

    const reset = () => {
      data.set(null);
      error.set(null);
      success.set(null);
    };

    return {
      data,
      isLoading,
      error,
      success,
      reset,
      execute: ((body?: TBody) => {
        isLoading.set(true);
        error.set(null);

        this.http.request<TResponse>(method, `${environment.baseUrl}${path}`, {
          body,
          headers: new HttpHeaders({ 'Content-Type': 'application/json' }),
          observe: 'body',
          responseType: 'json'
        })
          .pipe(
            catchError(err => {
              const apiError: ApiError = {
                status: err.status,
                message: err.error?.message ?? 'Erreur inconnue',
                raw: err
              };

              error.set(apiError);
              if (onError) {
                onError(apiError);
              }

              if (options.showErrorMessage !== false && !onError) {
                this.handleError(apiError, path);
              }

              return of(null);
            }),
            finalize(() => isLoading.set(false))
          )
          .subscribe(res => {
            if (res !== null) {
              data.set(res);

              if (onSuccess) {
                onSuccess(res);
              }

              success.set({ data: res, timestamp: Date.now() });

              if (options.showSuccessMessage) {
                this.handleSuccess(method, options.successMessage);
              }
            }
          });
      }) as TBody extends void ? () => void : (body: TBody) => void
    };
  }

  public post<TResponse, TBody = void>(
    path: string,
    onSuccess?: (data: TResponse) => void | Promise<void>,
    onError?: (error: ApiError) => void | Promise<void>,
    options: ApiOptions = {}
  ): ApiMutation<TResponse, TBody> {
    return this.mutation<TResponse, TBody>('POST', path, onSuccess, onError, options);
  }

  public put<TResponse, TBody = void>(
    path: string,
    onSuccess?: (data: TResponse) => void | Promise<void>,
    onError?: (error: ApiError) => void | Promise<void>,
    options: ApiOptions = {}
  ): ApiMutation<TResponse, TBody> {
    return this.mutation<TResponse, TBody>('PUT', path, onSuccess, onError, options);
  }

  public delete<TResponse, TBody = void>(
    path: string,
    onSuccess?: (data: TResponse) => void | Promise<void>,
    onError?: (error: ApiError) => void | Promise<void>,
    options: ApiOptions = {}
  ): ApiMutation<TResponse, TBody> {
    return this.mutation<TResponse, TBody>('DELETE', path, onSuccess, onError, options);
  }

  private handleError(error: ApiError, path: string) {
    console.error(`API error on ${path}`, error);
    this.messageService.add({
      severity: 'error',
      summary: 'Erreur',
      detail: error.message
    });
  }

  private handleSuccess(method: string, message?: string) {
    const fallback = {
      POST: 'Création réussie',
      PUT: 'Mise à jour réussie',
      DELETE: 'Suppression réussie'
    }[method] ?? 'Succès';

    this.messageService.add({
      severity: 'success',
      summary: 'Succès',
      detail: message ?? fallback
    });
  }
}
