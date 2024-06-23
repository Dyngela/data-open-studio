import {NodeTypes} from "../enums/node.enum";

export interface Node {
  id: string;
  name: string;
  inputs: Connector[];
  outputs: Connector[];
  position: { x: number; y: number };
  type: NodeTypes;
}

export interface Connector {
  id: string;
  type: string;
  connectedTo?: string[];
}
