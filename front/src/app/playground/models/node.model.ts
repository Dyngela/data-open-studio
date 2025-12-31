export interface NodeType {
  id: string;
  label: string;
  icon?: string;
  color?: string;
  inputs: number;
  outputs: number;
}

export interface NodeInstance {
  id: string;
  type: NodeType;
  position: { x: number; y: number };
  data?: any;
}

export interface Connection {
  id: string;
  sourceNodeId: string;
  sourcePort: number;
  targetNodeId: string;
  targetPort: number;
}

export interface Port {
  nodeId: string;
  portIndex: number;
  position: { x: number; y: number };
}
