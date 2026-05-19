import { writable } from 'svelte/store';
import { goto } from '$app/navigation';
import { authenticateUser } from '$lib/grpc/authService/authService';
import { resetAll } from './serverpoolStore';

interface AuthData {
  token: string;
  email: string;
  role: string;
}

function createAuthStore() {
  let initial: AuthData | null = null;

  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem('authData');
    if (saved) {
      const data: AuthData = JSON.parse(saved);
      if (data.token) {
        initial = data;
      }
    }
  }

  const store = writable<AuthData | null>(initial);

  store.subscribe((auth) => {
    if (typeof window === 'undefined') return;
    if (auth)
      localStorage.setItem('authData', JSON.stringify(auth));
    else
      localStorage.removeItem('authData');
  });

  return store;
}

export const authStore = createAuthStore();

// Parse role from token format "role:email:id"
function parseRole(token: string): string {
  const parts = token.split(':');
  if (parts.length >= 1) {
    return parts[0];
  }
  return 'student';
}

export function login(token: string, email: string) {
  const role = parseRole(token);
  authStore.set({ token, email, role });
}

export function logout() {
  authStore.set(null);
  resetAll();
  goto("/");
}

export async function tryLogin(email: string, password: string) {
  if (!email || !password) {
    return { success: false, error: 'Champs non rempli' };
  }

  try {
    const result = await authenticateUser(email, password);

    if (!result.success || !result.token) {
      return { success: false, error: 'Erreur lors de la connexion' };
    }

    login(result.token, email);

    return { success: true };
  } catch (err) {
    console.error(err);
    return { success: false, error: 'Erreur backend' };
  }
}
