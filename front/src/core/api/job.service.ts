import {BaseWebSocketService} from './base-ws.service';
import {inject, Injectable} from '@angular/core';
import {Job} from './job.model';

@Injectable({ providedIn: 'root' })
export class JobsWsService {
  private ws = inject(BaseWebSocketService); // your base WS runtime
  private userId = 123;           // optional metadata
  private username = 'John';

  // Observe job_update events
  jobUpdates = this.ws.channel<Job>('job_update');

  constructor() {
    this.ws.connect('wss://api.example.com/ws');
  }

  emitJobUpdate(job: Job) {
    this.jobUpdates.send(job);
  }
}
