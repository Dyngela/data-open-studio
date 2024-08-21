import {ApiService} from "./api.service.ts";
import {ConnectionData} from "../model/connection.model.ts";
import {GenericApiResponse} from "../model/api.model.ts";
import axios from "axios";

class ConnectionService {
    async validateConnection(connectionData: ConnectionData): Promise<GenericApiResponse<void>> {
        return await ApiService.post('/context', connectionData);
    }

    async getContextConnections()  {
        return await ApiService.get('/context');
    }
}

export default new ConnectionService();
