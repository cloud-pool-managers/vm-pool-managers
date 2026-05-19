<script lang="ts">
  import { onMount } from 'svelte';
  import { authStore } from '$lib/store';
  import { browser } from '$app/environment';

  interface VMInstance {
    id: string;
    name: string;
    ip: string;
    public_ip: string;
    az: string;
    status: string;
    healthy: boolean;
    activity_status: string;
    registered_at: string;
    last_seen: string;
    raw_meta: Record<string, string>;
  }

  interface InventoryPool {
    pool_id: string;
    user_id: string;
    vms: VMInstance[];
  }

  let pools: InventoryPool[] = $state([]);
  let loading = $state(true);
  let error = $state("");
  let lastRefresh = $state("");
  let autoRefresh: ReturnType<typeof setInterval> | null = null;

  async function fetchInventory() {
    try {
      const res = await fetch('/api/inventory');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      pools = await res.json();
      lastRefresh = new Date().toLocaleTimeString('fr-FR');
      error = "";
    } catch (err) {
      console.error(err);
      error = "Impossible de charger l'inventaire";
    } finally {
      loading = false;
    }
  }

  onMount(async () => {
    if (!browser) return;
    const auth = $authStore;
    if (!auth || auth.role !== 'admin') {
      window.location.href = '/';
      return;
    }
    await fetchInventory();
    autoRefresh = setInterval(fetchInventory, 15000);
    return () => {
      if (autoRefresh) clearInterval(autoRefresh);
    };
  });

  function timeSince(dateStr: string): string {
    const now = new Date();
    const then = new Date(dateStr);
    const diff = Math.floor((now.getTime() - then.getTime()) / 1000);
    if (diff < 60) return `${diff}s`;
    if (diff < 3600) return `${Math.floor(diff / 60)}min`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h`;
    return `${Math.floor(diff / 86400)}j`;
  }

  function totalVMs(): number {
    return pools.reduce((acc, p) => acc + p.vms.length, 0);
  }

  function healthyVMs(): number {
    return pools.reduce((acc, p) => acc + p.vms.filter(v => v.healthy).length, 0);
  }

  function readyVMs(): number {
    return pools.reduce((acc, p) => acc + p.vms.filter(v => v.status === 'ready').length, 0);
  }
</script>

<svelte:head>
  <title>Inventaire VM - CloudPoolManager</title>
</svelte:head>

<div class="space-y-6">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-2xl font-semibold text-gray-100 tracking-tight">Inventaire des machines virtuelles</h1>
      <p class="text-sm text-gray-500 mt-1">
        Supervision en temps reel des VMs provisionnees par serverpool
      </p>
    </div>
    <div class="flex items-center gap-3">
      {#if lastRefresh}
        <span class="text-xs text-gray-500">Maj. {lastRefresh}</span>
      {/if}
      <button
        onclick={fetchInventory}
        class="px-3 py-1.5 text-sm font-medium text-gray-300 bg-tertiary-600 border border-tertiary-400/20 rounded-lg hover:bg-tertiary-500 transition-colors"
      >
        Actualiser
      </button>
    </div>
  </div>

  <!-- Stats -->
  {#if !loading && !error}
  <div class="grid grid-cols-4 gap-4">
    <div class="bg-tertiary-600/50 border border-tertiary-400/15 rounded-xl p-4">
      <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Pools</p>
      <p class="text-2xl font-semibold text-gray-100 mt-1">{pools.length}</p>
    </div>
    <div class="bg-tertiary-600/50 border border-tertiary-400/15 rounded-xl p-4">
      <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">VMs Total</p>
      <p class="text-2xl font-semibold text-gray-100 mt-1">{totalVMs()}</p>
    </div>
    <div class="bg-tertiary-600/50 border border-tertiary-400/15 rounded-xl p-4">
      <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Healthy</p>
      <p class="text-2xl font-semibold text-option-400 mt-1">{healthyVMs()}/{totalVMs()}</p>
    </div>
    <div class="bg-tertiary-600/50 border border-tertiary-400/15 rounded-xl p-4">
      <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Ready</p>
      <p class="text-2xl font-semibold text-gray-100 mt-1">{readyVMs()}</p>
    </div>
  </div>
  {/if}

  <!-- Loading -->
  {#if loading}
  <div class="flex items-center justify-center py-20">
    <div class="w-6 h-6 border-2 border-gray-500 border-t-option-400 rounded-full animate-spin"></div>
    <span class="ml-3 text-sm text-gray-400">Chargement de l'inventaire...</span>
  </div>
  {/if}

  <!-- Error -->
  {#if error}
  <div class="bg-red-400/10 border border-red-400/20 rounded-xl p-4 text-sm text-red-400">
    {error}
  </div>
  {/if}

  <!-- Pool sections -->
  {#if !loading && !error}
  {#each pools as pool}
  <div class="bg-tertiary-600/30 border border-tertiary-400/15 rounded-xl overflow-hidden">
    <!-- Pool header -->
    <div class="px-5 py-3 border-b border-tertiary-400/15 flex items-center justify-between">
      <div class="flex items-center gap-3">
        <div class="w-2 h-2 rounded-full {pool.vms.every(v => v.healthy) ? 'bg-option-400' : 'bg-red-400'}"></div>
        <h2 class="text-sm font-semibold text-gray-200 tracking-wide">{pool.pool_id}</h2>
        <span class="text-xs text-gray-500">({pool.user_id})</span>
      </div>
      <span class="text-xs text-gray-500">{pool.vms.length} VM{pool.vms.length > 1 ? 's' : ''}</span>
    </div>

    <!-- VM table -->
    <table class="w-full text-sm">
      <thead>
        <tr class="text-xs text-gray-500 uppercase tracking-wider border-b border-tertiary-400/10">
          <th class="text-left px-5 py-2.5 font-medium">Nom</th>
          <th class="text-left px-5 py-2.5 font-medium">IP</th>
          <th class="text-left px-5 py-2.5 font-medium">Statut</th>
          <th class="text-left px-5 py-2.5 font-medium">Sante</th>
          <th class="text-left px-5 py-2.5 font-medium">Activite</th>
          <th class="text-right px-5 py-2.5 font-medium">Derniere activite</th>
        </tr>
      </thead>
      <tbody>
        {#each pool.vms as vm, i}
        <tr class="border-b border-tertiary-400/5 hover:bg-tertiary-500/20 transition-colors
          {i % 2 === 0 ? 'bg-transparent' : 'bg-tertiary-600/20'}">
          <td class="px-5 py-3">
            <span class="text-gray-300 font-mono text-xs">{vm.name}</span>
          </td>
          <td class="px-5 py-3">
            <span class="text-gray-300 font-mono text-xs">{vm.ip}</span>
          </td>
          <td class="px-5 py-3">
            <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium
              {vm.status === 'ready' ? 'bg-option-500/15 text-option-400' :
               vm.status === 'starting' ? 'bg-yellow-500/15 text-yellow-400' :
               'bg-red-500/15 text-red-400'}">
              {vm.status}
            </span>
          </td>
          <td class="px-5 py-3">
            <div class="flex items-center gap-2">
              <div class="w-1.5 h-1.5 rounded-full {vm.healthy ? 'bg-option-400' : 'bg-red-400'}"></div>
              <span class="text-xs text-gray-400">{vm.healthy ? 'OK' : 'KO'}</span>
            </div>
          </td>
          <td class="px-5 py-3">
            {#if vm.activity_status && vm.activity_status !== 'idle'}
              <span class="inline-flex items-center gap-1.5 px-2 py-1 bg-blue-500/10 border border-blue-500/20 text-blue-400 text-xs rounded-md font-medium">
                <span class="relative flex h-2 w-2">
                  <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"></span>
                  <span class="relative inline-flex rounded-full h-2 w-2 bg-blue-500"></span>
                </span>
                SSH Actif ({vm.activity_status})
              </span>
            {:else}
              <span class="text-xs text-gray-500">Inactif</span>
            {/if}
          </td>
          <td class="px-5 py-3 text-right">
            <span class="text-xs text-gray-500">il y a {timeSince(vm.last_seen)}</span>
          </td>
        </tr>
        {/each}
      </tbody>
    </table>
  </div>
  {/each}

  {#if pools.length === 0}
  <div class="text-center py-16">
    <p class="text-gray-500 text-sm">Aucune VM enregistree pour le moment</p>
    <p class="text-gray-600 text-xs mt-1">Les VMs apparaitront ici une fois provisionnees et enregistrees par l'agent</p>
  </div>
  {/if}
  {/if}
</div>
