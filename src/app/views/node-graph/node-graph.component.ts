import {Component, HostListener, OnInit} from '@angular/core';
import {Connection, Graph} from "../../models/graph.model";
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
    connections: [],
  };

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
        position: { x: 300, y: 300 },
        type: 'start'
      }
    );
  }

  onNodeMove(event: { id: string, x: number, y: number }) {
    const node = this.graph.nodes.find(n => n.id === event.id);
    if (node) {
      node.position = { x: event.x, y: event.y };
    }
  }

  onConnectStart(event: { nodeId: string, connectorId: string }) {
    // Logic for starting a connection
  }

  onConnectEnd(event: { nodeId: string, connectorId: string }) {
    // Logic for ending a connection
  }

  getNode(nodeId: string): Node {
    return this.graph.nodes.find(node => node.id === nodeId) as Node;
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

}
