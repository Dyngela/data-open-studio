import { Node } from './node.model';

export interface Graph {
  nodes: Node[];
  connections: Connection[];
}

export interface Connection {
  fromNode: string;
  fromConnector: string;
  toNode: string;
  toConnector: string;
}
