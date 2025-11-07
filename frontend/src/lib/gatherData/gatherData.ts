
export interface User {
    name: string;
    email: string;
}

export interface Serverpool {
    ID: string;
    serverpool_id: string;
    image_ref: string;
    flavor_ref: string;
    networks: string[];
    min_vm: number;
    max_vm: number;
    pending_jobs: number;
}

export interface Server {
    id: string;
    name: string;
    status: string;
    flavor: { id: string; name: string | null };
    image: { id: string; name: string | null };
    addresses: Record<string, { addr: string }[]>;
    created: string;
    updated?: string;
    host_id?: string;
    progress?: number;
}

export interface Config {
    name: string;
    data: string;
}

interface UserStore {
    user: User | null;
    serverpools: Serverpool[];
    servers: Record<string, Server[]>; // Clé : serverpool_id
    configs: Config[];
    error: string | null;
}

