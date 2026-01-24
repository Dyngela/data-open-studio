import {Component, inject, OnInit, signal} from '@angular/core';
import { Playground } from './playground/playground/playground';
import { JobsWsService } from '../core/api/job.service';
import { DbType } from '../core/api/metadata.type';
import {BaseWebSocketService} from '../core/services/base-ws.service';
import {DbNodeService} from '../core/api/db-node.service';

@Component({
  selector: 'app-root',
  imports: [Playground],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App implements OnInit {
  protected readonly title = signal('front');
  private readonly dbNodeService = inject(DbNodeService)

  protected guessDataModel =   this.dbNodeService.guessSchema( (response) => {
    console.log('Guessed Data Models:', response.dataModels);
  })

  ngOnInit(): void {
        this.guessDataModel.execute({
          nodeId: 'node-123',
        query: 'SELECT * FROM job',
        dbType: DbType.Postgres,
        host: 'localhost',
        port: 5433,
        database: 'data-open-studio',
        username: 'postgres',
        password: 'postgres'
      })
  }



  onGuessDataModel() {



  }
}
