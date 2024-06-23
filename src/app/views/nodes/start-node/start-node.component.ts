import {Component, EventEmitter, HostListener, OnInit, Output} from '@angular/core';
import {BaseNodeClass} from "../base-node.class";

@Component({
  selector: 'app-start-node',
  templateUrl: './start-node.component.html',
  styleUrls: ['./start-node.component.css']
})
export class StartNodeComponent extends BaseNodeClass {



  constructor() {
    super();
  }


}
