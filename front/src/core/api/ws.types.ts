import { Signal } from '@angular/core';

export type WsStatus =
  | 'connecting'
  | 'connected'
  | 'disconnected'
  | 'error';


export interface WsConnection<T = any> {
  status: Signal<WsStatus>;
  error: Signal<any>;
  send: (message: WsMessage) => void;
  close: () => void;
}

export type MessageType =
// Job
  | 'job_update'
  | 'job_delete'
  | 'job_create'
  | 'job_execute'
  // User
  | 'cursor_move'
  | 'chat'
  | 'user_join'
  | 'user_leave'
  // System
  | 'error'
  | 'ping'
  | 'pong';

export interface WsMessage<T = any> {
  type: MessageType;
  jobId?: number;
  userId: number;
  username: string;
  timestamp: string; // ISO string
  data: T;
}
