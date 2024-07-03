import {Component, ElementRef} from '@angular/core';
import {TitleStrategy} from "@angular/router";
import {Title} from "@angular/platform-browser";

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent {
  title = 'Data-open-studio';

  constructor(public titleService: Title) {
    titleService.setTitle(this.title)
  }
}
