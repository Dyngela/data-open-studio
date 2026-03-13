// Request payload for guessing schema

export interface ScriptRequest {
  code: string;
}

export interface ScriptCompilationError {
  type?: string;
  message: string;
  line?: number;
  column?: number;
}

export interface ScriptResponse {
  output: string;
  error: boolean;
  compilationError?: ScriptCompilationError;
}