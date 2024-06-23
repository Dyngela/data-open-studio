import {Component} from "@angular/core";
import {Position} from "../../models/math.model";
import {ContextMenuItem} from "../../models/context-menu.model";
import {Node} from "../../models/node.model";
import {GraphService} from "../../services/graph.service";

@Component({
  template: '' // This will be an abstract base class, so no template
})
export class ContextMenuClass {

  contextMenuPosition: Position | null = null;
  contextMenuItems: ContextMenuItem[] = []

  constructor(private graphService: GraphService) {}

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
    this.graphService.getGraph().nodes.push(newNode);
    this.contextMenuPosition = null; // Hide the context menu
  }

}
