import {Injectable, signal} from '@angular/core';
import {
  Connection,
  Direction,
  NodeInstance,
  NodeSizeFn,
  NodeType,
  PortPositionFn,
  PortType,
  TempConnection,
} from './node.type';
import {JobWithNodes, Node as ApiNode} from '../api/job.type';

@Injectable({ providedIn: 'root' })
export class NodeGraphService {
  readonly nodes = signal<NodeInstance[]>([]);
  readonly connections = signal<Connection[]>([]);

  private nodeIdCounter = 0;
  private connectionIdCounter = 0;

  readonly NODE_DIMENSIONS = {
    width: 180,
    headerPadding: 12,
    bodyPadding: 16,
    portSize: 16,
    portBorder: 2,
    portGap: 12,
    portOffset: 10,
    estimatedHeight: 140,
  };

  readonly COLLISION_PADDING = 8;

  createNode(type: NodeType, position: { x: number; y: number }): NodeInstance {
    const node: NodeInstance = {
      id: `node-${this.nodeIdCounter++}`,
      type,
      position,
      config: {},
      status: 'idle',
    };
    this.nodes.update(nodes => [...nodes, node]);
    return node;
  }

  updateNodeConfig(nodeId: string, config: Record<string, any>): void {
    this.nodes.update(nodes =>
      nodes.map(node =>
        node.id === nodeId ? { ...node, config: { ...node.config, ...config } } : node,
      ),
    );
  }

  updateNodePosition(nodeId: string, position: { x: number; y: number }): void {
    this.nodes.update(nodes =>
      nodes.map(node => (node.id === nodeId ? { ...node, position } : node)),
    );
  }

  getNodeById(nodeId: string): NodeInstance | undefined {
    return this.nodes().find(n => n.id === nodeId);
  }

  createConnection(
    source: { nodeId: string; portIndex: number; portType: PortType },
    target: { nodeId: string; portIndex: number; portType: PortType },
  ): Connection {
    const connection: Connection = {
      id: `connection-${this.connectionIdCounter++}`,
      sourceNodeId: source.nodeId,
      sourcePort: source.portIndex,
      sourcePortType: source.portType,
      targetNodeId: target.nodeId,
      targetPort: target.portIndex,
      targetPortType: target.portType,
    };
    this.connections.update(conns => [...conns, connection]);
    return connection;
  }

  loadFromJob(job: JobWithNodes): void {
    if (job.nodes && job.nodes.length > 0) {
      const nodeInstances: NodeInstance[] = job.nodes.map(apiNode => ({
        id: `node-${apiNode.id}`,
        type: this.getNodeTypeFromApiType(apiNode.type),
        position: { x: apiNode.xpos, y: apiNode.ypos },
        config: (apiNode.data as Record<string, any>) || {},
        status: 'idle' as const,
      }));
      this.nodes.set(nodeInstances);
      this.nodeIdCounter = Math.max(...job.nodes.map(n => n.id)) + 1;
    }
  }

  toApiNodes(jobId: number): ApiNode[] {
    return this.nodes().map(node => ({
      id: parseInt(node.id.replace('node-', ''), 10) || 0,
      type: this.getApiTypeFromNodeType(node.type.id),
      name: node.type.label,
      xpos: node.position.x,
      ypos: node.position.y,
      inputPort: [],
      outputPort: [],
      data: node.config || {},
      jobId,
    }));
  }

  getNodeTypeFromApiType(apiType: string): NodeType {
    const typeMap: Record<string, NodeType> = {
      start: {
        id: 'start', label: 'Start', icon: 'pi-play', type: 'start',
        hasFlowInput: false, hasFlowOutput: true, hasDataInput: false, hasDataOutput: false,
      },
      db_input: {
        id: 'db-input', label: 'DB Input', icon: 'pi-database', type: 'input',
        hasFlowInput: true, hasFlowOutput: true, hasDataInput: false, hasDataOutput: true,
      },
      db_output: {
        id: 'db-output', label: 'DB Output', icon: 'pi-database', type: 'output',
        hasFlowInput: true, hasFlowOutput: true, hasDataInput: true, hasDataOutput: false,
      },
      map: {
        id: 'transform', label: 'Transform', icon: 'pi-cog', type: 'process',
        hasFlowInput: true, hasFlowOutput: true, hasDataInput: true, hasDataOutput: true,
      },
    };
    return typeMap[apiType] || typeMap['start'];
  }

  getApiTypeFromNodeType(nodeTypeId: string): 'start' | 'db_input' | 'db_output' | 'map' {
    const typeMap: Record<string, 'start' | 'db_input' | 'db_output' | 'map'> = {
      start: 'start',
      'db-input': 'db_input',
      'db-output': 'db_output',
      transform: 'map',
    };
    return typeMap[nodeTypeId] || 'start';
  }

  calculatePortPosition(
    node: NodeInstance,
    portIndex: number,
    portType: Direction,
    connectionType: PortType,
    panOffset: { x: number; y: number },
  ): { x: number; y: number } {
    const dim = this.NODE_DIMENSIONS;
    const estimatedHeaderHeight = dim.headerPadding * 2 + 16;

    if (connectionType === 'flow') {
      const portY = estimatedHeaderHeight / 2;
      const portCenterOffset = dim.portSize / 2;
      const portX =
        portType === 'input'
          ? -dim.portOffset + portCenterOffset
          : dim.width + dim.portOffset - portCenterOffset;

      return {
        x: node.position.x + portX + panOffset.x,
        y: node.position.y + portY + panOffset.y,
      };
    }

    // Data ports
    const portCount =
      portType === 'input'
        ? (node.type.hasDataInput ? 1 : 0)
        : (node.type.hasDataOutput ? 1 : 0);

    const bodyTop = estimatedHeaderHeight + dim.bodyPadding;
    const totalPortsHeight = portCount * dim.portSize + (portCount - 1) * dim.portGap;
    const portsStartY = bodyTop - totalPortsHeight / 2;
    const portY = portsStartY + portIndex * (dim.portSize + dim.portGap) + dim.portSize / 2;

    const portCenterOffset = dim.portSize / 2;
    const portX =
      portType === 'input'
        ? -dim.portOffset + portCenterOffset
        : dim.width + dim.portOffset - portCenterOffset;

    return {
      x: node.position.x + portX + panOffset.x,
      y: node.position.y + portY + panOffset.y,
    };
  }

  findNonOverlappingPosition(x: number, y: number): { x: number; y: number } {
    const nodeWidth = this.NODE_DIMENSIONS.width;
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

      if (!overlapping) break;

      finalX += 30;
      finalY += 30;
      attempts++;
    }

    return { x: finalX, y: finalY };
  }

  resolveCollision(
    nodeId: string,
    desiredX: number,
    desiredY: number,
    nodeSizeFn: NodeSizeFn,
  ): { x: number; y: number } {
    let x = desiredX;
    let y = desiredY;
    const padding = this.COLLISION_PADDING;
    const movingSize = nodeSizeFn(nodeId);

    for (let i = 0; i < 10; i++) {
      const movingRect = {
        x: x - padding,
        y: y - padding,
        width: movingSize.width + padding * 2,
        height: movingSize.height + padding * 2,
      };

      let adjusted = false;

      for (const node of this.nodes()) {
        if (node.id === nodeId) continue;
        const size = nodeSizeFn(node.id);
        const otherRect = {
          x: node.position.x,
          y: node.position.y,
          width: size.width,
          height: size.height,
        };

        const overlapX =
          Math.min(movingRect.x + movingRect.width, otherRect.x + otherRect.width) -
          Math.max(movingRect.x, otherRect.x);
        const overlapY =
          Math.min(movingRect.y + movingRect.height, otherRect.y + otherRect.height) -
          Math.max(movingRect.y, otherRect.y);

        if (overlapX > 0 && overlapY > 0) {
          if (overlapX < overlapY) {
            const movingCenterX = movingRect.x + movingRect.width / 2;
            const otherCenterX = otherRect.x + otherRect.width / 2;
            if (movingCenterX < otherCenterX) {
              x -= overlapX;
            } else {
              x += overlapX;
            }
          } else {
            const movingCenterY = movingRect.y + movingRect.height / 2;
            const otherCenterY = otherRect.y + otherRect.height / 2;
            if (movingCenterY < otherCenterY) {
              y -= overlapY;
            } else {
              y += overlapY;
            }
          }

          adjusted = true;
          break;
        }
      }

      if (!adjusted) return { x, y };
    }

    return { x, y };
  }

  getConnectionPath(connection: Connection, portPositionFn: PortPositionFn): string {
    const sourcePos = portPositionFn(
      connection.sourceNodeId,
      connection.sourcePort,
      Direction.OUTPUT,
      connection.sourcePortType,
    );
    const targetPos = portPositionFn(
      connection.targetNodeId,
      connection.targetPort,
      Direction.INPUT,
      connection.targetPortType,
    );

    const dx = targetPos.x - sourcePos.x;
    const controlPointOffset = Math.abs(dx) * 0.5;

    return `M ${sourcePos.x} ${sourcePos.y} C ${sourcePos.x + controlPointOffset} ${sourcePos.y}, ${
      targetPos.x - controlPointOffset
    } ${targetPos.y}, ${targetPos.x} ${targetPos.y}`;
  }

  getTempConnectionPath(temp: TempConnection | null): string {
    if (!temp) {
      return '';
    }
    const dx = temp.x2 - temp.x1;
    const controlPointOffset = Math.abs(dx) * 0.5;

    return `M ${temp.x1} ${temp.y1} C ${temp.x1 + controlPointOffset} ${temp.y1}, ${
      temp.x2 - controlPointOffset
    } ${temp.y2}, ${temp.x2} ${temp.y2}`;
  }

  getConnectionKind(connection: Connection): PortType {
    return connection.sourcePortType || 'data';
  }

  reset(): void {
    this.nodes.set([]);
    this.connections.set([]);
    this.nodeIdCounter = 0;
    this.connectionIdCounter = 0;
  }
}
