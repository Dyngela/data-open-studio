import {inject, Injectable, Signal} from '@angular/core';
import {HttpClient, HttpHeaders} from '@angular/common/http';
import {MessageService} from 'primeng/api';
import {Observable} from 'rxjs';
import {environment} from '../../environments/environment';

export interface ApiResult<T> {
  data: Signal<T | null>;
  error: Signal<ApiError | null>;
  isLoading: Signal<boolean>;
  refresh: () => void;
}

export interface MutationSuccess<T> {
  data: T;
  timestamp: number;
}

export interface MutationError {
  error: ApiError;
  timestamp: number;
}

export interface ApiMutation<TResponse = void, TBody = void> {
  data: Signal<TResponse | null>;
  execute: TBody extends void ? () => void : (body: TBody) => void;
  isLoading: Signal<boolean>;
  error: Signal<ApiError | null>;
  success: Signal<MutationSuccess<TResponse> | null>;
  reset: () => void;
}

export interface ApiOptions {
  showSuccessMessage?: boolean;
  successMessage?: string;
  showErrorMessage?: boolean;
}

export interface ApiError {
  status: number;
  message: string;
  raw?: any;
}

export interface SearchCriteria {
  name: string;
  value: any;
}
