import {Component, EventEmitter, HostListener, Input, OnInit, Output} from '@angular/core';
import {Node} from "../../models/node.model";

@Component({
  template: '' // This will be an abstract base class, so no template
})
export abstract class BaseNodeComponent {

  @Input() node!: Node;
  @Output() nodeMove = new EventEmitter<{ id: string, x: number, y: number }>();
  @Output() nodeRightClick = new EventEmitter<{ nodeId: string, x: number, y: number }>();
  @Output() inputDrop = new EventEmitter<{ event: MouseEvent, node: Node, connectorId: string }>();
  @Output() outputDrag = new EventEmitter<{ event: MouseEvent, node: Node, connectorId: string }>();

  private offset = { x: 0, y: 0 };
  private dragging = false;

  onInputMouseUp($event: MouseEvent, connectorId: string) {
    this.outputDrag.emit({ event: $event, node: this.node, connectorId: connectorId })
  }

  onOutputMouseDown($event: MouseEvent, connectorId: string) {
    this.inputDrop.emit({ event: $event, node: this.node, connectorId: connectorId })
  }

  @HostListener('mousedown', ['$event'])
  onMouseDown(event: MouseEvent) {
    const target = event.target as HTMLElement;
    // Check if the target is a triangle
    if (target.closest('.triangle-right')) {
      return; // Do not initiate drag if the target is a triangle
    }

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
      this.node.position = { x: newX, y: newY };
      this.nodeMove.emit({ id: this.node.id, x: newX, y: newY });
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
