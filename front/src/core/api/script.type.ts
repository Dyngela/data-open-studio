// Request payload for guessing schema

export interface ScriptRequest {
  code: string;
}

export interface ScriptResponse {
  output: string;
  error: boolean;
}