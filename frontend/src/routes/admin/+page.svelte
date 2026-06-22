<script lang="ts">
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';
  import { browser } from '$app/environment';
  import { goto } from '$app/navigation';
  import { apiFetch } from '$lib/api';
  import { authStore } from '$lib/store/authStore';
  import { meStore } from '$lib/store/meStore';

  interface Alert { vm: string; ip: string; port: number; service: string; }
  interface Orphan { name: string; id: string; ip: string; pool_id: string; user_id: string; }
  interface UserRow { email: string; role: string; }
  interface Stats { pools: number; vms: number; active: number; users: number; orphans: number; alerts: number; }

  let stats = $state<Stats | null>(null);
  let alerts = $state<Alert[]>([]);
  let orphans = $state<Orphan[]>([]);
  let users = $state<UserRow[]>([]);
  let loading = $state(true);
  let busy = $state('');
  let msg = $state('');

  async function load() {
    loading = true;
    try {
      const r = await apiFetch('/api/admin/console');
      if (r.ok) {
        const d = await r.json();
        stats = d.stats; alerts = d.alerts ?? []; orphans = d.orphans ?? []; users = d.users ?? [];
      }
    } catch { /* ignore */ }
    finally { loading = false; }
  }

  async function killSwitch() {
    if (!confirm($_('admin.killConfirm'))) return;
    busy = 'kill'; msg = '';
    try {
      const r = await apiFetch('/api/admin/kill-switch', { method: 'POST' });
      const d = await r.json();
      msg = r.ok ? $_('admin.killDone').replace('{n}', String(d.stopped ?? 0)) : ($_('admin.actionError'));
    } catch { msg = $_('admin.actionError'); }
    finally { busy = ''; load(); }
  }

  async function cleanupOrphans() {
    if (!confirm($_('admin.cleanupConfirm'))) return;
    busy = 'cleanup'; msg = '';
    try {
      const r = await apiFetch('/api/admin/cleanup-orphans', { method: 'POST' });
      const d = await r.json();
      msg = r.ok ? $_('admin.cleanupDone').replace('{n}', String(d.removed ?? 0)) : ($_('admin.actionError'));
    } catch { msg = $_('admin.actionError'); }
    finally { busy = ''; load(); }
  }

  onMount(() => {
    if (!browser) return;
    const isAdmin = $meStore?.is_admin ?? ($authStore?.role === 'admin');
    if (!isAdmin) { goto('/home'); return; }
    load();
  });
</script>

<svelte:head><title>{$_('admin.pageTitle')}</title></svelte:head>

<div class="space-y-6 animate-fade-up">
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-3xl font-bold text-primary-800 dark:text-primary-300">{$_('admin.title')}</h1>
      <p class="text-sm text-neutral-500 mt-1">{$_('admin.subtitle')}</p>
    </div>
    <button onclick={load} disabled={loading} class="btn btn-secondary text-sm">{$_('admin.refresh')}</button>
  </div>

  {#if msg}<div class="px-4 py-2 rounded-lg bg-primary-50 text-primary-800 text-sm dark:bg-primary-900/20 dark:text-primary-200">{msg}</div>{/if}

  {#if stats}
    <div class="grid grid-cols-3 md:grid-cols-6 gap-3">
      {#each [
        { k: $_('admin.statPools'), v: stats.pools, c: 'text-primary-700' },
        { k: $_('admin.statVms'), v: stats.vms, c: 'text-primary-700' },
        { k: $_('admin.statActive'), v: stats.active, c: 'text-green-600' },
        { k: $_('admin.statUsers'), v: stats.users, c: 'text-primary-700' },
        { k: $_('admin.statOrphans'), v: stats.orphans, c: stats.orphans > 0 ? 'text-amber-600' : 'text-neutral-500' },
        { k: $_('admin.statAlerts'), v: stats.alerts, c: stats.alerts > 0 ? 'text-red-600' : 'text-green-600' },
      ] as s}
        <div class="card p-4"><p class="section-label">{s.k}</p><p class="text-2xl font-bold tabular-nums mt-1 {s.c}">{s.v}</p></div>
      {/each}
    </div>
  {/if}

  <!-- Alertes de sécurité (G1) -->
  <div class="card p-5">
    <h2 class="text-sm font-bold text-neutral-800 dark:text-neutral-200 mb-2">🔒 {$_('admin.securityAlerts')}</h2>
    {#if alerts.length === 0}
      <p class="text-sm text-green-600">{$_('admin.noAlerts')}</p>
    {:else}
      <p class="text-xs text-neutral-500 mb-2">{$_('admin.alertsHint')}</p>
      <div class="space-y-1.5">
        {#each alerts as a}
          <div class="flex items-center gap-3 text-sm px-3 py-2 rounded-lg bg-red-50 border border-red-200 dark:bg-red-900/20 dark:border-red-800">
            <span class="text-red-600 font-semibold">⚠ {a.service}</span>
            <span class="text-neutral-700 dark:text-neutral-300">{a.vm}</span>
            <span class="text-xs text-neutral-400 font-mono">{a.ip}:{a.port}</span>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- VMs orphelines -->
  <div class="card p-5">
    <div class="flex items-center justify-between mb-2">
      <h2 class="text-sm font-bold text-neutral-800 dark:text-neutral-200">👻 {$_('admin.orphans')}</h2>
      {#if orphans.length > 0}
        <button onclick={cleanupOrphans} disabled={busy === 'cleanup'} class="btn btn-secondary text-xs !text-red-600">{$_('admin.cleanup')}</button>
      {/if}
    </div>
    {#if orphans.length === 0}
      <p class="text-sm text-neutral-400">{$_('admin.noOrphans')}</p>
    {:else}
      <div class="space-y-1 text-sm">
        {#each orphans as o}
          <div class="flex items-center gap-3 text-neutral-700 dark:text-neutral-300">
            <span class="font-medium">{o.name}</span>
            <span class="text-xs text-neutral-400">{o.pool_id || '—'} · {o.ip || '—'}</span>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Utilisateurs -->
  <div class="card overflow-hidden">
    <div class="px-5 py-3 border-b border-neutral-200 dark:border-neutral-700 text-sm font-bold text-neutral-800 dark:text-neutral-200">{$_('admin.users')} ({users.length})</div>
    <div class="divide-y divide-neutral-100 dark:divide-neutral-800 max-h-72 overflow-y-auto">
      {#each users as u}
        <div class="px-5 py-2 flex items-center justify-between text-sm">
          <span class="text-neutral-800 dark:text-neutral-200">{u.email}</span>
          <span class="text-[10px] font-semibold px-2 py-0.5 rounded border border-neutral-200 dark:border-neutral-700 text-neutral-500">{u.role || 'student'}</span>
        </div>
      {/each}
    </div>
  </div>

  <!-- Kill-switch -->
  <div class="card p-5 border-red-200 dark:border-red-900/40">
    <h2 class="text-sm font-bold text-red-700 dark:text-red-400 mb-1">{$_('admin.killTitle')}</h2>
    <p class="text-xs text-neutral-500 mb-3">{$_('admin.killHint')}</p>
    <button onclick={killSwitch} disabled={busy === 'kill'} class="btn btn-danger text-sm">{$_('admin.killButton')}</button>
  </div>
</div>
