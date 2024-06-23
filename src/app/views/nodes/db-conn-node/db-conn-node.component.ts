import { Component } from '@angular/core';
import {BaseNodeClass} from "../base-node.class";

@Component({
  selector: 'app-db-conn-node',
  templateUrl: './db-conn-node.component.html',
  styleUrls: ['./db-conn-node.component.css']
})
export class DbConnNodeComponent extends BaseNodeClass {
  constructor() {
    super();
  }

}
