<script lang="ts">
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api';
  import { _ } from 'svelte-i18n';

  let open = $state(false);
  let message = $state('');
  let active = $state(false);
  let saving = $state(false);
  let saved = $state(false);

  async function load() {
    try {
      const r = await apiFetch('/api/announcement');
      if (r.ok) { const d = await r.json(); message = d.message || ''; active = !!d.active; }
    } catch { /* ignore */ }
  }
  async function save() {
    saving = true; saved = false;
    try {
      const r = await apiFetch('/api/admin/announcement', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message, active }),
      });
      if (r.ok) { const d = await r.json(); active = !!d.active; saved = true; setTimeout(() => (saved = false), 1500); }
    } catch { /* ignore */ }
    finally { saving = false; }
  }

  onMount(load);
</script>

<div class="relative">
  <button onclick={() => (open = !open)} title={$_('inventory.announcementTitle')} aria-label={$_('inventory.announcementTitle')}
    class="relative p-2 rounded-full text-neutral-500 dark:text-neutral-400 hover:text-primary-700 dark:hover:text-primary-300 hover:bg-black/5 dark:hover:bg-white/5 transition-colors">
    <svg class="w-[18px] h-[18px]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="M11 5.882V19.24a1.76 1.76 0 01-3.417.592l-2.147-6.15M18 13a3 3 0 100-6M5.436 13.683A4.001 4.001 0 017 6h1.832c4.1 0 7.625-1.234 9.168-3v14c-1.543-1.766-5.067-3-9.168-3H7a3.988 3.988 0 01-1.564-.317z"/>
    </svg>
    {#if active}
      <span class="absolute -top-0.5 -right-0.5 w-2.5 h-2.5 bg-amber-500 rounded-full ring-2 ring-white dark:ring-neutral-900"></span>
    {/if}
  </button>

  {#if open}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="fixed inset-0 z-40" onclick={() => (open = false)}></div>
    <div class="absolute right-0 mt-2 w-80 rounded-xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 shadow-xl z-50 p-3 space-y-3">
      <p class="text-xs font-semibold text-neutral-500 dark:text-neutral-400">{$_('inventory.announcementTitle')}</p>
      <textarea bind:value={message} rows="3" placeholder={$_('inventory.announcementPlaceholder')}
        class="field text-sm w-full resize-none"></textarea>
      <div class="flex items-center justify-between">
        <label class="flex items-center gap-2 text-sm text-neutral-600 dark:text-neutral-300">
          <input type="checkbox" bind:checked={active} class="w-4 h-4 accent-primary-700" /> {$_('inventory.showAnnouncement')}
        </label>
        <button onclick={save} disabled={saving} class="btn btn-primary text-sm">
          {saved ? '✓' : $_('common.save')}
        </button>
      </div>
    </div>
  {/if}
</div>
