import {Component, input, output} from '@angular/core';
import {NodeInstance} from '../../../core/nodes-services/node.type';

@Component({
  selector: 'app-log-modal',
  imports: [],
  templateUrl: './log.modal.html',
  styleUrl: './log.modal.css',
})
export class LogModal {
  close = output<void>();
  node = input.required<NodeInstance>();
}
