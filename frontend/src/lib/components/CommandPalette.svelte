<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { apiFetch } from '$lib/api';
  import { authStore } from '$lib/store/authStore';
  import { meStore } from '$lib/store/meStore';
  import { _ } from 'svelte-i18n';

  interface Item { label: string; sub: string; href: string; }

  let open = $state(false);
  let q = $state('');
  let sel = $state(0);
  let pools = $state<{ label: string; sub: string; id: string }[]>([]);
  let inputEl: HTMLInputElement | null = $state(null);

  const isStaff = $derived($meStore?.is_staff ?? ($authStore?.role === 'admin'));
  const isAdmin = $derived($meStore?.is_admin ?? ($authStore?.role === 'admin'));

  const navItems = $derived(() => {
    const items: Item[] = [];
    const goTo = $_('palette.goTo');
    if (isStaff) {
      if (isAdmin) items.push({ label: $_('nav.inventory'), sub: goTo, href: '/inventory' });
      items.push({ label: $_('nav.serverpools'), sub: goTo, href: '/serverpool' });
      items.push({ label: $_('nav.grading'), sub: goTo, href: '/grading' });
      if (isAdmin) {
        items.push({ label: $_('nav.configs'), sub: goTo, href: '/config' });
        items.push({ label: $_('nav.proposeImage'), sub: goTo, href: '/propose-image' });
      }
    }
    items.push({ label: $_('common.settings'), sub: goTo, href: '/profile' });
    return items;
  });

  const results = $derived(() => {
    const query = q.trim().toLowerCase();
    const nav = navItems().filter(i => !query || i.label.toLowerCase().includes(query));
    const poolItems: Item[] = (query
      ? pools.filter(p => (p.label + ' ' + p.sub).toLowerCase().includes(query))
      : pools
    ).slice(0, 8).map(p => ({ label: p.label, sub: $_('palette.pool') + ' · ' + p.sub, href: '/serverpool/' + p.id }));
    return [...nav, ...poolItems];
  });

  async function loadPools() {
    if (!isAdmin) return;
    try {
      const r = await apiFetch('/api/inventory');
      if (r.ok) {
        const data = await r.json();
        pools = (data ?? []).map((p: any) => ({ label: p.label || p.pool_id, sub: p.user_id, id: p.pool_id }));
      }
    } catch { /* ignore */ }
  }

  function openPalette() { open = true; q = ''; sel = 0; loadPools(); setTimeout(() => inputEl?.focus(), 30); }
  function close() { open = false; }
  function activate(i: Item) { close(); goto(i.href); }

  function onKey(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault();
      if (open) close(); else openPalette();
      return;
    }
    if (!open) return;
    const r = results();
    if (e.key === 'Escape') close();
    else if (e.key === 'ArrowDown') { e.preventDefault(); sel = Math.min(sel + 1, r.length - 1); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); sel = Math.max(sel - 1, 0); }
    else if (e.key === 'Enter') { e.preventDefault(); if (r[sel]) activate(r[sel]); }
  }

  onMount(() => {
    window.addEventListener('keydown', onKey);
    window.addEventListener('open-command-palette', openPalette);
    return () => {
      window.removeEventListener('keydown', onKey);
      window.removeEventListener('open-command-palette', openPalette);
    };
  });
</script>

{#if open}
  <div class="fixed inset-0 z-[60] flex items-start justify-center pt-[12vh] px-4">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="fixed inset-0 bg-neutral-900/40 backdrop-blur-sm" onclick={close}></div>
    <div class="relative w-full max-w-lg bg-white dark:bg-neutral-800 rounded-xl shadow-2xl overflow-hidden">
      <input bind:this={inputEl} bind:value={q} oninput={() => (sel = 0)}
        placeholder={$_('palette.placeholder')}
        class="w-full px-4 py-3 text-sm bg-transparent outline-none border-b border-neutral-100 dark:border-neutral-700 text-neutral-800 dark:text-neutral-100" />
      <div class="max-h-80 overflow-y-auto py-1">
        {#each results() as item, i}
          <button onclick={() => activate(item)} onmouseenter={() => (sel = i)}
            class="w-full text-left px-4 py-2.5 flex items-center justify-between {i === sel ? 'bg-primary-50 dark:bg-primary-900/30' : ''}">
            <span class="text-sm text-neutral-800 dark:text-neutral-200">{item.label}</span>
            <span class="text-xs text-neutral-400">{item.sub}</span>
          </button>
        {/each}
        {#if results().length === 0}
          <p class="text-sm text-neutral-400 px-4 py-6 text-center">{$_('common.noResult')}</p>
        {/if}
      </div>
      <div class="px-4 py-2 border-t border-neutral-100 dark:border-neutral-700 text-[11px] text-neutral-400">{$_('palette.hint')}</div>
    </div>
  </div>
{/if}
