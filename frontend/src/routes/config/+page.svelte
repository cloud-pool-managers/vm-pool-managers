<script lang="ts">
	import {
        Button,
        Dropdown,
        DropdownItem,
        Label,
        Textarea,
        Input,
    } from "flowbite-svelte";
    import { createConfig, updateConfig, deleteConfig } from '$lib/index';
    import { authStore, configs} from '$lib/store'
	import { ChevronDownOutline } from "flowbite-svelte-icons";
	import { onMount } from "svelte";
    import type { Config } from "$lib/type";

    let config_name: string = "Configurations";
    let textspacedisplay: boolean = false;
    let text: string = "";
    let newconfigname: string = "";
    let configlist: Config[] = [];
    let token: string | null = null;

    $: configlist = $configs;
    $: token = $authStore?.token ?? null;



    const handleClickDropdown = async (e: Event) => {
        e.preventDefault();
        const target = e.target as HTMLButtonElement;
        config_name = target.name;
        text = configlist.find(c => c.name === target.name)?.data || "";
        textspacedisplay = true;
        newconfigname = target.name;
    }

    const handleNewConfig = async (e: Event) => {
        e.preventDefault();
        config_name = "Configurations";
        text = "";
        textspacedisplay = true;
        newconfigname = "";
    }
    
    onMount(async () => {
        if (!token) {
            window.location.href = '/';
        }
    });

    async function handlecreateConfig() {
        // Logique pour créer une nouvelle configuration
        console.log("Creating new configuration:", newconfigname, text);
        await createConfig($authStore?.email ?? "",newconfigname, text);
        config_name = newconfigname;
    }

    async function handleupdateConfig() {
        console.log("Updating configuration:", newconfigname, text);
        await updateConfig($authStore?.email ?? "", newconfigname, text);
    }

    async function handledeleteConfig() {
        console.log("Deleting configuration:", config_name);
        await deleteConfig($authStore?.email ?? "", newconfigname);
        config_name = "Configurations";
        text = "";
        textspacedisplay = false;
        newconfigname = "";
    }

</script>

<!-- Header Actions -->
<div class="flex justify-between items-center mb-6 mt-4">
  <div class="flex gap-4">
    <Button size="md" class="w-64 h-12 bg-tertiary-400 hover:bg-tertiary-500 text-white shadow-md">
      {config_name} <ChevronDownOutline class="ms-2 h-6 text-white" />
    </Button>
    <Dropdown simple isOpen={false} class="mt-2 bg-tertiary-300 border-tertiary-200">
      {#each configlist as config}
        <DropdownItem name={config.name} onclick={handleClickDropdown} class="hover:bg-tertiary-400 text-white">
            {config.name}
        </DropdownItem>
      {/each}
    </Dropdown>
  </div>

  <Button
    size="lg"
    class="bg-option-500 hover:bg-option-600 shadow-lg px-6 py-2.5 font-semibold transition-transform hover:scale-105"
    onclick={handleNewConfig}>
      + Nouvelle configuration
  </Button>
</div>

<!-- Main Dashboard Area -->
{#if textspacedisplay}
  <div class="mt-8 bg-tertiary-300 rounded-xl p-8 shadow-xl border border-tertiary-200">
    <div class="mb-6 border-b border-tertiary-200 pb-4">
        <h2 class="text-2xl font-bold text-white tracking-wide">Éditeur de Configuration</h2>
        <p class="text-gray-400 mt-1">Créez ou modifiez le script d'initialisation de vos VMs.</p>
    </div>

    <div class="mb-6">
        <Label for="config-name" class="mb-2 text-gray-300 font-semibold">Nom de la configuration</Label>
        <Input id="config-name" type="text" placeholder="Ex: setup_web_server"
            class="bg-tertiary-400 text-white border-tertiary-200 focus:ring-option-500 focus:border-option-500" bind:value={newconfigname} />
    </div>

    <div class="mb-6">
        <Label for="textarea-id" class="mb-2 text-gray-300 font-semibold">Script bash (cloud-init)</Label>
        <Textarea id="textarea-id" placeholder="#!/bin/bash&#10;apt-get update..."
             rows={18} bind:value={text} class="font-mono text-sm bg-tertiary-400 text-gray-200 border-tertiary-200 focus:ring-option-500 focus:border-option-500"/>
    </div>

    <div class="flex flex-wrap gap-4 pt-4 border-t border-tertiary-200">
        {#if config_name !== newconfigname}
            <Button size="lg" class="bg-option-500 hover:bg-option-600 shadow-md text-white font-semibold" onclick={handlecreateConfig}>
                Enregistrer la configuration
            </Button>
            <div class="flex-grow"></div>
            <Button size="lg" class="bg-tertiary-500 text-gray-400 cursor-not-allowed shadow-md border border-tertiary-400" disabled>
                Supprimer
            </Button>
        {:else}
            <Button size="lg" class="bg-option-500 hover:bg-option-600 shadow-md text-white font-semibold" onclick={handleupdateConfig}>
                Mettre à jour
            </Button>
            <div class="flex-grow"></div>
            <Button size="lg" class="bg-red-600/90 hover:bg-red-600 shadow-md text-white font-semibold" onclick={handledeleteConfig}>
                Supprimer
            </Button>
        {/if}
    </div>
  </div>
{:else}
  <div class="mt-12 bg-tertiary-300 rounded-2xl p-16 text-center border border-tertiary-200 shadow-lg flex flex-col items-center justify-center h-96">
    <div class="text-6xl mb-6 opacity-50">📜</div>
    <h3 class="text-2xl font-bold text-white mb-3">Aucune configuration sélectionnée</h3>
    <p class="text-gray-400 text-lg max-w-md">Sélectionnez une configuration existante via le menu déroulant ci-dessus, ou créez-en une nouvelle.</p>
  </div>
{/if}