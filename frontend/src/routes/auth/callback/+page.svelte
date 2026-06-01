<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { browser } from '$app/environment';
  import { page } from '$app/state';
  import { loginOIDC } from '$lib/store/authStore';

  let error = $state('');

  onMount(async () => {
    if (!browser) return;

    // Use SvelteKit page state which handles client-side routing safely
    const code = page.url.searchParams.get('code');
    const state = page.url.searchParams.get('state');
    const storedState = sessionStorage.getItem('oidc_state');
    const codeVerifier = sessionStorage.getItem('oidc_code_verifier');

    if (!code) {
      const dexError = page.url.searchParams.get('error');
      const dexDesc = page.url.searchParams.get('error_description');
      error = dexError ? `Erreur Dex: ${dexError} — ${dexDesc}` : `Code manquant. URL courante: ${window.location.href}`;
      return;
    }

    if (state !== storedState) {
      error = 'State OIDC invalide.';
      return;
    }

    try {
      // Use Caddy proxy for token exchange (avoids mixed content / port issues)
      const tokenResp = await fetch(`/dex/token`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams({
          grant_type: 'authorization_code',
          code,
          redirect_uri: window.location.origin + '/auth/callback',
          client_id: 'cloudpoolmanager',
          code_verifier: codeVerifier ?? '',
        }),
      });

      if (!tokenResp.ok) {
        const text = await tokenResp.text();
        error = `Erreur échange token: ${text}`;
        return;
      }

      const tokens = await tokenResp.json();
      sessionStorage.removeItem('oidc_state');
      sessionStorage.removeItem('oidc_code_verifier');

      await loginOIDC(tokens.id_token, tokens.access_token);
      const { get } = await import('svelte/store');
      const { authStore } = await import('$lib/store/authStore');
      const auth = get(authStore);
      goto(auth?.role === 'admin' ? '/serverpool' : '/student');
    } catch (e: any) {
      error = e?.message ?? 'Erreur inconnue';
    }
  });
</script>

<div class="max-w-md mx-auto py-20 text-center">
  {#if error}
    <div class="card p-6 text-red-700 bg-red-50 border border-red-200">
      <p class="font-semibold mb-1">Erreur de connexion</p>
      <p class="text-sm">{error}</p>
      <a href="/" class="btn btn-secondary mt-4 inline-block">Retour à l'accueil</a>
    </div>
  {:else}
    <p class="text-neutral-500 text-sm">Connexion en cours…</p>
    <div class="mt-4 w-6 h-6 border-2 border-primary-200 border-t-primary-700 rounded-full mx-auto" style="animation: spinnerGlow 0.6s linear infinite;"></div>
  {/if}
</div>
