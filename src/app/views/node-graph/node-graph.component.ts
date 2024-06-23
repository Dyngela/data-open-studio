import {AfterViewInit, Component, ElementRef, HostListener, inject, OnInit, ViewChild} from '@angular/core';
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
export class NodeGraphComponent implements AfterViewInit, OnInit {
  graphService = inject(GraphService)
  graph: Graph = { nodes: [] };
  contextMenu: ContextMenuClass;
  renderer!: RendererClass;
  @ViewChild('svgContainer', { static: true }) svgContainer!: ElementRef<SVGElement>;

  constructor() {
    this.contextMenu = new ContextMenuClass(this.graphService);
    this.graph = this.graphService.getGraph();
    this.initializeGraph();
  }

  ngOnInit(): void {
    this.renderer = new RendererClass(this.svgContainer);
  }

  ngAfterViewInit(): void {
    const s = this.renderer.getPortPosition('output1')
    const e = this.renderer.getPortPosition('2input2')
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
    this.renderer.updateAllConnections()
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

  onNodeMove(event: { id: string, x: number, y: number }) {
    const node = this.graph.nodes.find(n => n.id === event.id);
    if (node) {
      node.position = { x: event.x, y: event.y };
      this.renderer.updateConnectionPaths(node);
    }
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

  @HostListener('window:mousemove', ['$event'])
  onDocumentMouseMove(event: MouseEvent) {
    if (this.renderer.draggingConnection) {
      const svgRect = this.svgContainer.nativeElement.getBoundingClientRect();
      this.renderer.currentX = event.clientX - svgRect.left;
      this.renderer.currentY = event.clientY - svgRect.top;
    }
  }

  @HostListener('document:mouseup', ['$event'])
  onDocumentMouseUp(event: MouseEvent) {
    if (this.renderer.draggingConnection) {
      this.renderer.resetDraggingState();
    }
  }


  logtest() {
    console.log(this.renderer.connections)
  }
}
