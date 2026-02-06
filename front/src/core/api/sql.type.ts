export interface GuessQueryRequest {
  prompt: string;
  connectionId: number;
  schemaOptimizationNeeded?: boolean;
  previousMessages?: string[];
}

export interface GuessQueryResponse {
  query: string;
}

export interface OptimizeQueryRequest {
  query: string;
  connectionId: number;
}

export interface OptimizeQueryResponse {
  optimizedQuery: string;
  explanation: string;
}
