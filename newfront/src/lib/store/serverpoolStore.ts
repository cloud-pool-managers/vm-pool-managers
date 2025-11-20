import { writable } from "svelte/store";

import {
    getAllImages,
    getAllFlavors,
    getAllNetworks,
    getAllServers,
    getAllServerPools,
} from "$lib/index";

import type {
    Image,
    Flavor,
    Network,
    Server,
    ServerPool,
} from "../grpc/frontcontrol_pb";


// ==========================================================================
// Stores
// ==========================================================================
export const images = writable<Image[]>([]);
export const flavors = writable<Flavor[]>([]);
export const networks = writable<Network[]>([]);
export const servers = writable<Server[]>([]);
export const serverPools = writable<ServerPool[]>([]);


// ==========================================================================
// Loaders (chargent les données et mettent à jour les stores)
// ==========================================================================

export async function loadImages() {
    const data = await getAllImages();
    images.set(data);
}

export async function loadFlavors() {
    const data = await getAllFlavors();
    flavors.set(data);
}

export async function loadNetworks() {
    const data = await getAllNetworks();
    networks.set(data);
}

export async function loadServers(user: string) {
    const data = await getAllServers(user);
    servers.set(data);
}

export async function loadServerPools(user: string) {
    const data = await getAllServerPools(user);
    serverPools.set(data);
}


// ==========================================================================
// Helper pour tout charger d'un coup (infrastructure générale)
// ==========================================================================
export async function loadAll(user: string) {
    await Promise.all([
        loadImages(),
        loadFlavors(),
        loadNetworks(),
        loadServers(user),
        loadServerPools(user),
    ]);
}