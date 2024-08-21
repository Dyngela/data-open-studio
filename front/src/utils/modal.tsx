import React from "react";
import ReactDOM from "react-dom";

interface ModalProps {
    children: React.ReactNode;
    onClose: () => void;
}

const Modal: React.FC<ModalProps> = ({ children, onClose }) => {
    return ReactDOM.createPortal(
        <>
            <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                <div className="bg-white rounded-lg shadow-lg p-6 relative w-full h-full m-8">
                    {children}
                    <button
                        onClick={onClose}
                        className="absolute top-2 right-2 text-gray-600 hover:text-gray-900"
                    >
                        X
                    </button>
                </div>
            </div>
        </>,
        document.body
    );
};

export default Modal;
