import {inject, Injectable} from '@angular/core';
import { BaseApiService } from '../services/base-api.service';
import { ApiMutation, ApiResult, SearchCriteria } from '../services/base-api.type';
import {
  Job,
  JobWithNodes,
  CreateJobRequest,
  UpdateJobRequest,
  ShareJobRequest,
  DeleteResponse
} from './job.type';
import {ApiError} from './api-response.type';

@Injectable({ providedIn: 'root' })
export class JobService {
  private api = inject(BaseApiService)
  private readonly basePath = '/jobs';

  /**
   * Get all jobs for the current user
   * Optionally filter by filePath
   */
  getAll(filePath?: string): ApiResult<Job[]> {
    const criteria: SearchCriteria[] = filePath
      ? [{ name: 'filePath', value: filePath }]
      : [];
    return this.api.get<Job[]>(this.basePath, criteria);
  }

  /**
   * Get a single job by ID with its nodes and sharing info
   */
  getById(id: number): ApiResult<JobWithNodes> {
    return this.api.get<JobWithNodes>(`${this.basePath}/${id}`);
  }

  /**
   * Create a new job
   */
  create(
    onSuccess?: (data: JobWithNodes) => void,
    onError?: (error: any) => void
  ): ApiMutation<JobWithNodes, CreateJobRequest> {
    return this.api.post<JobWithNodes, CreateJobRequest>(
      this.basePath,
      onSuccess,
      onError
    );
  }

  /**
   * Update an existing job
   */
  update(
    id: number,
    onSuccess?: (data: JobWithNodes) => void,
    onError?: (error: any) => void
  ): ApiMutation<JobWithNodes, UpdateJobRequest> {
    return this.api.put<JobWithNodes, UpdateJobRequest>(
      `${this.basePath}/${id}`,
      onSuccess,
      onError
    );
  }

  /**
   * Delete a job (only owner can delete)
   */
  deleteJob(
    id: number,
    onSuccess?: (data: DeleteResponse) => void,
    onError?: (error: any) => void
  ): ApiMutation<DeleteResponse, void> {
    return this.api.delete<DeleteResponse, void>(
      `${this.basePath}/${id}`,
      onSuccess,
      onError
    );
  }

  /**
   * Share a job with users (only owner can share)
   */
  share(
    id: number,
    onSuccess?: (data: JobWithNodes) => void,
    onError?: (error: any) => void
  ): ApiMutation<JobWithNodes, ShareJobRequest> {
    return this.api.post<JobWithNodes, ShareJobRequest>(
      `${this.basePath}/${id}/share`,
      onSuccess,
      onError
    );
  }

  /**
   * Remove users from job access (only owner can unshare)
   */
  unshare(
    id: number,
    onSuccess?: (data: JobWithNodes) => void,
    onError?: (error: any) => void
  ): ApiMutation<JobWithNodes, ShareJobRequest> {
    return this.api.delete<JobWithNodes, ShareJobRequest>(
      `${this.basePath}/${id}/share`,
      onSuccess,
      onError
    );
  }

  execute(id: number,
          onSuccess?: (data: void) => void,
          onError?: (error: any) => void
  ): ApiMutation<ApiError<any>, null> {
    return this.api.post(`${this.basePath}/${id}/execute`)
  }
}
