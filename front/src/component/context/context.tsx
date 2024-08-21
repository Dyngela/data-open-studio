import React, {useState} from "react";
import {ConnectionData} from "../../model/connection.model.ts";
import connectionService from "../../service/connection.service.ts";
import { ToastContainer, toast } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';

const drivers = ["Postgres", "Oracle", "SQL Server", "MySQL"];

const Context: React.FC = () => {
    const [driver, setDriver] = useState("");
    const [host, setHost] = useState("");
    const [port, setPort] = useState("");
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [connectionName, setConnectionName] = useState("");

    const [error, setError] = useState("");
    const [success, setSuccess] = useState("");

    const validateForm = () => {
        if (!driver || !host || !port || !username || !connectionName) {
            setError("All fields are required.");
            return false;
        }
        if (isNaN(Number(port))) {
            setError("Port must be a number.");
            return false;
        }
        setError("");
        return true;
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!validateForm()) return;

        const connectionData: ConnectionData = {
            database: driver,
            host,
            port,
            username,
            password,
            name: connectionName,
        };

        connectionService.validateConnection(connectionData)
        .then((_) => {
            toast.success("Context ajouté avec succès")
        })
        .catch((error) => {
            toast.error(error.message)
        })
    };



    return (
        <div className="p-4">
            <h2 className="text-lg font-bold mb-4">Database Connection</h2>
            {error && <div className="text-red-500 mb-4">{error}</div>}
            {success && <div className="text-green-500 mb-4">{success}</div>}
            <form onSubmit={handleSubmit}>
                <div className="mb-4">
                    <label className="block text-white">Connection Name</label>
                    <input
                        type="text"
                        value={connectionName}
                        onChange={(e) => setConnectionName(e.target.value)}
                        className="mt-1 p-2 w-full border border-gray-300 rounded text-gray-700"
                        required
                    />
                </div>
                <div className="mb-4">
                    <label className="block text-white">
                        Driver
                    </label>
                    <select
                        value={driver}
                        onChange={(e) => setDriver(e.target.value)}
                        className="mt-1 p-2 w-full border border-gray-300 rounded bg-gray-800 text-white"
                        required
                    >
                        <option value="" disabled>Select a driver</option>
                        {drivers.map((drv) => (
                            <option key={drv} value={drv}>
                                {drv}
                            </option>
                        ))}
                    </select>
                </div>
                <div className="mb-4">
                    <label className="block text-white">Host</label>
                    <input
                        type="text"
                        value={host}
                        onChange={(e) => setHost(e.target.value)}
                        className="mt-1 p-2 w-full border border-gray-300 rounded text-gray-700"
                        required
                    />
                </div>
                <div className="mb-4">
                    <label className="block text-white">Port</label>
                    <input
                        type="text"
                        value={port}
                        onChange={(e) => setPort(e.target.value)}
                        className="mt-1 p-2 w-full border border-gray-300 rounded text-gray-700"
                        required
                    />
                </div>
                <div className="mb-4">
                    <label className="block text-white">Username</label>
                    <input
                        type="text"
                        value={username}
                        onChange={(e) => setUsername(e.target.value)}
                        className="mt-1 p-2 w-full border border-gray-300 rounded text-gray-700"
                        required
                    />
                </div>
                <div className="mb-4">
                    <label className="block text-white">Password</label>
                    <input
                        type="password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        className="mt-1 p-2 w-full border border-gray-300 rounded text-gray-700"
                    />
                </div>
                <div className="flex space-x-2">
                    <button
                        type="submit"
                        className="px-4 py-2 font-semibold text-white bg-blue-600 rounded hover:bg-blue-500"
                    >
                        Save Connection
                    </button>
                </div>
            </form>
        </div>
    );
}

export default Context;
