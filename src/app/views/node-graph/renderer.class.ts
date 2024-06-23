import {Component, ElementRef, inject} from "@angular/core";
import {Node} from "../../models/node.model";
import {Position} from "../../models/math.model";
import {ContextMenuItem} from "../../models/context-menu.model";
import {GraphService} from "../../services/graph.service";
import {NodeTypes} from "../../enums/node.enum";

@Component({
  template: ''
})
export class RendererClass {
  graphService = inject(GraphService);

  connections: { from: Node; to: Node; fromConnectorId: string, toConnectorId: string, path: string }[] = [];
  draggingConnection = false;
  startX = 0;
  startY = 0;
  currentX = 0;
  currentY = 0;
  currentStartConnector: { node: Node, connectorId: string } | null = null;
  startOutputElement: HTMLElement | null = null;
  contextMenuPosition: Position | null = null;
  contextMenuItems: ContextMenuItem[] = []

  onRightClick(event: MouseEvent) {
    event.preventDefault();
    this.contextMenuPosition = { x: event.clientX, y: event.clientY };
    this.showGraphContextMenu();
  }

  onNodeRightClick(event: { nodeId: string, x: number, y: number }) {
    this.contextMenuPosition = { x: event.x, y: event.y };
    this.showNodeContextMenu(event.nodeId);
  }

  showGraphContextMenu() {
    this.contextMenuItems = [
      {
        label: 'Add Start Node',
        action: () => this.addNode('Node A', NodeTypes.START)
      },
      {
        label: 'Add DB Connection Node',
        action: () => this.addNode('Node B', NodeTypes.DB_CONNECTION)
      },
      {
        label: 'Add End Node',
        action: () => this.addNode('Node C', NodeTypes.END)
      }
    ];
  }

  showNodeContextMenu(nodeId: string) {
    this.contextMenuItems = [
      { label: 'Edit Node', action: () => this.editNode(nodeId) },
      { label: 'Delete Node', action: () => this.deleteNode(nodeId) },
      { label: 'Duplicate Node', action: () => console.log('Duplicate Node ' + nodeId) }
    ];
  }

  private deleteNode(nodeId: string) {
    this.connections = this.connections.filter(connection => connection.from.id !== nodeId && connection.to.id !== nodeId);
    this.graphService.getGraph().nodes.forEach(node => {
      node.inputs = node.inputs.filter(input => !input.connectedTo || !input.connectedTo.includes(nodeId));
      node.outputs = node.outputs.filter(output => !output.connectedTo || !output.connectedTo.includes(nodeId));
    });
    this.graphService.getGraph().nodes = this.graphService.getGraph().nodes.filter(node => node.id !== nodeId);
  }

  editNode(node: Node | string) {
    console.log('Edit Node ' + node);
  }

  addNode(name: string, type: NodeTypes) {
    if (!this.contextMenuPosition) return;
    const newNode: Node = {
      id: `${Date.now()}`, // Simple unique ID
      name,
      inputs: [{ id: `${Date.now()}-input`, type: 'input' }],
      outputs: [{ id: `${Date.now()}-output`, type: 'output' }],
      position: { x: this.contextMenuPosition.x - 75, y: this.contextMenuPosition.y - 50 }, // Centered on click position
      type: type
    };
    this.graphService.getGraph().nodes.push(newNode);
    this.contextMenuPosition = null; // Hide the context menu
  }

  onOutputMouseDown(event: { event: MouseEvent, node: Node, connectorId: string }) {
    event.event.stopPropagation();
    this.draggingConnection = true;
    this.currentStartConnector = { node: event.node, connectorId: event.connectorId };

    const targetElement = event.event.target as HTMLElement;
    const rect = targetElement.getBoundingClientRect();
    const svgRect = this.graphService.getSvgContainer().nativeElement.getBoundingClientRect();

    // Calculate the starting positions relative to the SVG container
    this.startX = rect.left + rect.width / 2 - svgRect.left + this.graphService.getSvgContainer().nativeElement.scrollLeft;
    this.startY = rect.top + rect.height / 2 - svgRect.top + this.graphService.getSvgContainer().nativeElement.scrollTop;

    this.currentX = this.startX;
    this.currentY = this.startY;
  }

  onInputMouseUp(event: { event: MouseEvent, node: Node, connectorId: string }) {
    event.event.stopPropagation();
    if (this.draggingConnection && this.currentStartConnector) {
      const targetElement = event.event.target as HTMLElement;
      const rect = targetElement.getBoundingClientRect();
      const svgRect = this.graphService.getSvgContainer().nativeElement.getBoundingClientRect();
      const endX = rect.left + rect.width / 2 - svgRect.left + this.graphService.getSvgContainer().nativeElement.scrollLeft;
      const endY = rect.top + rect.height / 2 - svgRect.top + this.graphService.getSvgContainer().nativeElement.scrollTop;

      this.createConnection(
        this.currentStartConnector,
        { node: event.node, connectorId: event.connectorId },
        { x: this.startX, y: this.startY },
        { x: endX, y: endY }
      );
    }
    this.resetDraggingState();
  }

  createConnection(from: { node: Node, connectorId: string }, to: { node: Node, connectorId: string }, start: Position, end: Position) {
    const path = this.calculatePath(start.x, start.y, end.x, end.y);
    let input = from.node.outputs.find(output => output.id === from.connectorId);
    let output = to.node.inputs.find(input => input.id === to.connectorId);
    if (input && output) {
      input.connectedTo = input.connectedTo || [];
      input.connectedTo.push(output.id);
      output.connectedTo = output.connectedTo || [];
      output.connectedTo.push(input.id);
      this.connections.push({ from: from.node, to: to.node, fromConnectorId: from.connectorId, toConnectorId: to.connectorId, path: path });
    }
  }

  calculatePath(startX: number, startY: number, endX: number, endY: number): string {
    const dx = endX - startX;
    const curveFactor = Math.abs(dx) * 0.3; // Adjust this factor to control the curve smoothness
    return `M${startX},${startY} C ${startX + curveFactor},${startY} ${endX - curveFactor},${endY} ${endX},${endY}`;
  }

  resetDraggingState() {
    this.draggingConnection = false;
    this.currentStartConnector = null;
    this.startOutputElement = null;
    this.startX = 0;
    this.startY = 0;
    this.currentX = 0;
    this.currentY = 0;
  }

  updateAllConnections() {
    this.connections.forEach(connection => {
      const fromPosition = this.getPortPosition(connection.fromConnectorId);
      const toPosition = this.getPortPosition(connection.toConnectorId);
      if (fromPosition && toPosition) {
        connection.path = this.calculatePath(fromPosition.x, fromPosition.y, toPosition.x, toPosition.y);
      }
    });
  }

  updateConnectionPaths(node: Node) {
    this.connections.forEach(connection => {
      if (connection.from.id === node.id || connection.to.id === node.id) {
        const start : { x: number, y: number } | null = this.getPortPosition(connection.fromConnectorId);
        const end : { x: number, y: number } | null = this.getPortPosition(connection.toConnectorId);
        if (start && end) {
          connection.path = this.calculatePath(start?.x, start?.y, end.x, end?.y);
        }
      }
    });
  }

  getPortPosition(portId: string): { x: number, y: number } | null {
    const portElement = document.getElementById(portId);
    if (portElement) {
      const rect = portElement.getBoundingClientRect();
      const svgRect = this.graphService.getSvgContainer().nativeElement.getBoundingClientRect();
      return {
        x: rect.left + rect.width / 2 - svgRect.left,
        y: rect.top + rect.height / 2 - svgRect.top
      };
    }
    return null;
  }
}
