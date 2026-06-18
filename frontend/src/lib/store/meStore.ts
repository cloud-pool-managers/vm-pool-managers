import { writable } from 'svelte/store';
import { apiFetch } from '$lib/api';

// Rôle effectif de l'utilisateur, source de vérité = backend /api/me
// (le JWT ne reflète pas les rôles fins gérés en base : prof / ta / chercheur).
export interface Me {
  email: string;
  role: string;        // admin | prof | ta | student | chercheur
  is_admin: boolean;
  is_staff: boolean;   // admin | prof | ta
}

export const meStore = writable<Me | null>(null);

export async function loadMe(): Promise<void> {
  try {
    const r = await apiFetch('/api/me');
    if (r.ok) meStore.set(await r.json());
  } catch {
    /* ignore — on retombe sur le rôle du JWT */
  }
}

export function resetMe(): void {
  meStore.set(null);
}
