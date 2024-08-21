import {Handle, Node, NodeProps, Position} from '@xyflow/react';
import React, {useEffect, useState} from "react";
import Modal from "../../utils/modal.tsx";
import connectionService from "../../service/connection.service.ts";
import {toast} from "react-toastify";

type input = Node<{ }, 'dbInput'>;

async function getContextConn() {
    try {
        const response = await connectionService.getContextConnections();
        return response.data;
    } catch (error) {
        toast.error(error.message);
        return [];
    }
}

export default function DbInputNode({ }: NodeProps<input>) {
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [contextConn, setContextConn] = useState([]);
    const [selectedContextConn, setSelectedContextConn] = useState("");

    useEffect(() => {
        getContextConn().then((data) => {
            setContextConn(data);
        });
    }, []);

    const handleDoubleClick = () => {
        setIsModalOpen(true);
    };

    const handleCloseModal = () => {
        setIsModalOpen(false);
    };

    const save = () => {
        // Save the form data to the database
        console.log(selectedContextConn);
    }

    return (
        <>
            <div
                onDoubleClick={handleDoubleClick}
                className="bg-gradient-to-br bg-gray-700 rounded-lg shadow-lg p-4 cursor-pointer"
            >
                DB Input
                <Handle
                    type="source"
                    position={Position.Right}
                    id={"start-output"}
                    className={"bg-green-500 h-2 top-3"}
                />
            </div>

            {isModalOpen && (
                <Modal onClose={handleCloseModal}>
                    <div className="p-4">
                        <h2 className="text-lg font-semibold">Database Input Configuration</h2>

                        {/* Select Dropdown for Context Connections */}
                        <div className="mt-4">
                            <label className="block text-white mb-2">Select a Context Connection:</label>
                            <select
                                value={selectedContextConn}  // Assuming you have a state to manage the selected value
                                onChange={(e) => setSelectedContextConn(e.target.value)}  // Update the state with the selected value
                                className="p-2 w-full border border-gray-300 rounded bg-gray-800 text-white"
                            >
                                <option value="" disabled>Select a connection</option>
                                {contextConn.map((conn) => (
                                    <option key={conn.id} value={conn.id}>
                                        {conn.name}
                                    </option>
                                ))}
                            </select>
                        </div>

                        {/* Other form fields for DB configuration can go here */}

                        <button
                            onClick={save}
                            className="mt-4 px-4 py-2 bg-blue-500 text-white rounded mr-4"
                        >
                            Save
                        </button>
                        <button
                            onClick={handleCloseModal}
                            className="mt-4 px-4 py-2 bg-blue-500 text-white rounded"
                        >
                            Close
                        </button>

                    </div>
                </Modal>
            )}
        </>
    );
}

