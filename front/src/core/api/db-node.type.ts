// Request payload for guessing schema
import {DataModel, DbType} from './metadata.type';

export interface GuessSchemaRequest {
  nodeId: string;
  query: string;
  connectionId: number;
}

// Response from guess schema endpoint
export interface GuessSchemaResponse {
  nodeId: string;
  dataModels: DataModel[];
}
