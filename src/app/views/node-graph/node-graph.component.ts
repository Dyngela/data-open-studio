import {AfterViewInit, Component, ElementRef, HostListener, inject, OnInit, ViewChild} from '@angular/core';
import {Graph} from "../../models/graph.model";
import {GraphService} from "../../services/graph.service";
import {RendererClass} from "./renderer.class";
import {NodeTypes} from "../../enums/node.enum";
import {Node} from "../../models/node.model";

@Component({
  selector: 'app-node-graph',
  templateUrl: './node-graph.component.html',
  styleUrls: ['./node-graph.component.css']
})
export class NodeGraphComponent implements AfterViewInit {
  protected readonly NodeTypes = NodeTypes;

  graphService = inject(GraphService)
  graph: Graph = { nodes: [] };
  renderer!: RendererClass;
  @ViewChild('svgContainer') svgContainer!: ElementRef<SVGElement>;

  constructor() {
    this.renderer = new RendererClass();
    this.graph = this.graphService.getGraph();
  }

  ngAfterViewInit(): void {
    this.graphService.setSvgContainer(this.svgContainer);

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

  onNodeMove(event: { id: string, x: number, y: number }) {
    const node = this.graph.nodes.find(n => n.id === event.id);
    if (node) {
      node.position = { x: event.x, y: event.y };
      this.renderer.updateConnectionPaths(node);
    }
  }

  @HostListener('document:click', ['$event'])
  onDocumentClick() {
    this.renderer.contextMenuPosition = null; // Close context menu on outside click
  }

  @HostListener('document:keydown', ['$event'])
  onDocumentKeyDown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      this.renderer.contextMenuPosition = null; // Close context menu on Escape key
    }
  }

  @HostListener('window:mousemove', ['$event'])
  onDocumentMouseMove(event: MouseEvent) {
    if (this.renderer.draggingConnection) {
      const svgRect = this.svgContainer.nativeElement.getBoundingClientRect() as DOMRect;
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

  onNodeDoubleClick(node: Node) {
    this.renderer.editNode(node);

  }

  logtest() {
    console.log(this.renderer.connections)
  }

}
