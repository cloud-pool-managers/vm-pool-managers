import { writable } from "svelte/store";
import { jwtDecode }  from 'jwt-decode';

interface User {
    user_id: string;
    name: string;
    email: string;
    token: string;
}

export const user = writable<User | null>(null);

export function initUser() {
    const token = localStorage.getItem('token');
    if (token) {
        try {
            const payload: any = jwtDecode(token);

            user.set({
                user_id: payload.user_id,
                name: payload.name,
                email: payload.email,
                token
            });
        } catch (err) {
            console.error('JWT invalide', err);
            localStorage.removeItem('token');
            user.set(null)
        }
    }
}

export async function loginUser(email: string, password: string) {
	const res = await fetch('http://localhost:8080/login', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ email, password })
	});

	if (!res.ok) {
		const data = await res.json();
		throw new Error(data.message || 'Erreur de connexion');
	}

	const data = await res.json();
	localStorage.setItem('token', data.token);

	// décoder le JWT pour initialiser le store
	const payload: any = jwtDecode(data.token);
	user.set({
		user_id: payload.user_id,
		name: payload.name,
        email: payload.email,
		token: data.token
	});
}

let error: string | null = null;
let loading = false;

export async function createUser(name: string, email: string, password: string, confirmpassword: string,) {
    error = null
    if (password != confirmpassword) {
        error = "Les mots de passe ne correspondent pas";
        return;
    }
    loading = true
    try {
        const res = await fetch('http://localhost:8080/users', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({name, email, password})
        });

        if (!res.ok) {
            const data = await res.json();
            error = data.message || 'Erreur lors du signin'
            return;
        }

        const data = await res.json();
        console.log('Compte crée:', data);

        localStorage.setItem('token', data.token);
    
    } catch {
        error = 'Impossible de contacter le serveur';
    } finally {
        loading = false
    }
    
}

// fonction logout
export function logoutUser() {
	localStorage.removeItem('token');
	user.set(null);
}