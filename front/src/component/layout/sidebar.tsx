import {useDrag} from 'react-dnd';
import './sidebar.css'
import React from "react";
import {AppNode} from "../nodes/types/appNode.ts";

interface SidebarItemProps {
    node: Omit<AppNode, 'id'>;
}

const SidebarItem: React.FC<SidebarItemProps> = ({ node }) => {
    const [, drag] = useDrag(() => ({
        type: 'node', // Type for drag-and-drop
        item: { ...node },
    }));

    return (
        <div ref={drag} className="sidebar-item">
            {node.data.label}
        </div>
    );
};

const startNode: Omit<AppNode, 'id'> = {
    type: 'start',
    data: { label: 'Start Node' },
    position: { x: 0, y: 0 },
};

const mapNode: Omit<AppNode, 'id'> = {
    type: 'map',
    data: { label: 'Map Node' },
    position: { x: 0, y: 0 },
};

const dbInputNode: Omit<AppNode, 'id'> = {
    type: 'dbInput',
    data: { label: 'DB Input Node' },
    position: { x: 0, y: 0 },
};

const Sidebar = () => {
    return (
        <aside className="sidebar">
            <SidebarItem node={startNode} />
            <SidebarItem node={mapNode} />
            <SidebarItem node={dbInputNode} />
        </aside>
    );
};

export default Sidebar;
