<script lang="ts">
  import { _ } from 'svelte-i18n';
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api';
  import type { Image, Flavor, Network, Config } from '$lib/type';

  let {
    open = $bindable(),
    images,
    flavors,
    networks,
    configs,
    selectedGroupImage = $bindable(),
    selectedImage = $bindable(),
    selectedFlavor = $bindable(),
    selectedNetwork = $bindable(),
    selectedConfigFile = $bindable(),
    scheduleDay = $bindable(),
    scheduleTime = $bindable(),
    scheduleWindowHours = $bindable(),
    offDays = $bindable(),
    appPort = $bindable(),
    computeMode = $bindable(),
    createError,
    createSuccess,
    handleCreateServerpool,
    getUniqueFirstAlphaBlocks,
    filterImagesByPrefix,
  }: {
    open: boolean;
    images: Image[];
    flavors: Flavor[];
    networks: Network[];
    configs: Config[];
    selectedGroupImage: string | null;
    selectedImage: string | null;
    selectedFlavor: string;
    selectedNetwork: string;
    selectedConfigFile: string;
    scheduleDay: string;
    scheduleTime: string;
    scheduleWindowHours: number | undefined;
    offDays: { monday:boolean; tuesday:boolean; wednesday:boolean; thursday:boolean; friday:boolean; saturday:boolean; sunday:boolean; };
    appPort: number;
    computeMode: boolean;
    createError: string;
    createSuccess: boolean;
    handleCreateServerpool: (e: Event) => void;
    getUniqueFirstAlphaBlocks: (images: Image[]) => string[];
    filterImagesByPrefix: (images: Image[], prefix: string) => Image[];
  } = $props();

  // Estimateur de coût (F2) : tarifs récupérés du backend.
  let pricing = $state<{ currency: string; vcpu_hour: number; gb_hour: number } | null>(null);

  // Presets de pool : config de création sauvegardée, réapplicable.
  interface Preset { id: number; name: string; image: string; flavor: string; network: string; config: string; app_port: number; off_days: string; compute_mode: boolean; }
  let presets = $state<Preset[]>([]);
  let selectedPresetId = $state('');

  async function loadPresets() {
    try { const r = await apiFetch('/api/pool/presets'); if (r.ok) presets = (await r.json()).presets ?? []; } catch { /* ignore */ }
  }
  onMount(async () => {
    try { const r = await apiFetch('/api/pricing'); if (r.ok) pricing = await r.json(); } catch { /* ignore */ }
    loadPresets();
  });

  const DAYS = ['monday','tuesday','wednesday','thursday','friday','saturday','sunday'] as const;

  function applyPreset(idStr: string) {
    const p = presets.find((x) => String(x.id) === idStr);
    if (!p) return;
    selectedImage = p.image || null;
    selectedFlavor = p.flavor || '';
    selectedNetwork = p.network || '';
    selectedConfigFile = p.config || '';
    appPort = p.app_port || 0;
    computeMode = !!p.compute_mode;
    const set = new Set((p.off_days || '').split(',').map((d) => d.trim()));
    for (const d of DAYS) offDays[d] = set.has(d);
    // Tente de positionner le groupe d'image pour l'aperçu groupé.
    const img = images.find((i) => i.id === p.image);
    if (img) { const m = img.name.match(/^[A-Za-z]+/); selectedGroupImage = m ? m[0] : null; }
  }

  async function saveAsPreset() {
    const name = typeof window !== 'undefined' ? window.prompt($_('poolModal.presetPrompt')) : null;
    if (!name || !name.trim()) return;
    const offCsv = DAYS.filter((d) => offDays[d]).join(',');
    try {
      const r = await apiFetch('/api/pool/presets', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: name.trim(), image: selectedImage ?? '', flavor: selectedFlavor,
          network: selectedNetwork, config: selectedConfigFile, app_port: appPort,
          off_days: offCsv, compute_mode: computeMode,
        }),
      });
      if (r.ok) await loadPresets();
    } catch { /* ignore */ }
  }
  const flavorEstimate = $derived(() => {
    if (!pricing || !selectedFlavor) return null;
    const f = flavors.find((x) => x.id === selectedFlavor);
    if (!f) return null;
    const perHour = f.vcpus * pricing.vcpu_hour + (f.ram / 1024) * pricing.gb_hour;
    return { perHour, perDay: perHour * 24, currency: pricing.currency };
  });

  // Maps snapshot suffix → human label
  const jupyterSnapshotLabels: Record<string, string> = {
    'nfs':         'Jupyter + NFS (Corrigé)',
    'scipy':       'Python scientifique (scipy-notebook)',
    'scipy-plus':  'Python scientifique+',
    'datascience': 'Data Science (Python + R + Julia)',
    'julia':       'Julia',
    'bio583':      'BIO583',
    'eco589':      'ECO589',
    'compeco':     'Computational Economics',
    'mec431':      'MEC431',
    'mec558':      'MEC558',
    'map579':      'MAP579',
    'mec552a':     'MEC552A',
    'mec552b':     'MEC552B',
    'mec568':      'MEC568',
    'mec581':      'MEC581',
    'mec666':      'MEC666',
  };

  function getJupyterSnapshots(): { id: string; label: string }[] {
    return images
      .filter(i => i.name.startsWith('jupyter-snapshot-'))
      .map(i => {
        const suffix = i.name.replace('jupyter-snapshot-', '');
        return { id: i.id, label: jupyterSnapshotLabels[suffix] ?? suffix };
      })
      .sort((a, b) => a.label.localeCompare(b.label));
  }

  const JUPYTER_GROUP = 'JupyterHub';

  function getImageGroups(): string[] {
    // Exclude jupyter-snapshot-* and any jupyterhub* images from regular groups
    const regular = images.filter(i =>
      !i.name.startsWith('jupyter-snapshot-') &&
      !i.name.toLowerCase().startsWith('jupyterhub')
    );
    const groups = getUniqueFirstAlphaBlocks(regular);
    if (getJupyterSnapshots().length > 0) {
      return [JUPYTER_GROUP, ...groups];
    }
    return groups;
  }

  function onGroupChange(group: string) {
    selectedGroupImage = group;
    selectedImage = null;
    appPort = 0;
    selectedConfigFile = '';
  }

  function onJupyterSnapshotChange(imgId: string) {
    selectedImage = imgId;
    appPort = 8888;
    // Auto-select the matching autostart config (jupyter-snapshot-{suffix})
    const img = images.find(i => i.id === imgId);
    if (img) {
      const suffix = img.name.replace('jupyter-snapshot-', '');
      selectedConfigFile = `jupyter-snapshot-${suffix}`;
    }
  }

  const offDayLabels: { key: keyof typeof offDays; labelKey: string }[] = [
    { key: 'monday', labelKey: 'poolModal.dayMon' }, { key: 'tuesday', labelKey: 'poolModal.dayTue' },
    { key: 'wednesday', labelKey: 'poolModal.dayWed' }, { key: 'thursday', labelKey: 'poolModal.dayThu' },
    { key: 'friday', labelKey: 'poolModal.dayFri' }, { key: 'saturday', labelKey: 'poolModal.daySat' },
    { key: 'sunday', labelKey: 'poolModal.daySun' },
  ];

  // Jupyter snapshots don't expose min_disk, so without a default no flavor gets
  // recommended for them. They run a scientific Docker stack → assume ~20 GB.
  const DEFAULT_JUPYTER_DISK_GB = 20;

  function getImageDiskGb(img: Image): number {
    if (img.minDiskGigabytes > 0) return img.minDiskGigabytes;
    if (img.sizeBytes > 0n) return Math.ceil(Number(img.sizeBytes) / (1024 ** 3));
    if (img.name.startsWith('jupyter-snapshot-')) return DEFAULT_JUPYTER_DISK_GB;
    return 0;
  }

  // Recommended = the 2 smallest vd flavors with disk >= needed
  function getRecommendedFlavorIds(): Set<string> {
    if (!selectedImage) return new Set();
    const img = images.find(i => i.id === selectedImage);
    if (!img) return new Set();
    const needed = getImageDiskGb(img);
    if (needed === 0) return new Set();
    const top2 = flavors
      .filter(f => f.name.toLowerCase().startsWith('vd') && f.disk >= needed)
      .sort((a, b) => a.disk - b.disk || a.vcpus - b.vcpus)
      .slice(0, 2);
    return new Set(top2.map(f => f.id));
  }

  function flavorStatus(f: Flavor): 'recommended' | 'ok' | 'incompatible' | 'unknown' {
    if (!selectedImage) return 'unknown';
    const img = images.find(i => i.id === selectedImage);
    if (!img) return 'unknown';
    const needed = getImageDiskGb(img);
    if (needed === 0) return 'unknown';
    if (f.disk < needed) return 'incompatible';
    if (getRecommendedFlavorIds().has(f.id)) return 'recommended';
    return 'ok';
  }

  function formatRam(ram: number): string {
    if (ram <= 0) return '—';
    // OpenStack returns RAM in MB; if value < 16 it was likely already converted to GB
    if (ram < 16) return `${ram} GB`;
    if (ram >= 1024) return `${Math.round(ram / 1024)} GB`;
    return `${ram} MB`;
  }
</script>

{#if open}
  <div class="modal-overlay" role="dialog" aria-modal="true">
    <div class="modal-box modal-box-lg" style="max-height:90vh;overflow-y:auto;">

      <div class="flex items-center justify-between mb-6 pb-5 border-b border-neutral-200 dark:border-neutral-700">
        <div>
          <h3 class="text-lg font-bold text-neutral-900 dark:text-neutral-100">{ $_('poolModal.title') }</h3>
          <p class="text-sm text-neutral-500 dark:text-neutral-400 mt-0.5">{ $_('poolModal.subtitle') }</p>
        </div>
        <button onclick={() => open = false} class="text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200 transition-colors p-1 rounded hover:bg-neutral-100 dark:hover:bg-neutral-800">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
          </svg>
        </button>
      </div>

      {#if createSuccess}
        <div class="flex flex-col items-center justify-center py-16 gap-5 animate-fade-in">
          <div class="w-14 h-14 rounded-full bg-green-50 border border-green-200 flex items-center justify-center">
            <svg class="w-7 h-7 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
            </svg>
          </div>
          <div class="text-center">
            <p class="text-base font-bold text-neutral-900">{ $_('poolModal.successTitle') }</p>
            <p class="text-sm text-neutral-500 mt-1">{ $_('poolModal.successMessage') }</p>
          </div>
          <button onclick={() => open = false} class="btn btn-primary px-8">{ $_('poolModal.ok') }</button>
        </div>
      {:else}

      {#if createError}
        <div class="mb-5 px-4 py-3 rounded bg-red-50 border border-red-200 text-red-700 text-sm animate-fade-in">{createError}</div>
      {/if}

      <form class="space-y-6" onsubmit={handleCreateServerpool}>

        <!-- Presets : appliquer une config sauvegardée / enregistrer la config actuelle -->
        <div class="flex flex-wrap items-center gap-2 px-1">
          <span class="section-label">{ $_('poolModal.presets') }</span>
          <select bind:value={selectedPresetId} onchange={() => applyPreset(selectedPresetId)} class="field text-sm w-auto py-1.5">
            <option value="">{ $_('poolModal.presetApply') }</option>
            {#each presets as p}
              <option value={String(p.id)}>{p.name}</option>
            {/each}
          </select>
          <button type="button" onclick={saveAsPreset} class="btn btn-secondary text-xs">💾 { $_('poolModal.presetSave') }</button>
        </div>

        <!-- Section 1 + 2 -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-5">

          <!-- 1. Général -->
          <div class="card-elevated p-5 space-y-4">
            <h4 class="text-xs font-bold text-primary-700 uppercase tracking-widest border-b border-neutral-200 pb-2">{ $_('poolModal.sectionGeneral') }</h4>

            <div class="space-y-1.5">
              <label class="section-label">{ $_('poolModal.poolName') }</label>
              <input class="field" type="text" name="namesp" placeholder="TP-Reseaux-2026" required />
            </div>

            <div class="grid grid-cols-2 gap-3">
              <div class="space-y-1.5">
                <label class="section-label">{ $_('poolModal.minVms') }</label>
                <input class="field" type="number" name="min_vm" min="1" value="1" required />
              </div>
              <div class="space-y-1.5">
                <label class="section-label">{ $_('poolModal.maxVms') }</label>
                <input class="field" type="number" name="max_vm" min="1" value="5" required />
              </div>
            </div>

            <div class="space-y-1.5">
              <label class="section-label" for="app_port">{ $_('poolModal.appPort') } <span class="text-neutral-400 font-normal">{ $_('poolModal.optional') }</span></label>
              <div class="flex items-center gap-2">
                <input
                  id="app_port"
                  class="field w-36"
                  type="number"
                  min="0" max="65535"
                  bind:value={appPort}
                  placeholder={$_('poolModal.appPortPlaceholder')}
                />
                <p class="text-xs text-neutral-400 leading-snug">
                  { $_('poolModal.appPortHelpBefore') }<b>0</b>{ $_('poolModal.appPortHelpAfter') }
                </p>
              </div>
            </div>

            <!-- Type de pool « calcul » (B7) -->
            <label class="flex items-start gap-2.5 cursor-pointer pt-1">
              <input type="checkbox" bind:checked={computeMode} class="w-4 h-4 mt-0.5 accent-primary-700" />
              <span>
                <span class="text-sm font-medium text-neutral-800 dark:text-neutral-200">{ $_('poolModal.computeMode') }</span>
                <span class="block text-xs text-neutral-400">{ $_('poolModal.computeModeHint') }</span>
              </span>
            </label>
          </div>

          <!-- 2. Infrastructure — OS + Réseau -->
          <div class="card-elevated p-5 space-y-4">
            <h4 class="text-xs font-bold text-primary-700 uppercase tracking-widest border-b border-neutral-200 pb-2">{ $_('poolModal.sectionInfra') }</h4>

            <div class="space-y-1.5">
              <label class="section-label">{ $_('poolModal.operatingSystem') }</label>
              <select class="field" value={selectedGroupImage ?? ''} onchange={(e) => onGroupChange((e.target as HTMLSelectElement).value)} required>
                <option disabled selected value="">{ $_('poolModal.osFamilyPlaceholder') }</option>
                {#each getImageGroups() as group}
                  <option value={group}>{group}</option>
                {/each}
              </select>

              {#if selectedGroupImage === JUPYTER_GROUP}
                <select class="field mt-2" value={selectedImage ?? ''} onchange={(e) => onJupyterSnapshotChange((e.target as HTMLSelectElement).value)} required>
                  <option disabled selected value="">{ $_('poolModal.jupyterEnvPlaceholder') }</option>
                  {#each getJupyterSnapshots() as snap}
                    <option value={snap.id}>{snap.label}</option>
                  {/each}
                </select>
                <p class="text-xs text-neutral-400">{ $_('poolModal.jupyterAutoPort') }</p>
              {:else if selectedGroupImage}
                <select class="field mt-2" bind:value={selectedImage} required>
                  <option disabled selected value="">{ $_('poolModal.exactVersionPlaceholder') }</option>
                  {#each filterImagesByPrefix(images.filter(i => !i.name.startsWith('jupyter-snapshot-') && !i.name.toLowerCase().startsWith('jupyterhub')), selectedGroupImage) as img}
                    <option value={img.id}>{img.name}{img.minDiskGigabytes > 0 ? ` (${img.minDiskGigabytes}${$_('poolModal.gbRequiredSuffix')}` : ''}</option>
                  {/each}
                </select>
              {/if}
            </div>

            <div class="space-y-1.5">
              <label class="section-label">{ $_('poolModal.network') }</label>
              <select class="field" bind:value={selectedNetwork} required>
                <option disabled selected value="">{ $_('poolModal.choosePlaceholder') }</option>
                {#each networks as n}
                  <option value={n.id}>{n.name}</option>
                {/each}
              </select>
            </div>
          </div>
        </div>

        <!-- Flavor — pleine largeur -->
        <div class="card-elevated p-5 space-y-3">
          <h4 class="text-xs font-bold text-primary-700 uppercase tracking-widest border-b border-neutral-200 pb-2">
            Flavor
            {#if selectedImage}
              {@const img = images.find(i => i.id === selectedImage)}
              {#if img}
                {@const needed = getImageDiskGb(img)}
                {#if needed > 0}
                  <span class="text-neutral-400 font-normal normal-case tracking-normal ml-2">{ $_('poolModal.flavorImagePrefix') }{img.name.split('-')[0]}… · {needed}{ $_('poolModal.gbRequiredSuffixNoParen') }</span>
                {/if}
              {/if}
            {/if}
          </h4>

          {#if flavorEstimate()}
            {@const e = flavorEstimate()!}
            <p class="text-xs text-neutral-500 dark:text-neutral-400 mb-2">
              💶 {$_('poolModal.estimate')}
              <b class="text-primary-700 dark:text-primary-300">{e.perHour.toFixed(3)} {e.currency}/h</b>
              · {e.perDay.toFixed(2)} {e.currency}/{$_('poolModal.perDayUnit')} · {$_('poolModal.perVM')}
            </p>
          {/if}

          {#if !selectedImage}
            <select class="field" bind:value={selectedFlavor} required>
              <option disabled selected value="">{ $_('poolModal.selectImageFirst') }</option>
              {#each flavors as f}
                <option value={f.id}>{f.name} — {f.disk} GB · {f.vcpus} vCPU · {formatRam(f.ram)}</option>
              {/each}
            </select>
          {:else}
            {@const vdFlavors = flavors
              .filter(f => f.name.toLowerCase().startsWith('vd'))
              .sort((a, b) => {
                const rank = { recommended: 0, ok: 1, unknown: 2, incompatible: 3 };
                return rank[flavorStatus(a)] - rank[flavorStatus(b)] || a.disk - b.disk || a.vcpus - b.vcpus;
              })}
            {@const otherFlavors = flavors
              .filter(f => !f.name.toLowerCase().startsWith('vd'))
              .sort((a, b) => {
                const rank = { recommended: 0, ok: 1, unknown: 2, incompatible: 3 };
                return rank[flavorStatus(a)] - rank[flavorStatus(b)] || a.name.localeCompare(b.name, undefined, {numeric:true});
              })}

            <div class="grid grid-cols-1 gap-3">
              <!-- vd flavors -->
              {#if vdFlavors.length > 0}
                <div>
                  <p class="section-label mb-2">{ $_('poolModal.vdFlavors') }</p>
                  <div class="border border-neutral-200 dark:border-neutral-700 rounded overflow-hidden divide-y divide-neutral-100 dark:divide-neutral-800">
                    {#each vdFlavors as f}
                      {@const status = flavorStatus(f)}
                      <button
                        type="button"
                        onclick={() => status !== 'incompatible' && (selectedFlavor = f.id)}
                        class="w-full text-left px-4 py-2.5 flex items-center gap-4 transition-colors
                          {selectedFlavor === f.id
                            ? 'bg-primary-50 dark:bg-primary-900/30'
                            : status === 'incompatible'
                              ? 'bg-neutral-50 dark:bg-neutral-800/40 cursor-not-allowed'
                              : 'hover:bg-neutral-50 dark:hover:bg-neutral-800 cursor-pointer'}"
                      >
                        <!-- Selected indicator -->
                        <span class="w-3.5 h-3.5 rounded-full border-2 shrink-0 flex items-center justify-center
                          {selectedFlavor === f.id ? 'border-primary-700 bg-primary-700' : 'border-neutral-300 dark:border-neutral-600'}">
                          {#if selectedFlavor === f.id}
                            <span class="w-1.5 h-1.5 rounded-full bg-white"></span>
                          {/if}
                        </span>

                        <!-- Name -->
                        <span class="text-sm font-bold w-16 shrink-0 {status === 'incompatible' ? 'text-neutral-400 dark:text-neutral-500' : 'text-neutral-900 dark:text-neutral-100'}">{f.name}</span>

                        <!-- Specs -->
                        <span class="text-xs text-neutral-500 dark:text-neutral-400 flex items-center gap-3">
                          <span title={$_('poolModal.disk')}>{f.disk} GB</span>
                          <span class="text-neutral-300">·</span>
                          <span title={$_('poolModal.cpu')}>{f.vcpus} vCPU</span>
                          <span class="text-neutral-300">·</span>
                          <span title={$_('poolModal.ram')}>{formatRam(f.ram)}</span>
                        </span>

                        <!-- Badge -->
                        <span class="ml-auto text-xs font-bold shrink-0
                          {status === 'recommended' ? 'text-green-700 bg-green-50 border border-green-200 px-2 py-0.5 rounded'
                          : status === 'incompatible' ? 'text-red-500'
                          : ''}">
                          {#if status === 'recommended'}★ { $_('poolModal.recommended') }
                          {:else if status === 'incompatible'}✗ { $_('poolModal.diskInsufficient') } ({f.disk} GB &lt; {getImageDiskGb(images.find(i => i.id === selectedImage)!)} GB)
                          {/if}
                        </span>
                      </button>
                    {/each}
                  </div>
                </div>
              {/if}

              <!-- Autres flavors (repliées par défaut) -->
              {#if otherFlavors.length > 0}
                <details class="group">
                  <summary class="section-label cursor-pointer select-none hover:text-neutral-600 list-none flex items-center gap-1">
                    <svg class="w-3 h-3 transition-transform group-open:rotate-90" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
                    </svg>
                    { $_('poolModal.otherFlavors') } ({otherFlavors.length})
                  </summary>
                  <div class="mt-2 border border-neutral-200 dark:border-neutral-700 rounded overflow-hidden divide-y divide-neutral-100 dark:divide-neutral-800">
                    {#each otherFlavors as f}
                      {@const status = flavorStatus(f)}
                      <button
                        type="button"
                        onclick={() => status !== 'incompatible' && (selectedFlavor = f.id)}
                        class="w-full text-left px-4 py-2 flex items-center gap-4 transition-colors
                          {selectedFlavor === f.id
                            ? 'bg-primary-50 dark:bg-primary-900/30'
                            : status === 'incompatible'
                              ? 'bg-neutral-50 dark:bg-neutral-800/40 cursor-not-allowed'
                              : 'hover:bg-neutral-50 dark:hover:bg-neutral-800 cursor-pointer'}"
                      >
                        <span class="w-3.5 h-3.5 rounded-full border-2 shrink-0
                          {selectedFlavor === f.id ? 'border-primary-700 bg-primary-700' : 'border-neutral-300 dark:border-neutral-600'}"></span>
                        <span class="text-sm font-semibold w-32 shrink-0 truncate {status === 'incompatible' ? 'text-neutral-400 dark:text-neutral-500' : 'text-neutral-800 dark:text-neutral-200'}">{f.name}</span>
                        <span class="text-xs text-neutral-400">{f.disk} GB · {f.vcpus} vCPU · {formatRam(f.ram)}</span>
                        {#if status === 'incompatible'}
                          <span class="ml-auto text-xs text-red-400 shrink-0">✗ { $_('poolModal.diskInsufficient') }</span>
                        {/if}
                      </button>
                    {/each}
                  </div>
                </details>
              {/if}
            </div>

            {#if selectedFlavor}
              {@const sel = flavors.find(f => f.id === selectedFlavor)}
              {#if sel}
                <p class="text-xs text-neutral-500">
                  { $_('poolModal.selectedLabel') } <span class="font-bold text-primary-700">{sel.name}</span>
                  <span class="text-neutral-400 ml-1">— {sel.disk} GB · {sel.vcpus} vCPU · {formatRam(sel.ram)}</span>
                </p>
              {/if}
            {/if}
          {/if}
        </div>

        <!-- Section 3: Options avancées -->
        <div class="card-elevated p-5 space-y-5">
          <h4 class="text-xs font-bold text-primary-700 uppercase tracking-widest border-b border-neutral-200 pb-2">{ $_('poolModal.sectionAdvanced') }</h4>

          <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">

            <!-- Left: Config + off days -->
            <div class="space-y-4">
              <div class="space-y-1.5">
                <label class="section-label">{ $_('poolModal.initScript') }</label>
                <select class="field" bind:value={selectedConfigFile}>
                  <option value="">{ $_('poolModal.noneDefault') }</option>
                  {#each configs as c}
                    <option value={c.name}>{c.name}</option>
                  {/each}
                </select>
              </div>

              <div>
                <p class="section-label mb-2.5 block">{ $_('poolModal.offDays') }</p>
                <div class="flex flex-wrap gap-2">
                  {#each offDayLabels as { key, labelKey }}
                    <button
                      type="button"
                      class="w-9 h-9 rounded text-xs font-bold transition-all
                        {offDays[key]
                          ? 'bg-primary-700 text-white border border-primary-800'
                          : 'bg-white dark:bg-neutral-800 text-neutral-500 dark:text-neutral-300 border border-neutral-300 dark:border-neutral-600 hover:border-primary-400 hover:text-primary-600 dark:hover:border-primary-500'}"
                      onclick={() => offDays[key] = !offDays[key]}
                    >{$_(labelKey)}</button>
                  {/each}
                </div>
                <p class="text-xs text-neutral-400 mt-2">{ $_('poolModal.offDaysHelp') }</p>
              </div>
            </div>

            <!-- Right: Schedule -->
            <div class="space-y-3">
              <p class="section-label block">{ $_('poolModal.startSchedule') }</p>
              <div class="grid grid-cols-3 gap-3">
                <div class="space-y-1.5">
                  <label class="section-label">{ $_('poolModal.day') }</label>
                  <select class="field" bind:value={scheduleDay}>
                    <option value="">{ $_('poolModal.none') }</option>
                    <option value="1">{ $_('poolModal.monday') }</option>
                    <option value="2">{ $_('poolModal.tuesday') }</option>
                    <option value="3">{ $_('poolModal.wednesday') }</option>
                    <option value="4">{ $_('poolModal.thursday') }</option>
                    <option value="5">{ $_('poolModal.friday') }</option>
                  </select>
                </div>
                <div class="space-y-1.5">
                  <label class="section-label">{ $_('poolModal.hour') }</label>
                  <input class="field" type="time" bind:value={scheduleTime} />
                </div>
                <div class="space-y-1.5">
                  <label class="section-label">{ $_('poolModal.durationHours') }</label>
                  <input class="field" type="number" min="1" max="24" bind:value={scheduleWindowHours} placeholder="4" />
                </div>
              </div>
              <p class="text-xs text-neutral-400">{ $_('poolModal.scheduleHelp') }</p>
            </div>

          </div>
        </div>

        <!-- Footer -->
        <div class="flex items-center justify-end gap-3 pt-1">
          <button type="button" onclick={() => open = false} class="btn btn-secondary text-sm">
            { $_('poolModal.cancel') }
          </button>
          <button type="submit" class="btn btn-primary text-sm px-6">
            { $_('poolModal.create') }
          </button>
        </div>

      </form>
      {/if}
    </div>
  </div>
{/if}
