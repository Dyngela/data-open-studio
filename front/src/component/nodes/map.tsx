import {Handle, Node, NodeProps, Position} from "@xyflow/react";
import {useState} from "react";

type props = Node<{
    label: string;
    type: string;
    inputs: any[];
}, 'map'>;

export default function MapNode(data: NodeProps<props>) {
    const [input, setInput] = useState([])

    return (
        <div
            className="bg-gradient-to-br from-gray-800 to-gray-700 rounded-lg shadow-lg p-4 cursor-pointer">
            <div className="bg-gray-900 border-b border-gray-600 p-2 text-center text-white uppercase font-bold">
                {data.type}
            </div>
            <div>
                {input.map((_, index) => (
                   <Handle type={"target"} position={Position.Left} style={{top: 30 + 10 * index}} />
                ))}
            </div>
        </div>
    );
}
