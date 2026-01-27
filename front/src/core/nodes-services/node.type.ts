export interface NodeType {
  id: string;
  label: string;
  icon?: string;
  color?: string;
  hasDataInput: boolean;
  hasDataOutput: boolean;
  hasFlowInput: boolean;
  hasFlowOutput: boolean;
  type: 'start' | 'input' | 'output' | 'process';
}

export interface NodeInstance {
  id: number;
  type: NodeType;
  position: { x: number; y: number };
  config?: Record<string, any>;
  status?: 'idle' | 'waiting' | 'running' | 'success' | 'error';
}

export interface Connection {
  sourceNodeId: number;
  sourcePort: number;
  sourcePortType: PortType;
  targetNodeId: number;
  targetPort: number;
  targetPortType: PortType;
}

/** Callback to retrieve a node's rendered size from the DOM */
export type NodeSizeFn = (nodeId: number) => { width: number; height: number };

/** Callback to retrieve a port's rendered position from the DOM */
export type PortPositionFn = (
  nodeId: number,
  portIndex: number,
  portType: Direction,
  connectionType: PortType,
) => { x: number; y: number };

/** Coordinates for a temporary in-progress connection line */
export interface TempConnection {
  x1: number;
  y1: number;
  x2: number;
  y2: number;
}

export enum Direction {
  INPUT = 'input',
  OUTPUT = 'output'
}

export enum PortType {
  DATA = 'data',
  FLOW = 'flow'
}
