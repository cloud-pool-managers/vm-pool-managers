import { get } from 'svelte/store';
import { authStore } from './authStore';
import type { Config } from './fetchinit';

export async function createConfig(
    configs: string,
    text: string
) {
    const token = get(authStore);
    try {
        const res = await fetch(`http://localhost:8080/users/me/configs`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ name: configs, data: text })
        });
        if (!res.ok) {
            throw new Error("Impossible de créer la configuration");
        }
    } catch (err) {
        console.error(err);
        throw err;
    }
}

export async function updateConfig(
    configId: number,
    newConfigName: string,
    text: string
) {
    const token = get(authStore);
    try {
        const res = await fetch(`http://localhost:8080/users/me/configs/${configId}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ id: configId, name: newConfigName, data: text })
        });
        if (!res.ok) {
            throw new Error("Impossible de mettre à jour la configuration");
        }
    } catch (err) {
        console.error(err);
        throw err;
    }
}
