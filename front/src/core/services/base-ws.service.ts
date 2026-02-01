import { inject, Injectable, NgZone, OnDestroy, signal } from '@angular/core';
import { AuthService } from '../api/auth.service';
import { environment } from '../../environments/environment';

/** Matches the Go lib.Progress struct published via NATS */
export interface JobProgress {
  nodeId: number;
  nodeName: string;
  status: 'running' | 'completed' | 'failed';
  rowCount: number;
  message: string;
}

/** Envelope sent by the realtime server (outgoingMsg in Go) */
interface WsEnvelope {
  type: string;
  jobId: number;
  payload: JobProgress;
}

export type ProgressEvent = JobProgress & { jobId: number };
export type WsState = 'disconnected' | 'connecting' | 'connected';

@Injectable({ providedIn: 'root' })
export class JobRealtimeService implements OnDestroy {
  private auth = inject(AuthService);
  private zone = inject(NgZone);

  private ws: WebSocket | null = null;
  private subscribedJobId: number | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private intentionalClose = false;
  private progressListeners: ((p: ProgressEvent) => void)[] = [];

  readonly state = signal<WsState>('disconnected');

  /**
   * Register a callback invoked for each progress message (no batching).
   * Returns an unsubscribe function.
   */
  onProgress(listener: (p: ProgressEvent) => void): () => void {
    this.progressListeners.push(listener);
    return () => {
      const i = this.progressListeners.indexOf(listener);
      if (i >= 0) this.progressListeners.splice(i, 1);
    };
  }

  /**
   * Connect to the realtime WebSocket and subscribe to a job's progress.
   * Returns a Promise that resolves once the subscription message is sent.
   */
  subscribeToJob(jobId: number): Promise<void> {
    this.subscribedJobId = jobId;
    this.intentionalClose = false;

    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.sendSubscribe(jobId);
      return Promise.resolve();
    }

    return this.connect(jobId);
  }

  /** Disconnect and stop reconnecting */
  disconnect(): void {
    this.intentionalClose = true;
    this.subscribedJobId = null;
    this.clearReconnect();

    if (this.ws) {
      this.ws.close(1000, 'client disconnect');
      this.ws = null;
    }
    this.state.set('disconnected');
  }

  ngOnDestroy(): void {
    this.disconnect();
  }

  private connect(jobId: number): Promise<void> {
    const token = this.auth.getAccessToken();
    if (!token) {
      console.warn('JobRealtimeService: no access token, cannot connect');
      return Promise.reject('no access token');
    }

    this.state.set('connecting');
    const url = `${environment.wsUrl}?token=${encodeURIComponent(token)}`;

    return new Promise<void>((resolve, reject) => {
        this.ws = new WebSocket(url);

        this.ws.onopen = () => {
          this.zone.run(() => this.state.set('connected'));
          this.sendSubscribe(jobId);
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const envelope: WsEnvelope = JSON.parse(event.data);
            if (envelope.type === 'job.progress') {
              const progress: ProgressEvent = { ...envelope.payload, jobId: envelope.jobId };
              // Dispatch outside the current CD cycle to avoid NG0100
              // (node status update recalculates SVG paths mid-check)
              setTimeout(() => this.zone.run(() => {
                for (const fn of this.progressListeners) fn(progress);
              }));
            }
          } catch {
            // ignore non-JSON frames (pong, etc.)
          }
        };

        this.ws.onerror = (err) => {
          console.error('JobRealtimeService: ws error', err);
          reject(err);
        };

        this.ws.onclose = () => {
          this.zone.run(() => this.state.set('disconnected'));
          this.ws = null;

          if (!this.intentionalClose && this.subscribedJobId != null) {
            this.scheduleReconnect();
          }
        };
      });
  }

  private sendSubscribe(jobId: number): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ action: 'subscribe', jobId }));
    }
  }

  private scheduleReconnect(): void {
    this.clearReconnect();
    this.reconnectTimer = setTimeout(() => {
      if (this.subscribedJobId != null) {
        this.connect(this.subscribedJobId);
      }
    }, 3000);
  }

  private clearReconnect(): void {
    if (this.reconnectTimer != null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }
}
