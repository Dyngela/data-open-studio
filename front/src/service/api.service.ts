import axios from 'axios';

// Base URL for API requests
const baseURL = 'http://localhost:8080/api/v1';

// Create an axios instance with default settings
const apiClient = axios.create({
    baseURL,
    headers: {
        'Content-Type': 'application/json',
    },
});

// Error handling function
const formatErrors = (error) => {
    return Promise.reject(error.response ? error.response.data : error.message);
};

export const ApiService = {
    get(path: string, params = {}) {
        return apiClient
            .get(path, { params })
            .catch(formatErrors);
    },

    put(path: string, body = {}) {
        return apiClient
            .put(path, body)
            .catch(formatErrors);
    },

    post(path: string, body = {}) {
        return apiClient
            .post(path, body)
            .catch(formatErrors);
    },

    delete(path: string) {
        return apiClient
            .delete(path)
            .catch(formatErrors);
    },

    postFile(path: string, fileToUpload) {
        const formData = new FormData();
        formData.append('file', fileToUpload, fileToUpload.name);

        return apiClient
            .post(path, formData, {
                headers: {
                    'Content-Type': 'multipart/form-data',
                },
            })
            .catch(formatErrors);
    },

    putFile(path: string, body = {}, file = []) {
        const formData = new FormData();
        formData.append('data', JSON.stringify(body));

        file.forEach((f, i) => {
            formData.append(String(i), f, f.name);
        });

        return apiClient
            .put(path, formData, {
                headers: {
                    'Content-Type': 'multipart/form-data',
                },
            })
            .catch(formatErrors);
    },

    downloadFile(path: string, arg = null) {
        return apiClient
            .post(path, arg, {
                responseType: 'text',
            })
            .catch(formatErrors);
    },

    displayFileBlob(path: string, arg = null) {
        return apiClient
            .post(path, arg, {
                responseType: 'blob',
            })
            .catch(formatErrors);
    },

    downloadZip(path: string) {
        return apiClient
            .get(path, {
                responseType: 'arraybuffer',
            })
            .catch((error) => {
                if (error.response) {
                    // Handle HTTP error
                    return Promise.reject(error.response.data);
                } else {
                    // Handle network or client-side error
                    return Promise.reject(error.message);
                }
            });
    },
};
