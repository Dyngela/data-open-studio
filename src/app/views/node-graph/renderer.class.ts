import {Component, ElementRef, HostListener} from "@angular/core";
import {Node} from "../../models/node.model";
import {Position} from "../../models/math.model";

@Component({
  template: '' // This will be an abstract base class, so no template
})
export class RendererClass {
  connections: { from: Node; to: Node; fromConnectorId: string, toConnectorId: string, path: string }[] = [];
  draggingConnection = false;
  startX = 0;
  startY = 0;
  currentX = 0;
  currentY = 0;
  currentStartConnector: { node: Node, connectorId: string } | null = null;
  startOutputElement: HTMLElement | null = null;
  svgContainer!: ElementRef<SVGElement>;

  constructor(svg: ElementRef<SVGElement>) {
    this.svgContainer = svg;
  }

  onOutputMouseDown(event: { event: MouseEvent, node: Node, connectorId: string }) {
    event.event.stopPropagation();
    this.draggingConnection = true;
    this.currentStartConnector = { node: event.node, connectorId: event.connectorId };

    const targetElement = event.event.target as HTMLElement;
    const rect = targetElement.getBoundingClientRect();
    const svgRect = this.svgContainer.nativeElement.getBoundingClientRect();

    // Calculate the starting positions relative to the SVG container
    this.startX = rect.left + rect.width / 2 - svgRect.left + this.svgContainer.nativeElement.scrollLeft;
    this.startY = rect.top + rect.height / 2 - svgRect.top + this.svgContainer.nativeElement.scrollTop;

    this.currentX = this.startX;
    this.currentY = this.startY;
  }

  onInputMouseUp(event: { event: MouseEvent, node: Node, connectorId: string }) {
    event.event.stopPropagation();
    if (this.draggingConnection && this.currentStartConnector) {
      const targetElement = event.event.target as HTMLElement;
      const rect = targetElement.getBoundingClientRect();
      const svgRect = this.svgContainer.nativeElement.getBoundingClientRect();

      // Calculate the ending positions relative to the SVG container
      const endX = rect.left + rect.width / 2 - svgRect.left + this.svgContainer.nativeElement.scrollLeft;
      const endY = rect.top + rect.height / 2 - svgRect.top + this.svgContainer.nativeElement.scrollTop;

      this.createConnection(
        this.currentStartConnector,
        { node: event.node, connectorId: event.connectorId },
        { x: this.startX, y: this.startY },
        { x: endX, y: endY }
      );

      // Debugging output to verify end coordinates
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
    const dy = endY - startY;

    // Use the absolute difference to adjust the control point for a smoother curve
    const curveFactor = Math.abs(dx) * 0.3; // Adjust this factor to control the curve smoothness

    // Create a smooth Bézier curve with appropriate control points
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
      const svgRect = this.svgContainer.nativeElement.getBoundingClientRect();
      return {
        x: rect.left + rect.width / 2 - svgRect.left,
        y: rect.top + rect.height / 2 - svgRect.top
      };
    }
    return null;
  }
}
