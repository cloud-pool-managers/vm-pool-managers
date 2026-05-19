<script lang="ts">
  import '../app.css';
  import favicon from '$lib/assets/favicon.svg';
  import logo from '$lib/assets/IDCS.png'
  import {
    loadAll,
    login,
    logout,
    resetAll,
    subscribeUserUpdate,
  } from '$lib/index'
  import { authStore } from '$lib/store';
  import {
    Navbar,
    NavBrand,
    NavLi,
    NavUl,
    NavHamburger,
    Button
  } from 'flowbite-svelte';
  import { Modal, Label, Input } from 'flowbite-svelte';
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import { browser } from '$app/environment';
  import {
    createUser,
    authenticateUser,
  } from '$lib/grpc/authService/authService';

  
  let { children } = $props();

  let userStreamController: AbortController | null = null;

  authStore.subscribe(async (auth) => {
    if (!browser) return;

    if (userStreamController) {
      userStreamController.abort();
      userStreamController = null;
    }
    if (auth?.email) {
      userStreamController = new AbortController();
      await subscribeUserUpdate(auth.email, userStreamController.signal);
    }
  });

  onMount(async () => {
    if (!browser) return;
    const token = get(authStore);
    if (token) {
      await loadAll(token.email);
    } else {
      resetAll();
    }
  });

  // Modal Login
  let loginModal = $state(false);
  let loginError = $state("");
  let loginSuccess = $state(false);

  async function handleLogin(event: Event) {
    event.preventDefault();
    const form = event.target as HTMLFormElement;
    const data = new FormData(form);
    const email = data.get('email') as string;
    const password = data.get('password') as string;

    loginError = "";
    try {
      const result = await authenticateUser(email, password);
      if (!result.success) {
        loginError = "Identifiants incorrects";
        return;
      }

      login(result.token, email);
      loginSuccess = true;

      await loadAll(email);

      setTimeout(() => {
        form.reset();
        loginModal = false;
        loginSuccess = false;
      }, 1500);
    } catch (err) {
      loginError = "Erreur de connexion au serveur";
      console.error(err);
    }
  }

  // Modal Create Account
  let createAccountModal = $state(false);
  let createAccountError = $state("");
  let createAccountSuccess = $state(false);

  async function tryCreate(event: Event) {
    event.preventDefault();
    const form = event.target as HTMLFormElement;
    const data = new FormData(form);

    createAccountError = "";
    createAccountSuccess = false;

    const name = data.get("name") as string;
    const email = data.get("email") as string;
    const password = data.get("password") as string;
    const confirmpassword = data.get("confirmpassword") as string;

    if (!name || !email || !password || !confirmpassword) {
      createAccountError = "Tous les champs sont requis";
      return;
    }

    if (password !== confirmpassword) {
      createAccountError = "Les mots de passe ne correspondent pas";
      return;
    }

    try {
      const result = await createUser(name, email, password);
      if (result.success) {
        createAccountSuccess = true;
      } else {
        createAccountError = "Impossible de creer le compte";
        return;
      }
    } catch (err) {
      createAccountError = "Erreur de connexion au serveur";
      console.error(err);
    }

    setTimeout(() => {
      form.reset();
      createAccountModal = false;
      createAccountSuccess = false;
      loginModal = true;
    }, 2000);
  }
</script>


<svelte:head>
	<link rel="icon" href={favicon} />
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous">
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
</svelte:head>

<div class="min-h-screen bg-primary-500" style="font-family: 'Inter', sans-serif;">
	<Navbar class="sticky start-0 top-0 z-20 w-full bg-tertiary-600/95
    backdrop-blur-sm border-b border-tertiary-400/20">
		<NavBrand href="/">
			<img src={logo} class="me-3 h-6 sm:h-8" alt="IDCS Logo" />
			<span class="self-center text-lg font-semibold tracking-tight
        whitespace-nowrap text-gray-100">
        CloudPoolManager
      </span>
		</NavBrand>
	<div class="flex md:order-2 gap-2">
		{#if $authStore}
      {#if $authStore.role === 'admin'}
        <span class="self-center text-xs font-medium px-2.5 py-0.5 rounded-full
          bg-option-500/20 text-option-400 border border-option-500/30">
          Admin
        </span>
      {/if}
			<Button size="sm" color="red"
        class="font-medium text-sm"
        onclick={logout}>Deconnexion</Button>
		{:else}
		<Button
      size="sm"
      class="bg-secondary-400 border-0 text-white hover:bg-secondary-500 font-medium text-sm"
      onclick={() => (loginModal = true)}>
      Connexion
    </Button>
		<Button
      size="sm"
      class="bg-option-600 border-0 text-white hover:bg-option-700 font-medium text-sm"
      onclick={() => (createAccountModal = true)}>
      Inscription
    </Button>
		{/if}
		<NavHamburger />
	</div>
	<NavUl>
		<NavLi href="/" class="text-gray-300 hover:text-white text-sm font-medium">Accueil</NavLi>
		{#if $authStore}
      {#if $authStore.role === 'admin'}
		  <NavLi href="/inventory" class="text-gray-300 hover:text-white text-sm font-medium">Inventaire</NavLi>
      <NavLi href="/serverpool" class="text-gray-300 hover:text-white text-sm font-medium">Serverpools</NavLi>
      <NavLi href="/config" class="text-gray-300 hover:text-white text-sm font-medium">Configurations</NavLi>
      {/if}
      <NavLi href="/profile" class="text-gray-300 hover:text-white text-sm font-medium">Profil</NavLi>
		{/if}
	</NavUl>
	
	</Navbar>

	<!-- Login Modal -->
	 <Modal bind:open={loginModal} class="bg-tertiary-600 border border-tertiary-400/20">
		<form class="flex flex-col space-y-5" onsubmit={handleLogin}>
			<h3 class="text-xl font-semibold text-gray-100">Connexion</h3>
			{#if loginError}
				<p class="text-sm text-red-400 bg-red-400/10 px-3 py-2 rounded-lg">{loginError}</p>
			{/if}
			{#if loginSuccess}
				<p class="text-sm text-option-400 bg-option-400/10 px-3 py-2 rounded-lg">Connexion reussie</p>
			{/if}
			<Label class="space-y-1.5">
				<span class="text-sm text-gray-300">Email</span>
				<Input
          type="email"
          name="email"
          placeholder="nom@exemple.com"
          class="bg-tertiary-700 border-tertiary-400/30 text-gray-100 placeholder-gray-500"
          required/>
			</Label>
			<Label class="space-y-1.5">
				<span class="text-sm text-gray-300">Mot de passe</span>
				<Input
          type="password"
          name="password"
          placeholder="Votre mot de passe"
          class="bg-tertiary-700 border-tertiary-400/30 text-gray-100 placeholder-gray-500"
          required/>
			</Label>
			<Button type="submit" class="bg-secondary-400 hover:bg-secondary-500 text-white font-medium">Se connecter</Button>
		</form>
	 </Modal>

	<!-- Create Account Modal -->
	<Modal bind:open={createAccountModal} class="bg-tertiary-600 border border-tertiary-400/20">
		<form class="flex flex-col space-y-5" onsubmit={tryCreate}>
    <h3 class="text-xl font-semibold text-gray-100">Creer un compte</h3>
			{#if createAccountError}
				<p class="text-sm text-red-400 bg-red-400/10 px-3 py-2 rounded-lg">{createAccountError}</p>
			{/if}
			{#if createAccountSuccess}
				<p class="text-sm text-option-400 bg-option-400/10 px-3 py-2 rounded-lg">Compte cree avec succes</p>
			{/if}
			<Label class="space-y-1.5">
				<span class="text-sm text-gray-300">Nom</span>
				<Input type="text" name="name" placeholder="Votre nom"
          class="bg-tertiary-700 border-tertiary-400/30 text-gray-100 placeholder-gray-500"
          required/>
			</Label>
			<Label class="space-y-1.5">
				<span class="text-sm text-gray-300">Email</span>
				<Input 
          type="email" 
          name="email" 
          placeholder="nom@exemple.com"
          class="bg-tertiary-700 border-tertiary-400/30 text-gray-100 placeholder-gray-500"
          required/>
			</Label>
			<Label class="space-y-1.5">
				<span class="text-sm text-gray-300">Mot de passe</span>
				<Input
          type="password"
          name="password"
          placeholder="Choisir un mot de passe"
          class="bg-tertiary-700 border-tertiary-400/30 text-gray-100 placeholder-gray-500"
          required/>
			</Label>
			<Label class="space-y-1.5">
				<span class="text-sm text-gray-300">Confirmer le mot de passe</span>
				<Input 
          type="password" 
          name="confirmpassword" 
          placeholder="Confirmer le mot de passe"
          class="bg-tertiary-700 border-tertiary-400/30 text-gray-100 placeholder-gray-500" 
          required/>
			</Label>
			<Button type="submit" class="bg-option-600 hover:bg-option-700 text-white font-medium">Creer le compte</Button>
		</form>
	</Modal>

	<main class="pt-8 px-6 text-gray-300 max-w-7xl mx-auto">
		{@render children?.()}
	</main>

  <footer class="mt-16 py-6 border-t border-tertiary-500/30 text-center text-xs text-gray-500">
    CloudPoolManager &mdash; IDCS Infrastructure
  </footer>
</div>
