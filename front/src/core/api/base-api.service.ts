import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { inject } from '@angular/core';
import { Observable, throwError } from 'rxjs';
import { catchError, map } from 'rxjs/operators';
import { environment } from '../../environments/environment';
import {ApiResponse, Page, QueryParams} from '../models';

/**
 * Base API service with common HTTP methods
 * Extend this class for feature-specific services
 */
export abstract class BaseApiService {
  protected http = inject(HttpClient);
  protected baseUrl = environment.apiUrl;

  /**
   * Build complete URL with API version
   */
  protected buildUrl(endpoint: string): string {
    const version = environment.apiVersion;
    return `${this.baseUrl}/${version}/${endpoint}`;
  }

  /**
   * Build HTTP params from query object
   */
  protected buildParams(params?: QueryParams): HttpParams {
    let httpParams = new HttpParams();

    if (params) {
      Object.keys(params).forEach(key => {
        const value = params[key];
        if (value !== null && value !== undefined && value !== '') {
          httpParams = httpParams.append(key, String(value));
        }
      });
    }

    return httpParams;
  }

  /**
   * GET request
   */
  protected get<T>(endpoint: string, params?: QueryParams): Observable<T> {
    return this.http.get<T>(
      this.buildUrl(endpoint),
      { params: this.buildParams(params) }
    ).pipe(
      catchError(this.handleError)
    );
  }

  /**
   * GET request with ApiResponse wrapper
   */
  protected getWithResponse<T>(endpoint: string, params?: QueryParams): Observable<T> {
    return this.http.get<ApiResponse<T>>(
      this.buildUrl(endpoint),
      { params: this.buildParams(params) }
    ).pipe(
      map(response => response.data),
      catchError(this.handleError)
    );
  }

  /**
   * GET paginated list
   */
  protected getList<T>(endpoint: string, params?: QueryParams): Observable<Page<T>> {
    return this.http.get<Page<T>>(
      this.buildUrl(endpoint),
      { params: this.buildParams(params) }
    ).pipe(
      catchError(this.handleError)
    );
  }

  /**
   * POST request
   */
  protected post<T, R = T>(endpoint: string, body?: T): Observable<R> {
    return this.http.post<R>(
      this.buildUrl(endpoint),
      body
    ).pipe(
      catchError(this.handleError)
    );
  }

  /**
   * POST request with ApiResponse wrapper
   */
  protected postWithResponse<T, R = T>(endpoint: string, body: T): Observable<R> {
    return this.http.post<ApiResponse<R>>(
      this.buildUrl(endpoint),
      body
    ).pipe(
      map(response => response.data),
      catchError(this.handleError)
    );
  }

  /**
   * PUT request
   */
  protected put<T, R = T>(endpoint: string, body: T): Observable<R> {
    return this.http.put<R>(
      this.buildUrl(endpoint),
      body
    ).pipe(
      catchError(this.handleError)
    );
  }

  /**
   * PUT request with ApiResponse wrapper
   */
  protected putWithResponse<T, R = T>(endpoint: string, body: T): Observable<R> {
    return this.http.put<ApiResponse<R>>(
      this.buildUrl(endpoint),
      body
    ).pipe(
      map(response => response.data),
      catchError(this.handleError)
    );
  }

  /**
   * PATCH request
   */
  protected patch<T, R = T>(endpoint: string, body: Partial<T>): Observable<R> {
    return this.http.patch<R>(
      this.buildUrl(endpoint),
      body
    ).pipe(
      catchError(this.handleError)
    );
  }

  /**
   * PATCH request with ApiResponse wrapper
   */
  protected patchWithResponse<T, R = T>(endpoint: string, body: Partial<T>): Observable<R> {
    return this.http.patch<ApiResponse<R>>(
      this.buildUrl(endpoint),
      body
    ).pipe(
      map(response => response.data),
      catchError(this.handleError)
    );
  }

  /**
   * DELETE request
   */
  protected delete<T = void>(endpoint: string): Observable<T> {
    return this.http.delete<T>(
      this.buildUrl(endpoint)
    ).pipe(
      catchError(this.handleError)
    );
  }

  /**
   * DELETE request with ApiResponse wrapper
   */
  protected deleteWithResponse<T = void>(endpoint: string): Observable<T> {
    return this.http.delete<ApiResponse<T>>(
      this.buildUrl(endpoint)
    ).pipe(
      map(response => response.data),
      catchError(this.handleError)
    );
  }

  /**
   * Upload file
   */
  protected uploadFile<T>(endpoint: string, file: File, additionalData?: any): Observable<T> {
    const formData = new FormData();
    formData.append('file', file);

    if (additionalData) {
      Object.keys(additionalData).forEach(key => {
        formData.append(key, additionalData[key]);
      });
    }

    return this.http.post<T>(
      this.buildUrl(endpoint),
      formData
    ).pipe(
      catchError(this.handleError)
    );
  }

  /**
   * Handle HTTP errors
   */
  protected handleError(error: any): Observable<never> {
    let errorMessage = 'Une erreur est survenue';

    if (error.error instanceof ErrorEvent) {
      // Client-side error
      errorMessage = `Erreur: ${error.error.message}`;
    } else {
      // Server-side error
      errorMessage = error.error?.message || `Erreur serveur: ${error.status}`;
    }

    console.error('API Error:', error);
    return throwError(() => new Error(errorMessage));
  }
}
