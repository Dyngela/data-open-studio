import {AfterViewInit, Component, ElementRef, HostListener, OnInit, ViewChild} from '@angular/core';
import {Graph} from "../../models/graph.model";
import {Node} from "../../models/node.model";
import {ContextMenuItem} from "../../models/context-menu.model";
import {Position} from "../../models/math.model";
import {ContextMenuClass} from "./context-menu.class";
import {GraphService} from "../../services/graph.service";
import {RendererClass} from "./renderer.class";

@Component({
  selector: 'app-node-graph',
  templateUrl: './node-graph.component.html',
  styleUrls: ['./node-graph.component.css']
})
export class NodeGraphComponent implements AfterViewInit {

  graph: Graph = { nodes: [] };
  contextMenu: ContextMenuClass;
  renderer: RendererClass;
  @ViewChild('svgContainer', { static: true }) svgContainer!: ElementRef<SVGElement>;
  draggingConnection = false;
  currentStartConnector: { node: Node, connectorId: string } | null = null;
  startX = 0;
  startY = 0;
  currentX = 0;
  currentY = 0;
  startOutputElement: HTMLElement | null = null;

  constructor(graphService: GraphService) {
    this.contextMenu = new ContextMenuClass(graphService);
    this.renderer = new RendererClass();
    this.graph = graphService.getGraph();
    this.initializeGraph();
  }

  ngAfterViewInit(): void {
    const s = this.getPortPosition('output1')
    const e = this.getPortPosition('2input2')
    if (s == null || e == null) {
      return
    }
    this.renderer.connections.push({
      from: this.graph.nodes[0],
      to: this.graph.nodes[1],
      fromConnectorId: 'output1',
      toConnectorId: '2input2',
      path: ""
    })
    this.updateAllConnections()
  }

  initializeGraph() {
    this.graph.nodes.push(
      {
        id: '1',
        name: 'Start Node',
        inputs: [],
        outputs: [{ id: 'output1', type: 'output', connectedTo: ['2input2']}],
        position: { x: 100, y: 100 },
        type: 'start'
      },
      {
        id: '2',
        name: 'Connection Node',
        inputs: [{ id: '2input2', type: 'input', connectedTo: ['output1']}],
        outputs: [],
        position: { x: 500, y: 500 },
        type: 'start'
      }
    );


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

    // Debugging output to verify initial coordinates
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
      this.renderer.connections.push({ from: from.node, to: to.node, fromConnectorId: from.connectorId, toConnectorId: to.connectorId, path: path });
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

  @HostListener('window:mousemove', ['$event'])
  onDocumentMouseMove(event: MouseEvent) {
    if (this.draggingConnection) {
      const svgRect = this.svgContainer.nativeElement.getBoundingClientRect();
      this.currentX = event.clientX - svgRect.left;
      this.currentY = event.clientY - svgRect.top;
    }
  }

  @HostListener('document:mouseup', ['$event'])
  onDocumentMouseUp(event: MouseEvent) {
    if (this.draggingConnection) {
      this.resetDraggingState();
    }
  }

  onNodeMove(event: { id: string, x: number, y: number }) {
    const node = this.graph.nodes.find(n => n.id === event.id);
    if (node) {
      node.position = { x: event.x, y: event.y };
      this.updateConnectionPaths(node);
    }
  }

  updateAllConnections() {
    this.renderer.connections.forEach(connection => {
      const fromPosition = this.getPortPosition(connection.fromConnectorId);
      const toPosition = this.getPortPosition(connection.toConnectorId);
      if (fromPosition && toPosition) {
        connection.path = this.calculatePath(fromPosition.x, fromPosition.y, toPosition.x, toPosition.y);
      }
    });
  }

  updateConnectionPaths(node: Node) {
    this.renderer.connections.forEach(connection => {
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

  @HostListener('document:click', ['$event'])
  onDocumentClick() {
    this.contextMenu.contextMenuPosition = null; // Close context menu on outside click
  }

  @HostListener('document:keydown', ['$event'])
  onDocumentKeyDown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      this.contextMenu.contextMenuPosition = null; // Close context menu on Escape key
    }
  }

  logtest() {
    console.log(this.renderer.connections)
  }
}
