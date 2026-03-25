import { inject, Injectable } from '@angular/core';
import { BaseApiService } from '../services/base-api.service';
import { ApiMutation, ApiResult } from '../services/base-api.type';
import {
  Dataset,
  DatasetWithDetails,
  CreateDatasetRequest,
  UpdateDatasetRequest,
  DatasetPreviewRequest,
  DatasetPreviewResult,
  DatasetQueryRequest,
  DatasetQueryResult,
  DeleteResponse,
} from './dataset.type';

@Injectable({ providedIn: 'root' })
export class DatasetService {
  private api = inject(BaseApiService);
  private readonly basePath = '/datasets';

  getAll(onSuccess?: (data: Dataset[]) => void): ApiResult<Dataset[]> {
    return this.api.get<Dataset[]>(this.basePath, undefined, onSuccess);
  }

  getById(id: number, onSuccess?: (data: DatasetWithDetails) => void): ApiResult<DatasetWithDetails> {
    return this.api.get<DatasetWithDetails>(`${this.basePath}/${id}`, undefined, onSuccess);
  }

  create(
    onSuccess?: (data: DatasetWithDetails) => void,
    onError?: (error: any) => void
  ): ApiMutation<DatasetWithDetails, CreateDatasetRequest> {
    return this.api.post<DatasetWithDetails, CreateDatasetRequest>(this.basePath, onSuccess, onError);
  }

  update(
    id: number,
    onSuccess?: (data: DatasetWithDetails) => void,
    onError?: (error: any) => void
  ): ApiMutation<DatasetWithDetails, UpdateDatasetRequest> {
    return this.api.put<DatasetWithDetails, UpdateDatasetRequest>(`${this.basePath}/${id}`, onSuccess, onError);
  }

  deleteDataset(
    id: number,
    onSuccess?: (data: DeleteResponse) => void,
    onError?: (error: any) => void
  ): ApiMutation<DeleteResponse, void> {
    return this.api.delete<DeleteResponse, void>(`${this.basePath}/${id}`, onSuccess, onError);
  }

  refresh(
    id: number,
    onSuccess?: (data: DatasetWithDetails) => void,
    onError?: (error: any) => void
  ): ApiMutation<DatasetWithDetails, void> {
    return this.api.post<DatasetWithDetails, void>(`${this.basePath}/${id}/refresh`, onSuccess, onError);
  }

  preview(
    id: number,
    onSuccess?: (data: DatasetPreviewResult) => void,
    onError?: (error: any) => void
  ): ApiMutation<DatasetPreviewResult, DatasetPreviewRequest> {
    return this.api.post<DatasetPreviewResult, DatasetPreviewRequest>(`${this.basePath}/${id}/preview`, onSuccess, onError);
  }

  query(
    id: number,
    onSuccess?: (data: DatasetQueryResult) => void,
    onError?: (error: any) => void
  ): ApiMutation<DatasetQueryResult, DatasetQueryRequest> {
    return this.api.post<DatasetQueryResult, DatasetQueryRequest>(`${this.basePath}/${id}/query`, onSuccess, onError);
  }
}
