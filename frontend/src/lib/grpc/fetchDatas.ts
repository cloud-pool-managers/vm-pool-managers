import { PoolManagerClient } from "./PoolmanagerServiceClientPb";
import { StreamRessourceResponse, UserRequest } from "./poolmanager_pb";

const client = new PoolManagerClient("http://localhost:8080");

export async function fetchUserRessources(username: string, onData: (res: StreamRessourceResponse) => void): Promise<void> {
    return new Promise((resolve, reject) => {
        const request = new UserRequest();
        request.setUser(username);

        const stream = client.getStreamRessourcesUser(request, {});

        stream.on('data', (response: StreamRessourceResponse) => {
            onData(response);
        });

        stream.on('error', (err: any) => {
            console.error("Stream error:", err);
            reject(err);
        });

        stream.on('end', () => {
            console.log("Stream ended");
            resolve();
        });
    });
}