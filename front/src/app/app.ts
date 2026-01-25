import {Component, inject, OnInit, signal} from '@angular/core';
import {DbNodeService} from '../core/api/db-node.service';
import {DbType} from '../core/api/metadata.type';
import {RouterOutlet, RouterLink, RouterLinkActive} from '@angular/router';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, RouterLink, RouterLinkActive],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App implements OnInit {
  protected readonly title = signal('front');
  private readonly dbNodeService = inject(DbNodeService)

  protected guessDataModel =
    this.dbNodeService.guessSchema( (response) => {
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
}
