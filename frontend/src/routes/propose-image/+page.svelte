<script lang="ts">
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';
  import { apiFetch } from '$lib/api';
  import { authStore } from '$lib/store';
  import { browser } from '$app/environment';

  interface Proposal {
    id: number;
    github_url: string;
    name: string;
    description: string;
    status: string;
    created_at: string;
  }

  let githubUrl = $state('');
  let name = $state('');
  let description = $state('');

  let submitting = $state(false);
  let error = $state('');
  let successMsg = $state('');
  let proposals: Proposal[] = $state([]);

  onMount(() => {
    if (!browser) return;
    if (!$authStore || $authStore.role !== 'admin') {
      window.location.href = '/';
      return;
    }
    loadProposals();
  });

  async function loadProposals() {
    if (!$authStore?.email) return;
    try {
      const res = await apiFetch(`/api/image-proposals?user=${encodeURIComponent($authStore.email)}`);
      if (res.ok) proposals = (await res.json()) ?? [];
    } catch { /* ignore */ }
  }

  const githubOk = $derived(/^https?:\/\/github\.com\/.+/.test(githubUrl.trim()));
  const canSubmit = $derived(githubOk && name.trim().length > 0 && !submitting);

  async function submit() {
    error = '';
    successMsg = '';
    if (!canSubmit) {
      if (!githubOk) error = $_('proposeImage.errorInvalidUrl');
      else if (!name.trim()) error = $_('proposeImage.errorNameRequired');
      return;
    }
    submitting = true;
    try {
      const res = await apiFetch('/api/image-proposals', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          github_url: githubUrl.trim(),
          name: name.trim(),
          description: description.trim(),
          submitted_by: $authStore?.email ?? ''
        })
      });
      if (!res.ok) throw new Error(await res.text());
      successMsg = $_('proposeImage.successMsg');
      githubUrl = '';
      name = '';
      description = '';
      await loadProposals();
    } catch (e: any) {
      error = e.message || $_('proposeImage.errorSendFailed');
    } finally {
      submitting = false;
    }
  }

  function statusLabel(s: string): string {
    return s === 'approved' ? $_('proposeImage.statusApproved') : s === 'rejected' ? $_('proposeImage.statusRejected') : $_('proposeImage.statusPending');
  }
  function statusClass(s: string): string {
    if (s === 'approved') return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300';
    if (s === 'rejected') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300';
    return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300';
  }
  function fmtDate(s: string): string {
    try { return new Date(s).toLocaleString('fr-FR', { dateStyle: 'medium', timeStyle: 'short' }); }
    catch { return s; }
  }
</script>

<svelte:head><title>{$_('proposeImage.pageTitle')}</title></svelte:head>

<div class="max-w-3xl mx-auto px-6 py-8 animate-fade-up">
  <h1 class="text-2xl font-bold text-primary-800 dark:text-primary-300">{$_('proposeImage.heading')}</h1>
  <p class="text-sm text-neutral-500 dark:text-neutral-400 mt-1">
    {$_('proposeImage.intro')}
  </p>

  <!-- Formulaire -->
  <form class="card p-6 mt-6 space-y-4" onsubmit={(e) => { e.preventDefault(); submit(); }}>
    <div>
      <label for="gh" class="section-label block mb-1.5">{$_('proposeImage.githubLabel')}</label>
      <input
        id="gh"
        type="url"
        bind:value={githubUrl}
        placeholder={$_('proposeImage.githubPlaceholder')}
        class="field w-full {githubUrl && !githubOk ? 'border-red-400' : ''}"
        autocomplete="off"
      />
      <p class="text-xs text-neutral-400 mt-1">{$_('proposeImage.githubHelpStart')}<code>environment.yml</code>, <code>requirements.txt</code>, <code>Project.toml</code>, <code>postBuild</code>{$_('proposeImage.githubHelpEnd')}</p>
    </div>

    <div>
      <label for="nm" class="section-label block mb-1.5">{$_('proposeImage.nameLabel')}</label>
      <input id="nm" type="text" bind:value={name} placeholder={$_('proposeImage.namePlaceholder')} class="field w-full" autocomplete="off" />
    </div>

    <div>
      <label for="desc" class="section-label block mb-1.5">{$_('proposeImage.descLabel')}</label>
      <textarea id="desc" bind:value={description} rows="4" placeholder={$_('proposeImage.descPlaceholder')} class="field w-full resize-y"></textarea>
    </div>

    {#if error}
      <div class="px-3 py-2.5 rounded bg-red-50 border border-red-200 text-red-700 dark:bg-red-900/20 dark:border-red-800 dark:text-red-300 text-sm">{error}</div>
    {/if}
    {#if successMsg}
      <div class="px-3 py-2.5 rounded bg-green-50 border border-green-200 text-green-700 dark:bg-green-900/20 dark:border-green-800 dark:text-green-300 text-sm">{successMsg}</div>
    {/if}

    <div class="flex justify-end">
      <button type="submit" disabled={!canSubmit} class="btn btn-primary gap-2 disabled:opacity-50">
        {#if submitting}
          <span class="w-3.5 h-3.5 border-2 border-white/30 border-t-white rounded-full" style="animation:spinnerGlow 0.6s linear infinite;"></span>
        {:else}
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"/></svg>
        {/if}
        {$_('proposeImage.submitButton')}
      </button>
    </div>
  </form>

  <!-- Mes propositions -->
  <div class="mt-8">
    <p class="section-label mb-3">{$_('proposeImage.myProposals')}</p>
    {#if proposals.length === 0}
      <div class="card p-6 text-center text-sm text-neutral-400">{$_('proposeImage.noProposals')}</div>
    {:else}
      <div class="card overflow-hidden divide-y divide-neutral-100 dark:divide-neutral-800">
        {#each proposals as p}
          <div class="px-4 py-3 flex items-start justify-between gap-4">
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="font-semibold text-sm text-neutral-800 dark:text-neutral-200 truncate">{p.name}</span>
                <span class="text-[10px] font-bold px-2 py-0.5 rounded-full {statusClass(p.status)}">{statusLabel(p.status)}</span>
              </div>
              <a href={p.github_url} target="_blank" rel="noopener noreferrer" class="text-xs text-primary-600 dark:text-primary-400 hover:underline font-mono truncate block max-w-md">{p.github_url}</a>
              {#if p.description}<p class="text-xs text-neutral-500 mt-1 line-clamp-2">{p.description}</p>{/if}
            </div>
            <span class="text-[10px] text-neutral-400 shrink-0 whitespace-nowrap">{fmtDate(p.created_at)}</span>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
