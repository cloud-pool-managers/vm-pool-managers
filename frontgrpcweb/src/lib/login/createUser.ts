import { RessourceRequest } from "$lib/grpc/poolmanager_pb";
import { PoolManagerClient } from "$lib/grpc/PoolmanagerServiceClientPb";
import { Empty } from 'google-protobuf/google/protobuf/empty_pb';

async function createUser(user: string, data: Record<string, string>): Promise<void> {
    const client = new PoolManagerClient('http://localhost:8080');

    const request = new RessourceRequest();
    request.setUser(user);
    const dataMap = request.getDataMap();
    for (const key in data) {
        dataMap.set(key, data[key]);
    }

    return new Promise((resolve, reject) => {
        client.sendRessources(request, null, (err, response: any) => {
            if (err) {
                reject(err);
            } else {
                resolve();
            }
        });
    });
}

