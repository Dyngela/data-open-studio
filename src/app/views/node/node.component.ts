import {Component, Input, Output, EventEmitter, HostListener} from '@angular/core';
import { Node } from 'src/app/models/node.model';
@Component({
  selector: 'app-node',
  templateUrl: './node.component.html',
  styleUrls: ['./node.component.css']
})
export class NodeComponent {
  @Input() node!: Node;
  @Output() nodeMove = new EventEmitter<{ id: string, x: number, y: number }>();
  @Output() nodeRightClick = new EventEmitter<{ nodeId: string, x: number, y: number }>();

  private offset = { x: 0, y: 0 };
  private dragging = false;

  @HostListener('mousedown', ['$event'])
  onMouseDown(event: MouseEvent) {
    if (event.button === 0) { // Left mouse button
      this.offset = {
        x: event.clientX - this.node.position.x,
        y: event.clientY - this.node.position.y
      };
      this.dragging = true;
      event.preventDefault();
    }
  }

  @HostListener('document:mousemove', ['$event'])
  onMouseMove(event: MouseEvent) {
    if (this.dragging) {
      const newX = event.clientX - this.offset.x;
      const newY = event.clientY - this.offset.y;
      this.nodeMove.emit({ id: this.node.id, x: newX, y: newY });
      event.preventDefault();
    }
  }

  @HostListener('document:mouseup', ['$event'])
  onMouseUp(event: MouseEvent) {
    if (event.button === 0) { // Left mouse button
      this.dragging = false;
      event.preventDefault();
    }
  }

  @HostListener('contextmenu', ['$event'])
  onRightClick(event: MouseEvent) {
    event.preventDefault();
    event.stopPropagation(); // Prevent event from bubbling up
    // Emit event with the node ID
    this.nodeRightClick.emit({ nodeId: this.node.id, x: event.clientX, y: event.clientY });
  }
}
