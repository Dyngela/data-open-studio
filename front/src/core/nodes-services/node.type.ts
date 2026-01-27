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
  id: string;
  type: NodeType;
  position: { x: number; y: number };
  config?: Record<string, any>;
  status?: 'idle' | 'waiting' | 'running' | 'success' | 'error';
}

export interface Connection {
  id: string;
  sourceNodeId: string;
  sourcePort: number;
  sourcePortType: PortType;
  targetNodeId: string;
  targetPort: number;
  targetPortType: PortType;
  //  DATA ONLY
  dataSchema?: {
    columns: { name: string; type: string }[];
  };
  rowCount?: number;
}

export interface Port {
  nodeId: string;
  portType: PortType;
  portIndex: number;
  position: { x: number; y: number };
  direction: Direction;
}

export interface Pipeline {
  id: string;
  name: string;
  nodes: NodeInstance[];
  connections: Connection[]; // Liste unifiée contenant data et flow connections
}

/** Callback to retrieve a node's rendered size from the DOM */
export type NodeSizeFn = (nodeId: string) => { width: number; height: number };

/** Callback to retrieve a port's rendered position from the DOM */
export type PortPositionFn = (
  nodeId: string,
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
