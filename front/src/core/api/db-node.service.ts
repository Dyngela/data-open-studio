import {inject, Injectable} from '@angular/core';
import {GuessSchemaRequest, GuessSchemaResponse} from './db-node.type';
import {ApiMutation} from '../services/base-api.type';
import {BaseApiService} from '../services/base-api.service';



@Injectable({
  providedIn: 'root'
})
export class DbNodeService  {
  private api = inject(BaseApiService)

  /**
   * Guess the schema/data model from a SQL query
   * Introspects the database and returns column information
   */
  guessSchema(
    onSuccess?: (data: GuessSchemaResponse) => void,
    onError?: (error: any) => void
  ): ApiMutation<GuessSchemaResponse, GuessSchemaRequest> {
    return this.api.post<GuessSchemaResponse, GuessSchemaRequest>(
      '/db-node/guess-schema',
      onSuccess,
      onError
    );
  }
}
