import {inject, Injectable} from '@angular/core';
import { ScriptRequest, ScriptResponse } from './script.type';
import {ApiMutation} from '../services/base-api.type';
import {BaseApiService} from '../services/base-api.service';



@Injectable({
  providedIn: 'root'
})
export class ScriptService {
  private api = inject(BaseApiService)

  /**
   * Execute a piece of code
   */
  executeCode(
    onSuccess?: (data: ScriptResponse) => void,
    onError?: (error: any) => void
  ): ApiMutation<ScriptResponse, ScriptRequest> {
    return this.api.post<ScriptResponse, ScriptRequest>(
      '/script/execute',
      onSuccess,
      onError
    );
  }
}
