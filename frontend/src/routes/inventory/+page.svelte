<script lang="ts">
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api';
  import { authStore } from '$lib/store';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';
  import { simpleMode, refreshInterval } from '$lib/store/uiStore';
  import { browser } from '$app/environment';

  interface VMInstance {
    id: string; name: string; ip: string; public_ip: string; az: string;
    status: string; healthy: boolean; activity_status: string;
    registered_at: string; last_seen: string; raw_meta: Record<string, string>;
    power_state?: string;    // état Nova live : ACTIVE | SHUTOFF | SUSPENDED…
    guac_url?: string;
    student?: string;        // étudiant attribué (par IP)
    is_instructor?: boolean; // VM de l'enseignant
  }

  // Un seul badge d'état clair, basé sur l'état Nova réel.
  function powerBadge(ps?: string): { label: string; cls: string } {
    switch (ps) {
      case 'ACTIVE': return { label: 'Allumée', cls: 'bg-green-100 text-green-700 border-green-200' };
      case 'SHUTOFF': return { label: 'Éteinte', cls: 'bg-neutral-100 text-neutral-500 border-neutral-200' };
      case 'SUSPENDED':
      case 'PAUSED': return { label: 'Suspendue', cls: 'bg-amber-100 text-amber-700 border-amber-200' };
      case 'REBOOT':
      case 'HARD_REBOOT': return { label: 'Redémarrage…', cls: 'bg-sky-100 text-sky-700 border-sky-200' };
      case 'BUILD': return { label: 'Création…', cls: 'bg-sky-100 text-sky-700 border-sky-200' };
      case 'ERROR': return { label: 'Erreur', cls: 'bg-red-100 text-red-700 border-red-200' };
      default: return { label: ps || '—', cls: 'bg-neutral-100 text-neutral-500 border-neutral-200' };
    }
  }
  interface InventoryPool { pool_id: string; user_id: string; vms: VMInstance[]; linked_course?: string; label?: string; tags?: string; }
  const tagList = (t?: string) => (t || '').split(',').map(s => s.trim()).filter(Boolean);

  let pools: InventoryPool[] = $state([]);
  let loading = $state(true);
  let error = $state('');
  let lastRefresh = $state('');
  let refreshing = $state(false);

  async function fetchInventory(silent = false) {
    if (!silent) loading = true; else refreshing = true;
    try {
      const res = await apiFetch('/api/inventory');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      pools = await res.json();
      lastRefresh = new Date().toLocaleTimeString('fr-FR');
      error = '';
    } catch { error = "Impossible de charger l'inventaire"; }
    finally { loading = false; refreshing = false; }
  }

  // Actions de cycle de vie d'une VM (Phase 0 : /api/vm/action).
  let actingId = $state<string | null>(null);
  // Message d'action séparé de `error` (qui, lui, masque tout l'inventaire).
  let vmMsg = $state<{ type: 'ok' | 'err'; text: string } | null>(null);
  let vmMsgTimer: ReturnType<typeof setTimeout> | null = null;
  function showVmMsg(type: 'ok' | 'err', text: string) {
    vmMsg = { type, text };
    if (vmMsgTimer) clearTimeout(vmMsgTimer);
    vmMsgTimer = setTimeout(() => { vmMsg = null; }, 7000);
  }

  const ACTION_OK: Record<string, string> = {
    start: 'VM démarrée.', stop: 'VM arrêtée.', reboot: 'Redémarrage lancé.',
    suspend: 'VM suspendue.', resume: 'VM reprise.',
  };
  // Traduit les conflits OpenStack (409 task_state/vm_state) en messages clairs.
  function friendlyVMError(raw: string, action: string): string {
    const s = (raw || '').toLowerCase();
    if (action === 'start' && s.includes('vm_state active')) return 'La VM est déjà démarrée.';
    if (action === 'stop' && (s.includes('vm_state stopped') || s.includes('shutoff'))) return 'La VM est déjà arrêtée.';
    if (s.includes('task_state') || s.includes('reboot') || s.includes('powering') || s.includes('409') || s.includes('conflict')) {
      return 'Une action est déjà en cours sur cette VM — patiente quelques secondes puis réessaie.';
    }
    return "Action impossible dans l'état actuel de la VM.";
  }

  // Confirmation avant les actions disruptives (arrêter / redémarrer).
  let confirmState = $state<{ show: boolean; title: string; message: string; confirmText: string; danger: boolean; onConfirm: () => void }>(
    { show: false, title: '', message: '', confirmText: 'Confirmer', danger: false, onConfirm: () => {} }
  );
  function requestVmAction(vm: VMInstance, action: string) {
    if (action === 'stop' || action === 'reboot') {
      const verbe = action === 'stop' ? 'arrêter' : 'redémarrer';
      confirmState = {
        show: true,
        title: (action === 'stop' ? 'Arrêter' : 'Redémarrer') + ' la machine',
        message: `Voulez-vous vraiment ${verbe} la machine « ${vm.name} » ? Les sessions en cours seront interrompues.`,
        confirmText: action === 'stop' ? 'Arrêter' : 'Redémarrer',
        danger: true,
        onConfirm: () => vmAction(vm, action),
      };
    } else {
      vmAction(vm, action);
    }
  }

  // Réinitialisation (rebuild) — destructif.
  function requestVmRebuild(vm: VMInstance) {
    confirmState = {
      show: true,
      title: 'Réinitialiser la machine',
      message: `Réinitialiser « ${vm.name} » ? La VM sera réinstallée sur son image d'origine et toutes les données présentes sur la machine seront perdues.`,
      confirmText: 'Réinitialiser',
      danger: true,
      onConfirm: () => vmRebuild(vm),
    };
  }
  async function vmRebuild(vm: VMInstance) {
    if (actingId) return;
    actingId = vm.id;
    try {
      const res = await apiFetch('/api/vm/rebuild', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ server_id: vm.id }),
      });
      if (!res.ok) {
        const e = await res.json().catch(() => ({}));
        showVmMsg('err', 'Réinitialisation échouée : ' + (e.error || `HTTP ${res.status}`));
        actingId = null;
      } else {
        showVmMsg('ok', 'Réinitialisation lancée — la VM est en cours de réinstallation.');
        setTimeout(() => { fetchInventory(true); actingId = null; }, 2500);
      }
    } catch {
      showVmMsg('err', 'Réinitialisation impossible : service injoignable.');
      actingId = null;
    }
  }

  async function vmAction(vm: VMInstance, action: string) {
    if (actingId) return;
    actingId = vm.id;
    try {
      const res = await apiFetch('/api/vm/action', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ server_id: vm.id, action }),
      });
      if (!res.ok) {
        const e = await res.json().catch(() => ({}));
        showVmMsg('err', friendlyVMError(e.error || '', action));
        actingId = null;
      } else {
        showVmMsg('ok', ACTION_OK[action] || 'Action effectuée.');
        // On garde la VM verrouillée le temps de la transition, puis on rafraîchit.
        setTimeout(() => { fetchInventory(true); actingId = null; }, 2500);
      }
    } catch {
      showVmMsg('err', 'Action impossible : service injoignable.');
      actingId = null;
    }
  }

  // Annonce étudiants (broadcast).
  let annMessage = $state('');
  let annActive = $state(false);
  let annSaving = $state(false);
  let annOpen = $state(false);
  async function loadAnnouncement() {
    try {
      const r = await apiFetch('/api/announcement');
      if (r.ok) { const d = await r.json(); annMessage = d.message || ''; annActive = !!d.active; }
    } catch { /* ignore */ }
  }
  async function saveAnnouncement() {
    annSaving = true;
    try {
      const r = await apiFetch('/api/admin/announcement', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: annMessage, active: annActive }),
      });
      if (r.ok) { const d = await r.json(); annActive = !!d.active; showVmMsg('ok', 'Annonce enregistrée.'); }
      else showVmMsg('err', "Échec de l'enregistrement de l'annonce.");
    } catch { showVmMsg('err', "Échec de l'enregistrement."); }
    finally { annSaving = false; }
  }

  // Édition du libellé / des étiquettes d'un pool.
  let editingPool = $state<string | null>(null);
  let editLabel = $state('');
  let editTags = $state('');
  function startEditPool(p: InventoryPool) {
    editingPool = p.pool_id + ':' + p.user_id;
    editLabel = p.label || '';
    editTags = p.tags || '';
  }
  async function savePoolMeta(p: InventoryPool) {
    try {
      const r = await apiFetch('/api/pool/meta', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pool_id: p.pool_id, user_id: p.user_id, label: editLabel, tags: editTags }),
      });
      if (r.ok) { editingPool = null; showVmMsg('ok', 'Pool mis à jour.'); fetchInventory(true); }
      else showVmMsg('err', 'Échec de la mise à jour du pool.');
    } catch { showVmMsg('err', 'Échec de la mise à jour.'); }
  }

  onMount(() => {
    if (!browser) return;
    if (!$authStore || $authStore.role !== 'admin') { window.location.href = '/'; return; }
    fetchInventory();
    loadAnnouncement();
  });

  // Auto-refresh : intervalle configurable (Paramètres). Se recrée si l'intervalle change.
  $effect(() => {
    if (!browser || !$authStore || $authStore.role !== 'admin') return;
    const ms = Math.max(3, $refreshInterval || 15) * 1000;
    const id = setInterval(() => fetchInventory(true), ms);
    return () => clearInterval(id);
  });

  function timeSince(dateStr: string): string {
    const diff = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
    if (diff < 60) return `${diff}s`;
    if (diff < 3600) return `${Math.floor(diff/60)}min`;
    if (diff < 86400) return `${Math.floor(diff/3600)}h`;
    return `${Math.floor(diff/86400)}j`;
  }

  let poolSearch = $state('');
  const filteredPools = $derived(
    poolSearch.trim()
      ? pools.filter(p => {
          const q = poolSearch.trim().toLowerCase();
          return (p.pool_id + ' ' + p.user_id + ' ' + (p.linked_course || '') + ' ' + (p.label || '') + ' ' + (p.tags || '')).toLowerCase().includes(q)
            || p.vms.some(v => (v.student || '').toLowerCase().includes(q));
        })
      : pools
  );

  const totalVMs = $derived(pools.reduce((a, p) => a + p.vms.length, 0));
  const healthyVMs = $derived(pools.reduce((a, p) => a + p.vms.filter(v => v.healthy).length, 0));
  const readyVMs = $derived(pools.reduce((a, p) => a + p.vms.filter(v => v.status === 'ready').length, 0));
  const activeVMs = $derived(pools.reduce((a, p) => a + p.vms.filter(v => v.activity_status !== 'idle').length, 0));
</script>

<svelte:head><title>Inventaire VM — CloudPoolManager</title></svelte:head>

{#snippet actionButtons(vm: VMInstance)}
  {#if vm.id}
    {#if vm.power_state === 'SHUTOFF'}
      <button onclick={() => requestVmAction(vm,'start')} disabled={actingId===vm.id} title="Démarrer" class="btn btn-secondary text-xs px-2 py-1" aria-label="Démarrer">▶</button>
    {:else if vm.power_state === 'SUSPENDED' || vm.power_state === 'PAUSED'}
      <button onclick={() => requestVmAction(vm,'resume')} disabled={actingId===vm.id} title="Reprendre" class="btn btn-secondary text-xs px-2 py-1" aria-label="Reprendre">▶</button>
    {:else if vm.power_state === 'ACTIVE'}
      <button onclick={() => requestVmAction(vm,'stop')} disabled={actingId===vm.id} title="Arrêter" class="btn btn-secondary text-xs px-2 py-1" aria-label="Arrêter">⏹</button>
      <button onclick={() => requestVmAction(vm,'reboot')} disabled={actingId===vm.id} title="Redémarrer" class="btn btn-secondary text-xs px-2 py-1" aria-label="Redémarrer">↻</button>
    {:else}
      <button onclick={() => requestVmAction(vm,'start')} disabled={actingId===vm.id} title="Démarrer" class="btn btn-secondary text-xs px-2 py-1" aria-label="Démarrer">▶</button>
      <button onclick={() => requestVmAction(vm,'stop')} disabled={actingId===vm.id} title="Arrêter" class="btn btn-secondary text-xs px-2 py-1" aria-label="Arrêter">⏹</button>
    {/if}
    <button onclick={() => requestVmRebuild(vm)} disabled={actingId===vm.id} title="Réinitialiser (réinstalle l'OS, efface les données)" class="btn btn-secondary text-xs px-2 py-1 !text-red-600" aria-label="Réinitialiser">⟲</button>
  {/if}
{/snippet}

<ConfirmModal bind:show={confirmState.show} title={confirmState.title} message={confirmState.message}
  confirmText={confirmState.confirmText} danger={confirmState.danger} onConfirm={confirmState.onConfirm} />

{#if vmMsg}
  <div class="fixed top-6 right-6 z-50 max-w-sm px-5 py-4 rounded-xl shadow-2xl text-sm font-medium flex items-start gap-3 animate-fade-in
    {vmMsg.type === 'ok' ? 'bg-green-600 text-white' : 'bg-amber-500 text-white'}">
    <svg class="w-5 h-5 shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      {#if vmMsg.type === 'ok'}
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
      {:else}
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
      {/if}
    </svg>
    <span class="flex-1">{vmMsg.text}</span>
    <button onclick={() => (vmMsg = null)} class="opacity-80 hover:opacity-100 shrink-0" aria-label="Fermer">✕</button>
  </div>
{/if}

<div class="max-w-7xl mx-auto mb-4">
  <div class="card px-5 py-4">
    <button onclick={() => (annOpen = !annOpen)} class="flex items-center justify-between w-full text-left">
      <span class="text-sm font-semibold text-neutral-800">📣 Annonce étudiants {#if annActive}<span class="badge badge-ready ml-2">active</span>{/if}</span>
      <span class="text-neutral-400 text-xs">{annOpen ? '▲ replier' : '▼ gérer'}</span>
    </button>
    {#if annOpen}
      <div class="mt-3 space-y-3">
        <textarea class="field text-sm" rows="2" placeholder="Message affiché en bandeau à tous (ex. maintenance prévue jeudi 14h…)" bind:value={annMessage}></textarea>
        <div class="flex items-center justify-between">
          <label class="flex items-center gap-2 text-sm text-neutral-600">
            <input type="checkbox" bind:checked={annActive} class="w-4 h-4 accent-primary-700" /> Afficher l'annonce
          </label>
          <button onclick={saveAnnouncement} disabled={annSaving} class="btn btn-primary text-sm">Enregistrer</button>
        </div>
      </div>
    {/if}
  </div>
</div>

{#if $simpleMode}
<div class="space-y-6 animate-fade-up">
  <div class="flex items-start justify-between">
    <div>
      <h1 class="text-3xl font-bold text-primary-800">Mes étudiants</h1>
      <p class="text-sm text-neutral-500 mt-1">Suivez la connexion de vos étudiants en temps réel</p>
    </div>
    <button onclick={() => fetchInventory(true)} disabled={refreshing} class="btn btn-secondary text-xs px-3.5 py-2">
      <svg class="w-3.5 h-3.5 {refreshing ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
      </svg>
      Actualiser
    </button>
  </div>

  {#if loading}
    <div class="flex justify-center py-20"><div class="w-8 h-8 rounded-full border-2 border-neutral-200 border-t-primary-700" style="animation: spinnerGlow 0.7s linear infinite;"></div></div>
  {:else if error}
    <div class="card px-4 py-3 border-red-200 bg-red-50 text-red-700 text-sm">{error}</div>
  {:else if pools.length === 0}
    <div class="card flex flex-col items-center justify-center py-20 text-center">
      <p class="text-neutral-500 text-sm">Aucun cours actif pour le moment</p>
    </div>
  {:else}
    {#if pools.length > 1}
      <input class="field text-sm mb-3" type="text" placeholder="Filtrer les cours (nom, enseignant, étudiant…)" bind:value={poolSearch} />
    {/if}
    <div class="space-y-4">
      {#each filteredPools as pool, pi}
        {@const activeVms = pool.vms.filter(v => v.activity_status !== 'idle')}
        {@const connectedStudents = pool.vms.filter(v => v.activity_status !== 'idle' && v.student)}
        {@const readyVms = pool.vms.filter(v => v.status === 'ready' && !v.is_instructor)}
        <div class="card overflow-hidden animate-fade-up" style="animation-delay:{pi*0.06}s">
          <div class="flex items-center justify-between px-5 py-4 border-b border-neutral-100">
            <div>
              <div class="flex items-center gap-2 flex-wrap">
                <h2 class="text-sm font-bold text-neutral-900">{pool.label || pool.pool_id}</h2>
                {#if pool.linked_course}
                  <span class="text-[10px] font-medium px-1.5 py-0.5 rounded bg-primary-50 text-primary-700 border border-primary-200">🎓 {pool.linked_course}</span>
                {/if}
                {#each tagList(pool.tags) as tag}
                  <span class="text-[10px] font-medium px-1.5 py-0.5 rounded bg-neutral-100 text-neutral-600 border border-neutral-200">{tag}</span>
                {/each}
              </div>
              <p class="text-xs text-neutral-400 mt-0.5">
                <span class="{connectedStudents.length > 0 ? 'text-green-600 font-semibold' : 'text-neutral-400'}">
                  {connectedStudents.length} étudiant{connectedStudents.length > 1 ? 's' : ''} connecté{connectedStudents.length > 1 ? 's' : ''}
                </span>
                · {readyVms.length} machine{readyVms.length > 1 ? 's' : ''} disponible{readyVms.length > 1 ? 's' : ''}
              </p>
            </div>
            <div class="flex items-center gap-1.5">
              {#if activeVms.length > 0}
                <span class="animate-ping absolute inline-flex h-2 w-2 rounded-full bg-green-400 opacity-60"></span>
                <span class="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
                <span class="text-xs text-green-600 font-semibold">En cours</span>
              {:else}
                <span class="inline-flex rounded-full h-2 w-2 bg-neutral-300"></span>
                <span class="text-xs text-neutral-400">En attente</span>
              {/if}
            </div>
          </div>
          <div class="divide-y divide-neutral-50">
            {#each pool.vms as vm}
              {@const connected = vm.activity_status !== 'idle'}
              {@const label = vm.student ? vm.student : connected ? 'Connexion personnelle (enseignant)' : vm.is_instructor ? 'VM enseignant (réservée)' : vm.status === 'ready' ? 'Machine libre' : 'Démarrage…'}
              <div class="flex items-center justify-between gap-3 px-5 py-3 transition-colors {connected ? 'bg-green-50/70 dark:bg-green-900/10' : 'hover:bg-neutral-50 dark:hover:bg-white/[0.03]'}">
                <div class="flex items-center gap-3 min-w-0">
                  <!-- Avatar : initiale de l'étudiant, ou icône ; vert vif si connecté -->
                  <div class="relative w-9 h-9 rounded-full flex items-center justify-center text-sm font-bold shrink-0 transition-colors
                    {connected ? 'bg-green-500 text-white shadow-sm' : vm.is_instructor ? 'bg-primary-100 text-primary-600 dark:bg-primary-900/40 dark:text-primary-300' : 'bg-neutral-100 text-neutral-400 dark:bg-neutral-800'}">
                    {#if vm.student}
                      {vm.student.charAt(0).toUpperCase()}
                    {:else if connected || vm.is_instructor}
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/></svg>
                    {:else}
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/></svg>
                    {/if}
                    {#if connected}
                      <span class="absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full bg-green-500 ring-2 ring-white dark:ring-[#13151f]"></span>
                    {/if}
                  </div>
                  <div class="min-w-0">
                    <p class="text-sm font-semibold truncate {connected ? 'text-neutral-900 dark:text-white' : 'text-neutral-500 dark:text-neutral-400'}">{label}</p>
                    <p class="text-[11px] text-neutral-400 font-mono truncate">{vm.name}</p>
                  </div>
                </div>
                <div class="flex items-center gap-3 shrink-0">
                  {#if connected}
                    <span class="badge badge-ready">● En ligne</span>
                  {:else if vm.student}
                    <span class="text-xs text-neutral-400">Hors ligne</span>
                  {:else if vm.is_instructor}
                    <span class="text-xs text-neutral-400">Réservée</span>
                  {:else if vm.status === 'ready'}
                    <span class="text-xs text-neutral-400">En attente</span>
                  {:else}
                    <span class="text-xs text-amber-600">Démarrage…</span>
                  {/if}
                  {#if vm.guac_url}
                    <a href={vm.guac_url} target="_blank" rel="noopener" class="btn btn-secondary text-xs px-2 py-1">Terminal</a>
                  {/if}
                  {@render actionButtons(vm)}
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
{:else}
<div class="space-y-7 animate-fade-up">

  <!-- Header -->
  <div class="flex items-start justify-between">
    <div>
      <h1 class="text-3xl font-bold text-primary-800">Inventaire</h1>
      <p class="text-sm text-neutral-500 mt-1">Supervision en temps réel des instances provisionnées</p>
    </div>
    <div class="flex items-center gap-3">
      {#if lastRefresh}
        <span class="text-xs text-neutral-400">Maj {lastRefresh}</span>
      {/if}
      <button
        onclick={() => fetchInventory(true)}
        disabled={refreshing}
        class="btn btn-secondary text-xs px-3.5 py-2 gap-1.5"
      >
        <svg class="w-3.5 h-3.5 {refreshing ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
        </svg>
        Actualiser
      </button>
    </div>
  </div>

  <!-- Stats -->
  {#if !loading && !error}
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
      {#each [
        { label: 'Pools',       value: pools.length,                   accent: 'stat-accent-indigo',  color: 'text-primary-700' },
        { label: 'VMs total',   value: totalVMs,                       accent: 'stat-accent-violet',  color: 'text-primary-500' },
        { label: 'Santé',       value: `${healthyVMs}/${totalVMs}`,    accent: 'stat-accent-emerald', color: 'text-green-600'   },
        { label: 'Actives SSH', value: activeVMs,                      accent: 'stat-accent-amber',   color: 'text-amber-600'   },
      ] as stat, i}
        <div class="card card-interactive p-5 animate-fade-up" style="animation-delay:{i*0.05}s">
          <p class="section-label mb-2">{stat.label}</p>
          <p class="text-3xl font-bold {stat.color} tabular-nums tracking-tight">{stat.value}</p>
        </div>
      {/each}
    </div>
  {/if}

  <!-- Loading -->
  {#if loading}
    <div class="flex flex-col items-center justify-center py-24 gap-4">
      <div class="w-9 h-9 rounded-full border-2 border-neutral-200 border-t-primary-700" style="animation: spinnerGlow 0.7s linear infinite;"></div>
      <p class="text-sm text-neutral-500">Chargement de l'inventaire…</p>
    </div>
  {/if}

  <!-- Error -->
  {#if error}
    <div class="card px-4 py-3 border-red-200 bg-red-50 text-red-700 text-sm animate-fade-in">{error}</div>
  {/if}

  <!-- Pool sections -->
  {#if !loading && !error}
    {#if pools.length > 1}
      <input class="field text-sm mb-4" type="text" placeholder="Filtrer les pools (nom, enseignant, étudiant, cours…)" bind:value={poolSearch} />
    {/if}
    {#each filteredPools as pool, pi}
      <div class="card overflow-hidden animate-fade-up" style="animation-delay:{pi*0.06}s">
        <!-- Pool header -->
        <div class="flex items-center justify-between px-5 py-3.5 bg-neutral-50 border-b border-neutral-200">
          <div class="flex items-center gap-3">
            <div class="relative flex h-2.5 w-2.5">
              {#if pool.vms.every(v => v.healthy)}
                <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-60"></span>
              {/if}
              <span class="relative inline-flex rounded-full h-2.5 w-2.5 {pool.vms.every(v => v.healthy) ? 'bg-green-500' : 'bg-red-500'}"></span>
            </div>
            <span class="text-sm font-bold text-neutral-900">{pool.label || pool.pool_id}</span>
            {#if pool.label}<span class="text-xs text-neutral-400 font-mono">{pool.pool_id}</span>{/if}
            <span class="text-xs text-neutral-500">{pool.user_id}</span>
            {#if pool.linked_course}
              <span class="text-[10px] font-medium px-1.5 py-0.5 rounded bg-primary-50 text-primary-700 border border-primary-200">🎓 {pool.linked_course}</span>
            {/if}
            {#each tagList(pool.tags) as tag}
              <span class="text-[10px] font-medium px-1.5 py-0.5 rounded bg-neutral-100 text-neutral-600 border border-neutral-200">{tag}</span>
            {/each}
            <button onclick={() => startEditPool(pool)} title="Renommer / étiqueter" aria-label="Modifier le pool" class="text-neutral-400 hover:text-primary-700 text-xs">✎</button>
          </div>
          <span class="text-xs text-neutral-400 tabular-nums">{pool.vms.length} VM{pool.vms.length > 1 ? 's' : ''}</span>
        </div>

        {#if editingPool === pool.pool_id + ':' + pool.user_id}
          <div class="px-5 py-3 bg-neutral-50/60 border-b border-neutral-200 flex flex-wrap items-end gap-2">
            <div class="flex-1 min-w-[160px]">
              <label class="block text-[11px] text-neutral-500 mb-1">Nom d'affichage</label>
              <input class="field text-sm" type="text" placeholder={pool.pool_id} bind:value={editLabel} />
            </div>
            <div class="flex-1 min-w-[160px]">
              <label class="block text-[11px] text-neutral-500 mb-1">Étiquettes (séparées par des virgules)</label>
              <input class="field text-sm" type="text" placeholder="ex. TP, L1, semestre 2" bind:value={editTags} />
            </div>
            <button onclick={() => savePoolMeta(pool)} class="btn btn-primary text-sm">Enregistrer</button>
            <button onclick={() => (editingPool = null)} class="btn btn-secondary text-sm">Annuler</button>
          </div>
        {/if}

        <!-- Table -->
        <div class="overflow-x-auto">
          <table class="data-table">
            <thead>
              <tr>
                <th>Nom</th>
                <th>IP</th>
                <th>Statut</th>
                <th>Santé</th>
                <th>Activité</th>
                <th>Terminal</th>
                <th class="text-right">Dernière activité</th>
              </tr>
            </thead>
            <tbody>
              {#each pool.vms as vm}
                {@const connected = vm.activity_status !== 'idle'}
                <tr class="transition-colors {connected && vm.student ? 'bg-green-50/60 dark:bg-green-900/10' : ''}">
                  <td>
                    <div class="flex items-center gap-2.5">
                      <div class="w-7 h-7 rounded-full flex items-center justify-center text-[11px] font-bold shrink-0 transition-colors
                        {connected && vm.student ? 'bg-green-500 text-white shadow-sm' : vm.is_instructor ? 'bg-primary-100 text-primary-600 dark:bg-primary-900/40 dark:text-primary-300' : 'bg-neutral-100 text-neutral-400 dark:bg-neutral-800'}">
                        {#if vm.student}
                          {vm.student.charAt(0).toUpperCase()}
                        {:else if vm.is_instructor || connected}
                          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/></svg>
                        {:else}
                          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/></svg>
                        {/if}
                      </div>
                      <div class="flex flex-col gap-0.5 min-w-0">
                        {#if vm.student}
                          <span class="text-xs font-semibold truncate {connected ? 'text-green-700 dark:text-green-400' : 'text-neutral-700 dark:text-neutral-300'}">{vm.student}</span>
                        {:else if vm.is_instructor}
                          <span class="text-xs font-semibold text-primary-600 dark:text-primary-400">{connected ? 'Connexion personnelle' : 'VM enseignant'}</span>
                        {:else if connected}
                          <span class="text-xs font-semibold text-primary-600 dark:text-primary-400">Connexion personnelle</span>
                        {:else}
                          <span class="text-xs text-neutral-400">Machine libre</span>
                        {/if}
                        <span class="font-mono text-[10px] text-neutral-400 truncate">{vm.name}</span>
                      </div>
                    </div>
                  </td>
                  <td><span class="font-mono text-xs text-neutral-700">{vm.ip}</span></td>
                  <td>
                    {#if vm.power_state}
                      {@const b = powerBadge(vm.power_state)}
                      <span class="text-xs font-semibold px-2 py-0.5 rounded-full border {b.cls}">{b.label}</span>
                    {:else}
                      <span class="badge {vm.status === 'ready' ? 'badge-ready' : vm.status === 'starting' ? 'badge-starting' : 'badge-error'}">{vm.status}</span>
                    {/if}
                  </td>
                  <td>
                    <div class="flex items-center gap-1.5">
                      <span class="w-1.5 h-1.5 rounded-full {vm.healthy ? 'bg-green-500' : 'bg-red-500'}"></span>
                      <span class="text-xs font-medium {vm.healthy ? 'text-green-700' : 'text-red-700'}">{vm.healthy ? 'OK' : 'KO'}</span>
                    </div>
                  </td>
                  <td>
                    {#if vm.activity_status && vm.activity_status !== 'idle'}
                      <span class="badge badge-info gap-1.5">
                        <span class="relative flex h-1.5 w-1.5">
                          <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-sky-400 opacity-75"></span>
                          <span class="relative inline-flex rounded-full h-1.5 w-1.5 bg-sky-400"></span>
                        </span>
                        Sur Jupyter
                      </span>
                    {:else}
                      <span class="text-xs text-neutral-400">Inactif</span>
                    {/if}
                  </td>
                  <td>
                    {#if vm.guac_url}
                      <a href={vm.guac_url} target="_blank" rel="noopener"
                         class="btn btn-secondary text-xs px-2 py-1 flex items-center gap-1.5 w-fit">
                        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
                        </svg>
                        Terminal
                      </a>
                    {:else}
                      <span class="text-xs text-neutral-400">—</span>
                    {/if}
                    <div class="flex gap-1 mt-1">{@render actionButtons(vm)}</div>
                  </td>
                  <td class="text-right">
                    <span class="text-xs text-neutral-400 tabular-nums">il y a {timeSince(vm.last_seen)}</span>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/each}

    {#if pools.length === 0}
      <div class="card flex flex-col items-center justify-center py-24 text-center">
        <svg class="w-10 h-10 text-neutral-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
            d="M5 12h14M12 5l7 7-7 7"/>
        </svg>
        <p class="text-neutral-500 text-sm font-medium">Aucune VM provisionnée pour le moment</p>
        <p class="text-neutral-400 text-xs mt-1">Les instances apparaîtront ici une fois démarrées</p>
      </div>
    {/if}
  {/if}
</div>
{/if}
