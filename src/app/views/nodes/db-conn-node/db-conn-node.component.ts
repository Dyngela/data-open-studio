import { Component } from '@angular/core';
import {BaseNodeComponent} from "../base-node.component";

@Component({
  selector: 'app-db-conn-node',
  templateUrl: './db-conn-node.component.html',
  styleUrls: ['./db-conn-node.component.css']
})
export class DbConnNodeComponent extends BaseNodeComponent {
  constructor() {
    super();
  }

}
