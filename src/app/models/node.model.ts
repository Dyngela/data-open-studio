export interface Node {
  id: string;
  name: string;
  inputs: Connector[];
  outputs: Connector[];
  position: { x: number; y: number };
  type: string;
}

export interface Connector {
  id: string;
  type: string;
  connectedTo?: string[];
}
