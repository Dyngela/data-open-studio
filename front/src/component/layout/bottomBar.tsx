import React, { useCallback, useRef, useState } from 'react';
import Context from "../context/context.tsx";


interface BottomBarProps {
    onExport: () => void;
    onExec: () => void;
}

const TabSelector: React.FC<{
    isActive: boolean;
    children: React.ReactNode;
    onClick: () => void;
}> = ({ isActive, children, onClick }) => (
    <div
        className={`px-4 py-2 font-medium text-sm cursor-pointer ${
            isActive
                ? 'border-b-2 border-blue-500 text-blue-500'
                : 'text-gray-400 hover:text-blue-500'
        }`}
        onClick={onClick}
    >
        {children}
    </div>
);

const TabPanel: React.FC<{ hidden: boolean; children: React.ReactNode }> = ({ hidden, children }) => {
    return <div className={`${hidden ? 'hidden' : 'block'}`}>{children}</div>;
};

const BottomBar: React.FC<BottomBarProps> = ({ onExport, onExec }) => {
    const [height, setHeight] = useState(250);  // Default height of the bottom bar
    const [selectedTab, setSelectedTab] = useState('account');
    const barRef = useRef<HTMLDivElement>(null);

    const startResize = useCallback((e: React.MouseEvent) => {
        e.preventDefault();

        const handleMouseMove = (e: MouseEvent) => {
            requestAnimationFrame(() => {
                if (barRef.current) {
                    const newHeight = window.innerHeight - e.clientY;
                    setHeight(Math.max(newHeight, 250)); // Minimum height
                }
            });
        };

        const stopResize = () => {
            document.removeEventListener('mousemove', handleMouseMove);
            document.removeEventListener('mouseup', stopResize);
        };

        document.addEventListener('mousemove', handleMouseMove);
        document.addEventListener('mouseup', stopResize);
    }, []);

    return (
        <>
            <div
                className="fixed left-0 w-full cursor-row-resize bg-gray-600"
                onMouseDown={startResize}
                style={{ height: '5px', bottom: `${height}px`, zIndex: 10 }}  // Positioning the resize handle
            />
            <div
                className="fixed bottom-0 left-0 w-full bg-gray-800 flex flex-col"
                ref={barRef}
                style={{ height, zIndex: 9 }}  // Adjust the z-index for layering
            >
                <nav className="flex border-b border-gray-700">
                    <TabSelector
                        isActive={selectedTab === 'account'}
                        onClick={() => setSelectedTab('account')}
                    >
                        <i className=""></i>
                        Account
                    </TabSelector>
                    <TabSelector
                        isActive={selectedTab === 'company'}
                        onClick={() => setSelectedTab('company')}
                    >
                        Company
                    </TabSelector>
                </nav>
                <div className="flex-grow p-4 bg-gray-700 text-white overflow-y-auto">
                    <TabPanel hidden={selectedTab !== 'account'}>
                        <div className="flex space-x-2">
                            <button
                                onClick={onExport}
                                className="px-4 py-2 font-semibold text-white bg-blue-600 rounded hover:bg-blue-500"
                            >
                                Export Graph
                            </button>
                            <button
                                onClick={onExec}
                                className="px-4 py-2 font-semibold text-white bg-green-600 rounded hover:bg-green-500"
                            >
                                Exec
                            </button>
                        </div>
                    </TabPanel>
                    <TabPanel hidden={selectedTab !== 'company'}>
                        <Context/>
                    </TabPanel>
                </div>
            </div>
        </>
    );
};

export default BottomBar;
