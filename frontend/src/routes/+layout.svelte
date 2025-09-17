<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { onMount } from 'svelte';
	import DialogModal from '$lib/DialogModal.svelte';
	import { user, logoutUser, initUser, loginUser, createUser } from '$lib/stores/user';
	import { writable } from 'svelte/store';

	// modals
	let showLogin = false;
	let showSignup = false;
	let showEndSignup = false;

	// champs login/signup
	let loginEmail = '';
	let loginPassword = '';
	let signupName = '';
	let signupEmail = '';
	let signupPassword = '';
	let signupConfirmPassword = '';

	// message d'erreur
	let errorMsg = '';

	onMount(() => initUser());

	async function handleLogin() {
		errorMsg = '';
		try {
			await loginUser(loginEmail, loginPassword);
			showLogin = false; // ferme la modal
			// réinitialiser champs
			loginEmail = '';
			loginPassword = '';
		} catch (err: any) {
			errorMsg = err.message;
		}
	}

	async function handleSignup() {
		errorMsg = '';
		try {
			await createUser(signupName, signupEmail, signupPassword, signupConfirmPassword);
			showSignup = false;
			showEndSignup = true;
			signupName = '';
			signupEmail = '';
			signupPassword = '';
			signupConfirmPassword = '';
		} catch (err: any) {
			errorMsg = err.message;
		}
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="min-h-screen flex flex-col bg-gray-50 text-gray-800">
	<!-- NAVBAR -->
	<nav class="bg-white shadow-md px-6 py-4 flex justify-between items-center">
		<h1 class="text-2xl font-bold text-blue-600">PoolManagerCloud</h1>
		<div class="space-x-4 flex items-center">
			<a href="/" class="hover:text-blue-600">Accueil</a>
			{#if $user}
				<a href="/dashboard" class="hover:text-blue-600">Dashboard</a>
				<button
					class="bg-red-500 text-white px-4 py-2 rounded-lg hover:bg-red-600"
					on:click={logoutUser}
				>
					Déconnexion
				</button>
			{:else}
				<button
					class="bg-blue-500 text-white px-4 py-2 rounded-lg hover:bg-blue-600"
					on:click={() => (showLogin = true)}
				>
					Connexion
				</button>
				<button
					class="bg-green-500 text-white px-4 py-2 rounded-lg hover:bg-green-600"
					on:click={() => (showSignup = true)}
				>
					Inscription
				</button>
			{/if}
		</div>
	</nav>

	<!-- CONTENU DES PAGES -->
	<main class="flex-1">
		<slot />
	</main>

	<!-- FOOTER -->
	<footer class="bg-gray-100 text-center py-4 text-sm text-gray-600">
		© {new Date().getFullYear()} PoolManagerCloud. Tous droits réservés.
	</footer>
</div>

<!-- MODAL LOGIN -->
<DialogModal bind:showModal={showLogin}>
	<h3 slot="header">Connexion</h3>
  	<form on:submit|preventDefault={handleLogin}>
    	<input type="email" placeholder="Email" bind:value={loginEmail} required />
    	<input type="password" placeholder="Mot de passe" bind:value={loginPassword} required />
    	{#if errorMsg}<p class="text-red-500">{errorMsg}</p>{/if}
    	<button type="submit">Se connecter</button>
	</form>
</DialogModal>

<!-- MODAL SIGNUP -->
<DialogModal bind:showModal={showSignup}>
	<h3 slot="header">Inscription</h3>
	<form on:submit|preventDefault={handleSignup}>
    	<input type="text" placeholder="Nom" bind:value={signupName} required />
    	<input type="email" placeholder="Email" bind:value={signupEmail} required />
    	<input type="password" placeholder="Mot de passe" bind:value={signupPassword} required />
    	<input type="password" placeholder="Confirmer le mot de passe" bind:value={signupConfirmPassword} required />
    	{#if errorMsg}<p class="text-red-500">{errorMsg}</p>{/if}
    	<button type="submit">S’inscrire</button>
	</form>
</DialogModal>

<!-- MODAL FIN SIGNUP -->
<DialogModal bind:showModal={showEndSignup}>
	<h3 slot="header">Inscription complétée !</h3>
	<p>Votre compte a été créé avec succès.</p>
</DialogModal>
