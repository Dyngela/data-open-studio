import Layout from "./component/layout/layout.tsx";
import './index.css';
import {DndProvider} from "react-dnd";
import {HTML5Backend} from "react-dnd-html5-backend";
import {ToastContainer} from "react-toastify";
import React from "react";


export default function App() {
    return (
        <DndProvider backend={HTML5Backend}>
            <div className="App">
                <ToastContainer />
                <Layout/>
            </div>
        </DndProvider>

    );

}
