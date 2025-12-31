import { Component, input, output, ElementRef, AfterViewInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { CdkDrag } from '@angular/cdk/drag-drop';
import { NodeInstance } from '../models/node.model';

@Component({
  selector: 'app-node-instance',
  standalone: true,
  imports: [CommonModule, CdkDrag],
  templateUrl: './node-instance.html',
  styleUrl: './node-instance.css',
})
export class NodeInstanceComponent implements AfterViewInit {
  node = input.required<NodeInstance>();
  panOffset = input({ x: 0, y: 0 });
  isPanning = input(false);
  positionChanged = output<{ nodeId: string; position: { x: number; y: number } }>();
  outputPortClick = output<{ nodeId: string; portIndex: number }>();
  inputPortClick = output<{ nodeId: string; portIndex: number }>();

  constructor(private elementRef: ElementRef) {}

  ngAfterViewInit() {
    this.updatePortPositions();
  }

  onDragEnded(event: any) {
    const distance = event.distance;
    const currentNode = this.node();

    currentNode.position.x += distance.x;
    currentNode.position.y += distance.y;

    event.source.reset();

    this.positionChanged.emit({
      nodeId: currentNode.id,
      position: currentNode.position,
    });

    this.updatePortPositions();
  }

  onOutputPortClick(portIndex: number, event: MouseEvent) {
    event.stopPropagation();
    this.outputPortClick.emit({ nodeId: this.node().id, portIndex });
  }

  onInputPortClick(portIndex: number, event: MouseEvent) {
    event.stopPropagation();
    this.inputPortClick.emit({ nodeId: this.node().id, portIndex });
  }

  getInputPorts(): number[] {
    return Array.from({ length: this.node().type.inputs }, (_, i) => i);
  }

  getOutputPorts(): number[] {
    return Array.from({ length: this.node().type.outputs }, (_, i) => i);
  }

  private updatePortPositions() {
    setTimeout(() => {
      const currentNode = this.node();
      this.positionChanged.emit({
        nodeId: currentNode.id,
        position: currentNode.position,
      });
    }, 0);
  }
}
