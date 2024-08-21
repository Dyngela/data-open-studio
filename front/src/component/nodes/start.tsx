import {Handle, Node, NodeProps, Position} from '@xyflow/react';
import React from "react";

type NumberNode = Node<{ }, 'start'>;

export default function StartNode({ }: NodeProps<NumberNode>) {
    return (
        <div
            className="bg-gradient-to-br bg-gray-700 rounded-lg shadow-lg p-4 cursor-pointer">
                Start
            <Handle
                type="source"
                position={Position.Right}
                id={"start-output"}
                className={"bg-amber-400 h-2 top-3"}
            />
        </div>
    );
}

