<script lang="ts">
  import {
    Modal,
    Button,
    Label,
    Input,
    Select,
    Checkbox,
    Helper
  } from 'flowbite-svelte';

  import type { Image, Flavor, Network, Config } from '$lib/type';

  export let open: boolean;
  
  export let images: Image[];
  export let flavors: Flavor[];
  export let networks: Network[];
  export let configs: Config[];

  export let selectedGroupImage: string | null;
  export let selectedImage: string | null;
  export let selectedFlavor: string;
  export let selectedNetwork: string;
  export let selectedConfigFile: string;
  export let scheduleDay: string;
  export let scheduleTime: string;
  export let scheduleWindowHours: number | undefined;

  export let offDays: {
    monday: boolean;
    tuesday: boolean;
    wednesday: boolean;
    thursday: boolean;
    friday: boolean;
    saturday: boolean;
    sunday: boolean;
  };

  export let createError: string;
  export let createSuccess: boolean;

  export let handleCreateServerpool: (e: Event) => void;
  export let getUniqueFirstAlphaBlocks: (images: Image[]) => string[];
  export let filterImagesByPrefix: (images: Image[], prefix: string) => Image[];

  const offDayLabels: { key: keyof typeof offDays; label: string }[] = [
    { key: 'monday', label: 'Lundi' },
    { key: 'tuesday', label: 'Mardi' },
    { key: 'wednesday', label: 'Mercredi' },
    { key: 'thursday', label: 'Jeudi' },
    { key: 'friday', label: 'Vendredi' },
    { key: 'saturday', label: 'Samedi' },
    { key: 'sunday', label: 'Dimanche' }
  ];
</script>

<Modal bind:open size="xl" autoclose={false} class="w-full">
  <form
    class="flex flex-col space-y-8"
    on:submit|preventDefault={handleCreateServerpool}
  >
    <div class="border-b pb-4 border-gray-200 dark:border-gray-700">
      <h3 class="text-2xl font-semibold text-gray-900 dark:text-white">
        Créer un nouveau Serverpool
      </h3>
      <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
        Configurez un groupe de machines virtuelles pour vos étudiants.
      </p>
    </div>

    {#if createError}
      <div class="p-4 text-sm text-red-800 rounded-lg bg-red-50 dark:bg-gray-800 dark:text-red-400" role="alert">
        <span class="font-medium">Erreur :</span> {createError}
      </div>
    {/if}

    {#if createSuccess}
      <div class="p-4 text-sm text-green-800 rounded-lg bg-green-50 dark:bg-gray-800 dark:text-green-400" role="alert">
        <span class="font-medium">Succès !</span> Le Serverpool a été créé avec succès.
      </div>
    {/if}

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-x-8 gap-y-6">
      <!-- SECTION 1: Informations Générales -->
      <div class="space-y-5 bg-gray-50 p-5 rounded-xl border border-gray-100 dark:bg-gray-800 dark:border-gray-700">
        <h4 class="text-lg font-medium text-gray-800 dark:text-gray-200 border-b border-gray-200 dark:border-gray-700 pb-2">1. Informations Générales</h4>
        
        <Label class="space-y-2 block">
          <span class="text-sm font-semibold">Nom du Serverpool</span>
          <Input type="text" name="namesp" placeholder="Ex: TP-Reseaux-2026" required />
        </Label>

        <div class="grid grid-cols-2 gap-4">
          <Label class="space-y-2 block">
            <span class="text-sm font-semibold">Minimum de VMs</span>
            <Input type="number" name="min_vm" min="1" value="1" required />
          </Label>
          <Label class="space-y-2 block">
            <span class="text-sm font-semibold">Maximum de VMs</span>
            <Input type="number" name="max_vm" min="1" value="5" required />
          </Label>
        </div>
      </div>

      <!-- SECTION 2: Infrastructure -->
      <div class="space-y-5 bg-gray-50 p-5 rounded-xl border border-gray-100 dark:bg-gray-800 dark:border-gray-700">
        <h4 class="text-lg font-medium text-gray-800 dark:text-gray-200 border-b border-gray-200 dark:border-gray-700 pb-2">2. Infrastructure OpenStack</h4>

        <Label class="space-y-2 block">
          <span class="text-sm font-semibold">Système d'exploitation (Image)</span>
          <Select bind:value={selectedGroupImage} required class="mb-3">
            <option disabled selected value="">Choisir une famille d'OS...</option>
            {#each getUniqueFirstAlphaBlocks(images) as prefix}
              <option value={prefix}>{prefix}</option>
            {/each}
          </Select>

          {#if selectedGroupImage}
            <Select bind:value={selectedImage} required>
              <option disabled selected value="">Choisir la version exacte...</option>
              {#each filterImagesByPrefix(images, selectedGroupImage) as img}
                <option value={img.id}>{img.name}</option>
              {/each}
            </Select>
          {/if}
        </Label>

        <div class="grid grid-cols-2 gap-4">
          <Label class="space-y-2 block">
            <span class="text-sm font-semibold">Puissance (Flavor)</span>
            <Select bind:value={selectedFlavor} required>
              <option disabled selected value="">Choisir...</option>
              {#each flavors as f}
                <option value={f.id}>{f.name}</option>
              {/each}
            </Select>
          </Label>

          <Label class="space-y-2 block">
            <span class="text-sm font-semibold">Réseau virtuel</span>
            <Select bind:value={selectedNetwork} required>
              <option disabled selected value="">Choisir...</option>
              {#each networks as n}
                <option value={n.id}>{n.name}</option>
              {/each}
            </Select>
          </Label>
        </div>
      </div>
    </div>

    <!-- SECTION 3: Paramètres Avancés -->
    <div class="space-y-5">
      <h4 class="text-lg font-medium text-gray-800 dark:text-gray-200 border-b border-gray-200 dark:border-gray-700 pb-2">
        3. Paramètres Avancés (Optionnel)
      </h4>
      
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
        
        <!-- Scripts Cloud-Init -->
        <div class="space-y-3">
          <Label class="space-y-2 block">
            <span class="text-sm font-semibold text-gray-700 dark:text-gray-300">Script d'initialisation (Config)</span>
            <Select bind:value={selectedConfigFile}>
              <option value="">Aucun (Défaut)</option>
              {#each configs as c}
                <option value={c.name}>{c.name}</option>
              {/each}
            </Select>
            <Helper class="text-xs text-gray-500 mt-1">
              Permet d'exécuter un script bash (cloud-init) au démarrage de la VM pour pré-installer des logiciels spécifiques.
            </Helper>
          </Label>

          <div class="bg-blue-50 p-4 rounded-lg border border-blue-100 dark:bg-blue-900/30 dark:border-blue-800 mt-4">
            <span class="block text-sm font-semibold text-blue-800 dark:text-blue-300 mb-2">Extinction automatique</span>
            <div class="flex flex-wrap gap-x-4 gap-y-2 mt-2">
              {#each offDayLabels as { key, label }}
                <Checkbox bind:checked={offDays[key]} class="text-sm text-blue-900 dark:text-blue-200">{label}</Checkbox>
              {/each}
            </div>
            <Helper class="text-xs mt-2 text-blue-700 dark:text-blue-400">Suspend les VMs durant ces jours pour libérer les serveurs physiques.</Helper>
          </div>
        </div>

        <!-- Planning d'allumage -->
        <div class="space-y-3">
          <span class="block text-sm font-semibold text-gray-700 dark:text-gray-300">Heure d'ouverture du cours</span>
          
          <div class="grid grid-cols-3 gap-3">
            <div>
              <Label class="text-xs mb-1 text-gray-500">Jour</Label>
              <Select bind:value={scheduleDay}>
                <option value="">Aucun</option>
                <option value="1">Lundi</option>
                <option value="2">Mardi</option>
                <option value="3">Mercredi</option>
                <option value="4">Jeudi</option>
                <option value="5">Vendredi</option>
              </Select>
            </div>
            
            <div>
              <Label class="text-xs mb-1 text-gray-500">Heure de début</Label>
              <Input type="time" bind:value={scheduleTime} />
            </div>
            
            <div>
              <Label class="text-xs mb-1 text-gray-500">Durée (h)</Label>
              <Input type="number" min="1" max="24" bind:value={scheduleWindowHours} placeholder="ex: 4" />
            </div>
          </div>
          
          <Helper class="text-xs text-gray-500 mt-1">
            Si configuré, les VMs démarreront automatiquement à l'heure indiquée. Laissez vide pour une gestion manuelle.
          </Helper>
        </div>

      </div>
    </div>

    <!-- Actions -->
    <div class="flex justify-end gap-3 pt-6 border-t border-gray-200 dark:border-gray-700 mt-4">
      <Button color="alternative" on:click={() => open = false}>
        Annuler
      </Button>
      <Button type="submit" color="primary" class="px-8 font-medium">
        Créer le Serverpool
      </Button>
    </div>
  </form>
</Modal>
