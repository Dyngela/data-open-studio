import type { NodeTypes } from '@xyflow/react';

import StartNode from "./start.tsx";
import MapNode from "./map.tsx";
import DbInputNode from "./dbInputNode.tsx";

export const nodeTypes = {
  'start': StartNode,
  'map': MapNode,
  'dbInput': DbInputNode,
} satisfies NodeTypes;

