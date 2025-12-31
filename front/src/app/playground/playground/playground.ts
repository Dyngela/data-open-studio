import { Component, ViewChild, ElementRef, AfterViewInit, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { CdkDropList, CdkDragDrop } from '@angular/cdk/drag-drop';
import { NodePanel } from '../node-panel/node-panel';
import { NodeInstanceComponent } from '../node-instance/node-instance';
import { Minimap } from '../minimap/minimap';
import { NodeInstance, Connection, NodeType } from '../models/node.model';

@Component({
  selector: 'app-playground',
  standalone: true,
  imports: [CommonModule, CdkDropList, NodePanel, NodeInstanceComponent, Minimap],
  templateUrl: './playground.html',
  styleUrl: './playground.css',
})
export class Playground implements AfterViewInit {
  @ViewChild('playgroundArea', { static: false }) playgroundArea!: ElementRef;

  nodes = signal<NodeInstance[]>([]);
  connections = signal<Connection[]>([]);

  viewportWidth = signal(0);
  viewportHeight = signal(0);

  private nodeIdCounter = 0;
  private connectionIdCounter = 0;

  private isConnecting = signal(false);
  private sourcePort = signal<{ nodeId: string; portIndex: number } | null>(null);
  tempConnection = signal<{ x1: number; y1: number; x2: number; y2: number } | null>(null);

  panOffset = signal({ x: 0, y: 0 });
  protected isPanning = signal(false);
  private panStart = signal({ x: 0, y: 0 });

  ngAfterViewInit() {
    this.updateViewportSize();
    window.addEventListener('resize', () => this.updateViewportSize());
  }

  private updateViewportSize() {
    if (this.playgroundArea) {
      const rect = this.playgroundArea.nativeElement.getBoundingClientRect();
      this.viewportWidth.set(rect.width);
      this.viewportHeight.set(rect.height);
    }
  }

  onDrop(event: CdkDragDrop<any>) {
    if (event.previousContainer.id === 'node-panel-list') {
      const nodeType = event.item.data as NodeType;
      const playgroundRect = this.playgroundArea.nativeElement.getBoundingClientRect();
      const offset = this.panOffset();

      let dropX = event.dropPoint.x - playgroundRect.left - offset.x;
      let dropY = event.dropPoint.y - playgroundRect.top - offset.y;

      const position = this.findNonOverlappingPosition(dropX, dropY);

      const newNode: NodeInstance = {
        id: `node-${this.nodeIdCounter++}`,
        type: nodeType,
        position: position,
        data: {},
      };

      this.nodes.update(nodes => [...nodes, newNode]);
    }
  }

  onNodePositionChanged(event: { nodeId: string; position: { x: number; y: number } }) {
    this.nodes.update(nodes => {
      const node = nodes.find((n) => n.id === event.nodeId);
      if (node) {
        node.position = event.position;
      }
      return [...nodes];
    });
  }

  onOutputPortClick(event: { nodeId: string; portIndex: number }) {
    if (!this.isConnecting()) {
      this.isConnecting.set(true);
      this.sourcePort.set({ nodeId: event.nodeId, portIndex: event.portIndex });
    }
  }

  onInputPortClick(event: { nodeId: string; portIndex: number }) {
    const source = this.sourcePort();
    if (this.isConnecting() && source) {
      const newConnection: Connection = {
        id: `connection-${this.connectionIdCounter++}`,
        sourceNodeId: source.nodeId,
        sourcePort: source.portIndex,
        targetNodeId: event.nodeId,
        targetPort: event.portIndex,
      };

      this.connections.update(connections => [...connections, newConnection]);
      this.isConnecting.set(false);
      this.sourcePort.set(null);
      this.tempConnection.set(null);
    }
  }

  onPlaygroundMouseDown(event: MouseEvent) {
    if (event.button === 1) {
      event.preventDefault();
      this.isPanning.set(true);
      this.panStart.set({ x: event.clientX, y: event.clientY });
    }
  }

  onPlaygroundMouseMove(event: MouseEvent) {
    if (this.isPanning()) {
      const start = this.panStart();
      const offset = this.panOffset();
      const deltaX = event.clientX - start.x;
      const deltaY = event.clientY - start.y;

      this.panOffset.set({
        x: offset.x + deltaX,
        y: offset.y + deltaY,
      });

      this.panStart.set({ x: event.clientX, y: event.clientY });
      return;
    }

    const source = this.sourcePort();
    if (this.isConnecting() && source) {
      const sourceNode = this.nodes().find((n) => n.id === source.nodeId);
      if (sourceNode) {
        const playgroundRect = this.playgroundArea.nativeElement.getBoundingClientRect();
        const offset = this.panOffset();
        const sourcePort = this.getPortPosition(
          source.nodeId,
          source.portIndex,
          'output'
        );

        this.tempConnection.set({
          x1: sourcePort.x,
          y1: sourcePort.y,
          x2: event.clientX - playgroundRect.left - offset.x,
          y2: event.clientY - playgroundRect.top - offset.y,
        });
      }
    }
  }

  onPlaygroundMouseUp(event: MouseEvent) {
    if (event.button === 1) {
      this.isPanning.set(false);
    }
  }

  onPlaygroundClick(event: MouseEvent) {
    if (this.isPanning()) {
      return;
    }

    if (this.isConnecting()) {
      this.isConnecting.set(false);
      this.sourcePort.set(null);
      this.tempConnection.set(null);
    }
  }

  getConnectionPath(connection: Connection): string {
    const sourcePos = this.getPortPosition(connection.sourceNodeId, connection.sourcePort, 'output');
    const targetPos = this.getPortPosition(connection.targetNodeId, connection.targetPort, 'input');

    const dx = targetPos.x - sourcePos.x;
    const controlPointOffset = Math.abs(dx) * 0.5;

    return `M ${sourcePos.x} ${sourcePos.y} C ${sourcePos.x + controlPointOffset} ${sourcePos.y}, ${targetPos.x - controlPointOffset} ${targetPos.y}, ${targetPos.x} ${targetPos.y}`;
  }

  getTempConnectionPath(): string {
    const temp = this.tempConnection();
    if (!temp) return '';

    const dx = temp.x2 - temp.x1;
    const controlPointOffset = Math.abs(dx) * 0.5;

    return `M ${temp.x1} ${temp.y1} C ${temp.x1 + controlPointOffset} ${temp.y1}, ${temp.x2 - controlPointOffset} ${temp.y2}, ${temp.x2} ${temp.y2}`;
  }

  private getPortPosition(
    nodeId: string,
    portIndex: number,
    portType: 'input' | 'output'
  ): { x: number; y: number } {
    const node = this.nodes().find((n) => n.id === nodeId);
    if (!node) return { x: 0, y: 0 };

    const portSelector = `[data-node-id="${nodeId}"][data-port-index="${portIndex}"][data-port-type="${portType}"]`;
    const portElement = this.playgroundArea.nativeElement.querySelector(portSelector);

    if (portElement) {
      const playgroundRect = this.playgroundArea.nativeElement.getBoundingClientRect();
      const portRect = portElement.getBoundingClientRect();

      return {
        x: portRect.left + portRect.width / 2 - playgroundRect.left,
        y: portRect.top + portRect.height / 2 - playgroundRect.top,
      };
    }

    return { x: node.position.x, y: node.position.y };
  }

  private findNonOverlappingPosition(x: number, y: number): { x: number; y: number } {
    const nodeWidth = 180;
    const nodeHeight = 100;
    const minDistance = 20;

    let finalX = x;
    let finalY = y;
    let attempts = 0;
    const maxAttempts = 50;

    while (attempts < maxAttempts) {
      let overlapping = false;

      for (const node of this.nodes()) {
        const dx = Math.abs(finalX - node.position.x);
        const dy = Math.abs(finalY - node.position.y);

        if (dx < nodeWidth + minDistance && dy < nodeHeight + minDistance) {
          overlapping = true;
          break;
        }
      }

      if (!overlapping) {
        break;
      }

      finalX += 30;
      finalY += 30;
      attempts++;
    }

    return { x: finalX, y: finalY };
  }
}
