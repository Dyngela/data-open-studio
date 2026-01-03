import {computed, Injectable, Signal, signal} from '@angular/core';
import {MessageType, WsConnection, WsMessage, WsStatus} from './ws.types';

export interface WsChannel<T> {
  data: Signal<T | null>;
  isConnected: Signal<boolean>;
  error: Signal<any>;
  send: (payload: T) => void;
}

@Injectable({ providedIn: 'root' })
export class BaseWebSocketService {
  private socket: WebSocket | null = null;

  private status = signal<WsStatus>('disconnected');
  public isConnected = computed(() => this.status() === 'connected');
  private error = signal<any>(null);
  private lastMessage = signal<WsMessage | null>(null);

  private reconnectAttempts = 0;
  private readonly maxReconnects = 5;

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
    };

    this.socket.onmessage = (event) => {
      try {
        this.lastMessage.set(JSON.parse(event.data))
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

  channel<T>(type: MessageType): WsChannel<T> {
    const data = computed<T | null>(() => {
      const msg = this.lastMessage()?.data;
      return msg?.type === type ? (msg) : null;
    });


    return {
      data,
      isConnected: this.isConnected,
      error: this.error,
      send: (payload: T) => {
        if (!this.isConnected()) return;
        const p: WsMessage<T> = {timestamp: '', userId: 0, username: '', type, data: payload };
        this.socket?.send(JSON.stringify(p));
      }
    };
  }
}
