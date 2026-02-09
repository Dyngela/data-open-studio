import { inject, Injectable } from '@angular/core';
import { BaseApiService } from '../services/base-api.service';
import { ApiMutation, ApiResult } from '../services/base-api.type';
import {
  Trigger,
  TriggerWithDetails,
  TriggerRule,
  TriggerExecution,
  CreateTriggerRequest,
  UpdateTriggerRequest,
  CreateTriggerRuleRequest,
  UpdateTriggerRuleRequest,
  LinkJobRequest,
  DeleteResponse,
  UnlinkJobResponse,
  TriggerJobLink,
} from './trigger.type';

@Injectable({ providedIn: 'root' })
export class TriggerService {
  private api = inject(BaseApiService);
  private readonly basePath = '/triggers';

  /**
   * Get all triggers for the current user
   */
  getAll(): ApiResult<Trigger[]> {
    return this.api.get<Trigger[]>(this.basePath);
  }

  /**
   * Get a single trigger by ID with full details
   */
  getById(id: number): ApiResult<TriggerWithDetails> {
    return this.api.get<TriggerWithDetails>(`${this.basePath}/${id}`);
  }

  /**
   * Create a new trigger
   */
  create(
    onSuccess?: (data: TriggerWithDetails) => void,
    onError?: (error: any) => void
  ): ApiMutation<TriggerWithDetails, CreateTriggerRequest> {
    return this.api.post<TriggerWithDetails, CreateTriggerRequest>(
      this.basePath,
      onSuccess,
      onError
    );
  }

  /**
   * Update an existing trigger
   */
  update(
    id: number,
    onSuccess?: (data: TriggerWithDetails) => void,
    onError?: (error: any) => void
  ): ApiMutation<TriggerWithDetails, UpdateTriggerRequest> {
    return this.api.put<TriggerWithDetails, UpdateTriggerRequest>(
      `${this.basePath}/${id}`,
      onSuccess,
      onError
    );
  }

  /**
   * Delete a trigger
   */
  deleteTrigger(
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
   * Activate a trigger (start polling)
   */
  activate(
    id: number,
    onSuccess?: (data: TriggerWithDetails) => void,
    onError?: (error: any) => void
  ): ApiMutation<TriggerWithDetails, void> {
    return this.api.post<TriggerWithDetails, void>(
      `${this.basePath}/${id}/activate`,
      onSuccess,
      onError
    );
  }

  /**
   * Pause a trigger (stop polling)
   */
  pause(
    id: number,
    onSuccess?: (data: TriggerWithDetails) => void,
    onError?: (error: any) => void
  ): ApiMutation<TriggerWithDetails, void> {
    return this.api.post<TriggerWithDetails, void>(
      `${this.basePath}/${id}/pause`,
      onSuccess,
      onError
    );
  }

  /**
   * Add a rule to a trigger
   */
  addRule(
    triggerId: number,
    onSuccess?: (data: TriggerRule) => void,
    onError?: (error: any) => void
  ): ApiMutation<TriggerRule, CreateTriggerRuleRequest> {
    return this.api.post<TriggerRule, CreateTriggerRuleRequest>(
      `${this.basePath}/${triggerId}/rules`,
      onSuccess,
      onError
    );
  }

  /**
   * Update a trigger rule
   */
  updateRule(
    triggerId: number,
    ruleId: number,
    onSuccess?: (data: TriggerRule) => void,
    onError?: (error: any) => void
  ): ApiMutation<TriggerRule, UpdateTriggerRuleRequest> {
    return this.api.put<TriggerRule, UpdateTriggerRuleRequest>(
      `${this.basePath}/${triggerId}/rules/${ruleId}`,
      onSuccess,
      onError
    );
  }

  /**
   * Delete a trigger rule
   */
  deleteRule(
    triggerId: number,
    ruleId: number,
    onSuccess?: (data: DeleteResponse) => void,
    onError?: (error: any) => void
  ): ApiMutation<DeleteResponse, void> {
    return this.api.delete<DeleteResponse, void>(
      `${this.basePath}/${triggerId}/rules/${ruleId}`,
      onSuccess,
      onError
    );
  }

  /**
   * Link a job to a trigger
   */
  linkJob(
    triggerId: number,
    onSuccess?: (data: TriggerJobLink) => void,
    onError?: (error: any) => void
  ): ApiMutation<TriggerJobLink, LinkJobRequest> {
    return this.api.post<TriggerJobLink, LinkJobRequest>(
      `${this.basePath}/${triggerId}/jobs`,
      onSuccess,
      onError
    );
  }

  /**
   * Unlink a job from a trigger
   */
  unlinkJob(
    triggerId: number,
    jobId: number,
    onSuccess?: (data: UnlinkJobResponse) => void,
    onError?: (error: any) => void
  ): ApiMutation<UnlinkJobResponse, void> {
    return this.api.delete<UnlinkJobResponse, void>(
      `${this.basePath}/${triggerId}/jobs/${jobId}`,
      onSuccess,
      onError
    );
  }

  /**
   * Get recent executions for a trigger
   */
  getExecutions(triggerId: number, limit: number = 20): ApiResult<TriggerExecution[]> {
    return this.api.get<TriggerExecution[]>(
      `${this.basePath}/${triggerId}/executions`,
      [{ name: 'limit', value: limit.toString() }]
    );
  }

}
