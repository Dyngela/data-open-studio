import {Component, ElementRef, HostListener, OnInit, ViewChild} from '@angular/core';
import {Graph} from "../../models/graph.model";
import {Node} from "../../models/node.model";
import {ContextMenuItem} from "../../models/context-menu.model";

@Component({
  selector: 'app-node-graph',
  templateUrl: './node-graph.component.html',
  styleUrls: ['./node-graph.component.css']
})
export class NodeGraphComponent implements OnInit {
  graph: Graph = {
    nodes: [],
  };

  @ViewChild('svgContainer', { static: true }) svgContainer!: ElementRef<SVGElement>;

  connections: { from: Node; to: Node; path: string }[] = [];
  draggingConnection = false;
  startX = 0;
  startY = 0;
  currentX = 0;
  currentY = 0;
  startNode: Node | null = null;
  startOutputElement: HTMLElement | null = null;

  contextMenuPosition: { x: number, y: number } | null = null;
  contextMenuItems: ContextMenuItem[] = []

  ngOnInit(): void {
    this.initializeGraph();
  }

  initializeGraph() {
    this.graph.nodes.push(
      {
        id: '1',
        name: 'Start Node',
        inputs: [{ id: 'input1', type: 'input' }, { id: 'input2', type: 'input'}, { id: 'input2', type: 'input'}, { id: 'input2', type: 'input'}],
        outputs: [{ id: 'output1', type: 'output' }, { id: 'output2', type: 'output' }, { id: 'output2', type: 'output' }, { id: 'output2', type: 'output' }],
        position: { x: 100, y: 100 },
        type: 'start'
      },
      {
        id: '2',
        name: 'Connection Node',
        inputs: [{ id: 'input2', type: 'input' }],
        outputs: [{ id: 'output2', type: 'output' }],
        position: { x: 500, y: 500 },
        type: 'start'
      }
    );
  }

  onOutputMouseDown(event: MouseEvent, node: Node) {
    event.stopPropagation();
    this.draggingConnection = true;
    this.startNode = node;
    const rect = (event.target as HTMLElement).getBoundingClientRect();
    const svgRect = this.svgContainer.nativeElement.getBoundingClientRect();
    this.startX = rect.left + rect.width / 2 - svgRect.left + window.scrollX;
    this.startY = rect.top + rect.height / 2 - svgRect.top + window.scrollY;
    this.currentX = this.startX;
    this.currentY = this.startY;
  }

  onInputMouseUp(event: MouseEvent, node: Node) {
    event.stopPropagation();
    if (this.draggingConnection && this.startNode) {
      const rect = (event.target as HTMLElement).getBoundingClientRect();
      const svgRect = this.svgContainer.nativeElement.getBoundingClientRect();
      const endX = rect.left + rect.width / 2 - svgRect.left + window.scrollX;
      const endY = rect.top + rect.height / 2 - svgRect.top + window.scrollY;
      console.log('End dragging at:', endX, endY);
      this.createConnection(this.startNode, node, { x: this.startX, y: this.startY }, { x: endX, y: endY });
    }
    this.resetDraggingState();
  }

  createConnection(fromNode: Node, toNode: Node, start: { x: number; y: number }, end: { x: number; y: number }) {
    const path = this.calculatePath(start.x, start.y, end.x, end.y);
    this.connections.push({ from: fromNode, to: toNode, path });
  }

  calculatePath(startX: number, startY: number, endX: number, endY: number): string {
    const dx = (endX - startX) / 2;
    const dy = (endY - startY) / 2;
    return `M ${startX},${startY} C ${startX + dx},${startY} ${endX - dx},${endY} ${endX},${endY}`;
  }

  resetDraggingState() {
    this.draggingConnection = false;
    this.startNode = null;
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
      this.currentX = event.clientX - svgRect.left + window.scrollX;
      this.currentY = event.clientY - svgRect.top + window.scrollY;
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

  updateConnectionPaths(node: Node) {
    this.connections.forEach(connection => {
      if (connection.from.id === node.id || connection.to.id === node.id) {
        const startX = connection.from.position.x + 28; // Adjust based on the node width/height
        const startY = connection.from.position.y + 28; // Adjust based on the node width/height
        const endX = connection.to.position.x + 28; // Adjust based on the node width/height
        const endY = connection.to.position.y + 28; // Adjust based on the node width/height
        connection.path = this.calculatePath(startX, startY, endX, endY);
      }
    });
  }

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
        label: 'Add Node A',
        action: () => this.addNode('Node A', this.contextMenuPosition)
      },
      {
        label: 'Add Node B',
        action: () => this.addNode('Node B', this.contextMenuPosition)
      },
      {
        label: 'Add Node C',
        action: () => this.addNode('Node C', this.contextMenuPosition)
      }
    ];
  }

  showNodeContextMenu(nodeId: string) {
    this.contextMenuItems = [
      { label: 'Edit Node', action: () => console.log('Edit Node ' + nodeId) },
      { label: 'Delete Node', action: () => console.log('Delete Node ' + nodeId) },
      { label: 'Duplicate Node', action: () => console.log('Duplicate Node ' + nodeId) }
    ];
  }

  addNode(name: string, position: { x: number, y: number } | null) {
    if (!position) return;
    const newNode: Node = {
      id: `${Date.now()}`, // Simple unique ID
      name,
      inputs: [{ id: `${Date.now()}-input`, type: 'input' }],
      outputs: [{ id: `${Date.now()}-output`, type: 'output' }],
      position: { x: position.x - 75, y: position.y - 50 }, // Centered on click position
      type: 'start'
    };
    this.graph.nodes.push(newNode);
    this.contextMenuPosition = null; // Hide the context menu
  }

  @HostListener('document:click', ['$event'])
  onDocumentClick() {
    this.contextMenuPosition = null; // Close context menu on outside click
  }

  @HostListener('document:keydown', ['$event'])
  onDocumentKeyDown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      this.contextMenuPosition = null; // Close context menu on Escape key
    }
  }

  logtest() {
    console.log(this.connections)
  }
}
