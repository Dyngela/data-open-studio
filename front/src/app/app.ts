import {Component, inject, OnInit, signal} from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { Playground } from './playground/playground/playground';
import { JobsWsService } from '../core/api/job.service';
import { DbType } from '../core/api/ws.types';
import {BaseWebSocketService} from '../core/api/base-ws.service';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, Playground],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App implements OnInit {
  protected readonly title = signal('front');
  private ws = inject(BaseWebSocketService);

  private jobsWs = inject(JobsWsService);

  ngOnInit(): void {
    this.ws.connect("http://localhost:8080/api/v1/ws/init")
  }

  // Create mutation for guessing data model
  guessDataModel = this.jobsWs.guessDataModel(
    (response) => console.log('Data models received:', response.dataModels),
    (error) => console.error('Error:', error.message)
  );

  // Example: trigger data model guess
  onGuessDataModel() {
    console.log('Guessing data model...');
    this.guessDataModel.execute({
      nodeId: 1,
      jobId: 1,
      query: 'SELECT * FROM employees',
      dbType: DbType.Postgres,
      dbSchema: 'public',
      host: 'localhost',
      port: 5434,
      database: 'testdb',
      username: 'testuser',
      password: 'testpass',
      sslMode: 'disable',
    });
  }
}
