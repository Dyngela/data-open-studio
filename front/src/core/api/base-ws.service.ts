import { computed, inject, Injectable, Signal, signal } from '@angular/core';
import { MessageType, WsConnection, WsMessage, WsStatus } from './ws.types';
import { MessageService } from 'primeng/api';

export interface WsChannel<T> {
  data: Signal<T | null>;
  isConnected: Signal<boolean>;
  error: Signal<any>;
  send: (payload: T) => void;
}

export interface WsError {
  message: string;
  raw?: any;
}

export interface WsResult<T> {
  data: Signal<T | null>;
  error: Signal<WsError | null>;
  isLoading: Signal<boolean>;
}

export interface WsMutationSuccess<T> {
  data: T;
  timestamp: number;
}

export interface WsMutation<TResponse, TRequest = void> {
  data: Signal<TResponse | null>;
  execute: TRequest extends void ? () => void : (request: TRequest) => void;
  isLoading: Signal<boolean>;
  error: Signal<WsError | null>;
  success: Signal<WsMutationSuccess<TResponse> | null>;
  reset: () => void;
}

export interface WsMutationOptions {
  showSuccessMessage?: boolean;
  successMessage?: string;
  showErrorMessage?: boolean;
  timeout?: number;
}

@Injectable({ providedIn: 'root' })
export class BaseWebSocketService {
  private messageService = inject(MessageService);
  private socket: WebSocket | null = null;

  private status = signal<WsStatus>('disconnected');
  public isConnected = computed(() => this.status() === 'connected');
  private error = signal<any>(null);
  private lastMessage = signal<WsMessage | null>(null);

  private reconnectAttempts = 0;
  private readonly maxReconnects = 5;

  // User context for messages
  private userId = signal(0);
  private username = signal('');

  connect(url: string): WsConnection {
    this.open(url);

    return {
      status: this.status,
      error: this.error,
      send: (msg) => this.send(msg),
      close: () => this.close()
    };
  }

  private open(url: string) {
    this.status.set('connecting');

    this.socket = new WebSocket(url);

    this.socket.onopen = () => {
      this.reconnectAttempts = 0;
      this.status.set('connected');

      // TODO
      // this.userId.set(userId);
      // this.username.set(username);
    };

    this.socket.onmessage = (event) => {
      try {
        this.lastMessage.set(JSON.parse(event.data));
      } catch (e) {
        this.error.set(e);
      }
    };

    this.socket.onerror = (err) => {
      this.error.set(err);
      this.status.set('error');
    };

    this.socket.onclose = () => {
      this.status.set('disconnected');
      this.tryReconnect(url);
    };
  }

  private send(message: WsMessage) {
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(message));
    }
  }

  private close() {
    this.socket?.close();
    this.socket = null;
    this.status.set('disconnected');
  }

  private tryReconnect(url: string) {
    if (this.reconnectAttempts >= this.maxReconnects) return;

    this.reconnectAttempts++;
    setTimeout(() => this.open(url), 1000 * this.reconnectAttempts);
  }

  /**
   * Create a WebSocket mutation (request/response pattern)
   * Similar to HTTP mutations but over WebSocket
   */
  mutation<TResponse, TRequest = void>(
    requestType: MessageType,
    responseType: MessageType,
    onSuccess?: (data: TResponse) => void,
    onError?: (error: WsError) => void,
    options: WsMutationOptions = {}
  ): WsMutation<TResponse, TRequest> {
    const data = signal<TResponse | null>(null);
    const isLoading = signal(false);
    const error = signal<WsError | null>(null);
    const success = signal<WsMutationSuccess<TResponse> | null>(null);

    const timeoutMs = options.timeout ?? 30000;

    const reset = () => {
      data.set(null);
      error.set(null);
      success.set(null);
    };

    const execute = (request?: TRequest) => {
      isLoading.set(true);
      error.set(null);

      let timeoutId: ReturnType<typeof setTimeout>;
      let messageHandler: (event: MessageEvent) => void;

      const cleanup = () => {
        clearTimeout(timeoutId);
        this.socket?.removeEventListener('message', messageHandler);
      };

      // Timeout handler
      timeoutId = setTimeout(() => {
        cleanup();
        const wsError: WsError = { message: 'Timeout: le serveur ne répond pas' };
        error.set(wsError);
        isLoading.set(false);

        if (onError) {
          onError(wsError);
        } else if (options.showErrorMessage !== false) {
          this.handleError(wsError);
        }
      }, timeoutMs);

      // Response handler
      messageHandler = (event: MessageEvent) => {
        try {
          const msg: WsMessage = JSON.parse(event.data);

          if (msg.type === responseType) {
            cleanup();
            isLoading.set(false);
            data.set(msg.data as TResponse);
            success.set({ data: msg.data as TResponse, timestamp: Date.now() });

            if (onSuccess) {
              onSuccess(msg.data as TResponse);
            }

            if (options.showSuccessMessage) {
              this.handleSuccess(options.successMessage);
            }
          } else if (msg.type === MessageType.Error) {
            cleanup();
            isLoading.set(false);
            const wsError: WsError = {
              message: (msg.data as any)?.message || 'Erreur serveur',
              raw: msg.data
            };
            error.set(wsError);

            if (onError) {
              onError(wsError);
            } else if (options.showErrorMessage !== false) {
              this.handleError(wsError);
            }
          }
        } catch (e) {
          // Ignore parse errors for non-matching messages
        }
      };

      this.socket?.addEventListener('message', messageHandler);

      // Send the request
      const message: WsMessage<TRequest> = {
        type: requestType,
        userId: this.userId(),
        username: this.username(),
        timestamp: new Date().toISOString(),
        data: request as TRequest
      };

      this.send(message);
    };

    return {
      data,
      isLoading,
      error,
      success,
      reset,
      execute: execute as TRequest extends void ? () => void : (request: TRequest) => void
    };
  }

  private handleError(error: WsError) {
    console.error('WebSocket error:', error);
    this.messageService.add({
      severity: 'error',
      summary: 'Erreur',
      detail: error.message
    });
  }

  private handleSuccess(message?: string) {
    this.messageService.add({
      severity: 'success',
      summary: 'Succès',
      detail: message ?? 'Opération réussie'
    });
  }
}
