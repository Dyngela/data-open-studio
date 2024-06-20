import {Component, EventEmitter, HostListener, OnInit, Output} from '@angular/core';
import {BaseNodeComponent} from "../base-node.component";

@Component({
  selector: 'app-start-node',
  templateUrl: './start-node.component.html',
  styleUrls: ['./start-node.component.css']
})
export class StartNodeComponent extends BaseNodeComponent {



  constructor() {
    super();
  }


}
