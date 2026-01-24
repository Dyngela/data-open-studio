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
  sourcePortType: 'data' | 'flow';
  targetNodeId: string;
  targetPort: number;
  targetPortType: 'data' | 'flow';
  //  DATA ONLY
  dataSchema?: {
    columns: { name: string; type: string }[];
  };
  rowCount?: number;
}

export interface Port {
  nodeId: string;
  portType: 'data' | 'flow';
  portIndex: number;
  position: { x: number; y: number };
  direction: 'input' | 'output';
}

export interface Pipeline {
  id: string;
  name: string;
  nodes: NodeInstance[];
  connections: Connection[]; // Liste unifiée contenant data et flow connections
}
