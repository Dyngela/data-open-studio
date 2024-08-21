import type {Node, BuiltInNode} from '@xyflow/react';

export type StartNode = Node<{ label: string }, 'start'>;
export type MapNode = Node<{ label: string }, 'map'>;
export type DbInputNode = Node<{ label: string }, 'dbInput'>;

export type AppNode = BuiltInNode | StartNode | MapNode | DbInputNode;


