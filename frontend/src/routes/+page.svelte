<script lang="ts">
  import { returnPoolsWithKey, attribVMinPool } from "$lib/grpc/attribVMService/attribVMService";

  let sshkey = $state("");
  let availablePools: { pool_id: string; user_id: string }[] = $state([]);
  let selectedPool: { pool_id: string; user_id: string } | null = $state(null);
  let vmIp = $state("");
  let vmUser = $state("");
  let loading = $state(false);
  let errorMsg = $state("");

  async function handleSSHKey() {
    if (!sshkey.trim()) return;

    loading = true;
    errorMsg = "";
    availablePools = [];
    selectedPool = null;
    vmIp = "";

    try {
      availablePools = await returnPoolsWithKey(sshkey);
      if (availablePools.length === 0) {
        errorMsg = "Aucun cours ou pool de VMs disponible actuellement.";
      }
    } catch (err) {
      console.error(err);
      errorMsg = "Erreur lors de la récupération des cours disponibles.";
    } finally {
      loading = false;
    }
  }

  function computeUsername(poolId: string): string {
    let name = ("student_" + poolId).split("@")[0].toLowerCase();
    name = name.replace(/[^a-z0-9_.-]/g, "");
    if (name.length > 32) name = name.substring(0, 32);
    return name;
  }

  async function assignVM(pool: { pool_id: string; user_id: string }) {
    selectedPool = pool;
    loading = true;
    errorMsg = "";
    vmIp = "";
    vmUser = "";

    try {
      vmIp = await attribVMinPool(pool.pool_id, pool.user_id, sshkey);
      vmUser = computeUsername(pool.pool_id);
    } catch (err: any) {
      console.error(err);
      errorMsg = err?.message || "Erreur lors de l'attribution de la VM.";
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head>
  <title>CloudPoolManager - Portail Étudiant</title>
</svelte:head>

<div class="max-w-3xl mx-auto py-12">
  <div class="text-center mb-12">
    <h1 class="text-3xl font-semibold text-white tracking-wide">Portail Étudiant</h1>
    <p class="text-gray-400 mt-3 text-lg">Obtenez votre machine virtuelle pour les cours pratiques.</p>
  </div>

  <div class="bg-tertiary-300 border border-tertiary-200 rounded-xl p-8 shadow-xl">
    
    {#if !vmIp}
      <div class="space-y-6">
        <div>
          <label for="sshkey" class="block text-sm font-bold tracking-wide text-gray-300 mb-2">Clé publique SSH</label>
          <p class="text-sm text-gray-400 mb-3">Collez votre clé publique ed25519 ou rsa pour vous authentifier sur la VM qui vous sera attribuée.</p>
          <textarea
            id="sshkey"
            bind:value={sshkey}
            rows="4"
            placeholder="ssh-ed25519 AAAA..."
            class="w-full bg-tertiary-400 text-white border-tertiary-200 focus:ring-option-500 focus:border-option-500 text-sm rounded-lg p-3 font-mono transition-colors"
          ></textarea>
        </div>

        <div class="flex justify-end pt-2">
          <button
            onclick={handleSSHKey}
            disabled={loading || !sshkey.trim()}
            class="px-6 py-2.5 bg-option-500 text-white font-semibold text-sm rounded-lg hover:bg-option-600 shadow-md transition-transform hover:scale-105 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          >
            {#if loading && !selectedPool}
              <div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
              Recherche...
            {:else}
              Rechercher les cours
            {/if}
          </button>
        </div>

        {#if errorMsg}
          <div class="p-4 bg-red-900/40 border border-red-500 text-red-200 text-sm rounded-lg mt-4 shadow-sm">
            {errorMsg}
          </div>
        {/if}

        {#if availablePools.length > 0}
          <div class="mt-8 pt-8 border-t border-tertiary-200">
            <h3 class="text-lg font-bold text-white mb-4">Cours disponibles</h3>
            <div class="grid gap-4">
              {#each availablePools as pool}
                <div class="flex items-center justify-between p-5 bg-tertiary-400 border border-tertiary-200 rounded-xl shadow-sm hover:border-option-500 transition-all">
                  <div>
                    <p class="text-lg font-bold text-white">{pool.pool_id}</p>
                    <p class="text-sm text-gray-400 mt-1">Professeur: <span class="text-gray-300">{pool.user_id}</span></p>
                  </div>
                  <button
                    onclick={() => assignVM(pool)}
                    disabled={loading}
                    class="px-5 py-2.5 bg-tertiary-500 border border-tertiary-200 text-white text-sm font-semibold rounded-lg hover:bg-tertiary-600 shadow-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                  >
                    {#if loading && selectedPool === pool}
                      <div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
                      Création...
                    {:else}
                      Rejoindre
                    {/if}
                  </button>
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    {:else}
      <!-- Success State -->
      <div class="text-center py-8">
        <div class="w-20 h-20 bg-option-500/20 rounded-full flex items-center justify-center mx-auto mb-6 shadow-inner border border-option-500/30">
          <svg class="w-10 h-10 text-option-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
          </svg>
        </div>
        <h2 class="text-3xl font-bold text-white mb-3">Machine Virtuelle Attribuée</h2>
        <p class="text-gray-400 text-lg mb-10">Votre environnement de travail est prêt.</p>
        
        <div class="inline-block bg-tertiary-400 border border-tertiary-200 shadow-lg rounded-xl p-8 w-full max-w-md">
          <p class="text-xs text-gray-400 uppercase tracking-widest font-semibold mb-3">Commande de connexion</p>
          <div class="flex items-center justify-center gap-3 bg-tertiary-300 p-4 rounded-lg border border-tertiary-200 shadow-inner">
            <code class="text-option-400 font-mono text-lg font-bold">ssh {vmUser}@{vmIp}</code>
          </div>
        </div>
        
        <div class="mt-12">
          <button
            onclick={() => { vmIp = ""; vmUser = ""; availablePools = []; sshkey = ""; }}
            class="text-sm font-semibold text-gray-400 hover:text-white transition-colors"
          >
            ← Retour à l'accueil
          </button>
        </div>
      </div>
    {/if}

  </div>
</div>
