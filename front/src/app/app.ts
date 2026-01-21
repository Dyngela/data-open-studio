import { Component, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import {Playground} from './playground/playground/playground';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, Playground],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App {
  protected readonly title = signal('front');
}
