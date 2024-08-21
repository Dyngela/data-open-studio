import React, { useCallback } from 'react';
import {
    ReactFlow,
    Background,
    Controls,
    useNodesState,
    useEdgesState, addEdge,
    Node, MiniMap,
} from '@xyflow/react';
import { useDrop } from 'react-dnd';
import Sidebar from "./sidebar.tsx";
import './layout.css'
import '@xyflow/react/dist/style.css';
import BottomBar from "./bottomBar.tsx";
import {v4 as uuidv4} from 'uuid';
import {AppNode} from "../nodes/types/appNode.ts";
import {nodeTypes} from "../nodes";

type DraggedItem = {
    type: string;
};

const nodeColor = (node: Node) => {
    switch (node.type) {
        case 'start':
            return '#6ede87';
        case 'output':
            return '#6865A5';
        default:
            return '#ff0072';
    }
};

const initialNodes: AppNode[] = [
    {
        id: '1',
        type: 'start',
        position: { x: 100, y: 100 },
        data: { label: 'Start' },
    },
    {
        id: '2',
        type: 'output',
        position: { x: 300, y: 100 },
        data: { label: 'Output' },
    },
];

const Layout: React.FC = () => {
    const [nodes, setNodes, onNodesChange] = useNodesState<AppNode>(initialNodes);
    const [edges, setEdges, onEdgesChange] = useEdgesState([]);

    const onDrop = useCallback(
        (item: DraggedItem, monitor: any) => {
            const offset = monitor.getSourceClientOffset();
            if (!offset) return;

            // Adjust the position relative to the ReactFlow container
            const reactFlowBounds = document.querySelector('.react-flow')?.getBoundingClientRect();
            if (!reactFlowBounds) return;

            const position = {
                x: offset.x - reactFlowBounds.left,
                y: offset.y - reactFlowBounds.top,
            };

            const newNode: AppNode = {
                id: uuidv4(),
                // @ts-ignore
                type: item.type,
                position,
                data: { label: `Node ${item.type}` },
            };

            setNodes((nds) => [...nds, newNode]);
        },
        [setNodes]
    );

    const [, drop] = useDrop(() => ({
        accept: 'node',
        drop: onDrop,
        collect: (monitor) => ({
            isOver: monitor.isOver(),
        }),
    }));

    const exportGraph = () => {
        const graphData = {
            nodes: nodes.map(node => ({
                id: node.id,
                type: node.type,
                position: node.position,
                data: node.data,
            })),
            edges: edges.map(edge => ({
                id: edge.id,
                source: edge.source,
                target: edge.target,
                type: edge.type,
                animated: edge.animated,
                style: edge.style,
            })),
        };

        const blob = new Blob([JSON.stringify(graphData, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'graph.json';
        a.click();
        URL.revokeObjectURL(url);
    };

    // @ts-ignore
    const exec = () => {
        // Empty function for now
    };

    return (
            <div className="dndflow">
                <Sidebar />
                <div ref={drop} className="canva">
                    <ReactFlow
                        nodeTypes={nodeTypes}
                        nodes={nodes}
                        edges={edges}
                        onNodesChange={onNodesChange}
                        onEdgesChange={onEdgesChange}
                        onConnect={(params) => setEdges((eds) => addEdge(params, eds))}
                        fitView
                    >
                        <Background variant={'lines' as any}/>
                        <MiniMap nodeColor={nodeColor} nodeStrokeWidth={3} zoomable pannable />
                        <Controls position={"top-right"} />
                    </ReactFlow>
                </div>
                <div>
                    <BottomBar onExport={exportGraph} onExec={exec} />
                </div>
            </div>
    );
};

export default Layout;
