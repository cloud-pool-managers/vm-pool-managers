<script lang="ts">
	import { Button, Dropdown, DropdownItem , Label, Textarea, Input } from "flowbite-svelte";
    import { authStore, serverpoolStore, createConfig, updateConfig, deleteConfig } from '$lib/index';
	import { ChevronDownOutline } from "flowbite-svelte-icons";
	import { onMount } from "svelte";
	import type { Config } from "$lib/stores/fetchinit";


    let configs: string = "Configurations";
    let token: string | null = null;
    let textspacedisplay: boolean = false;
    let text: string = "";
    let newconfigname: string = "";
    let configlist: Config[] = [];
    let configid: number = -1;
    
    $: token = $authStore;
    $: configlist = $serverpoolStore.configs;

    $: configid = configlist.find(c => c.name === newconfigname)?.id || -1;



    const handleClickDropdown = async (e: Event) => {
        e.preventDefault();
        const target = e.target as HTMLButtonElement;
        configs = target.name;
        text = configlist.find(c => c.name === target.name)?.data || "";
        textspacedisplay = true;
        newconfigname = target.name;
        configid = configlist.find(c => c.name === target.name)?.id || -1;
    }

    const handleNewConfig = async (e: Event) => {
        e.preventDefault();
        configs = "Configurations";
        text = "";
        textspacedisplay = true;
        newconfigname = "";
        configid = -1;
    }
    
    onMount(async () => {
        if (!token) {
            // Rediriger vers la page de connexion si le token n'existe pas
            window.location.href = '/';
        }
    });

    async function handlecreateConfig() {
        // Logique pour créer une nouvelle configuration
        console.log("Creating new configuration:", newconfigname, text);
        await createConfig(newconfigname, text);
        configs = newconfigname;
    }

    async function handleupdateConfig() {
        console.log("Updating configuration:", configid, newconfigname, text);
        await updateConfig(configid, newconfigname, text);
    }

    async function handledeleteConfig() {
        console.log("Deleting configuration:", configid);
        await deleteConfig(configid);
        configs = "Configurations";
        text = "";
        textspacedisplay = false;
        newconfigname = "";
        configid = -1;
    }

</script>

<Button size="md" class="w-48 h-12">
    {configs} <ChevronDownOutline class="ms-2 h-6 text-white" />
</Button>
<Dropdown simple isOpen={false} class="mt-2">
    {#each configlist as config}
        <DropdownItem name={config.name} onclick={handleClickDropdown}>{config.name}</DropdownItem>
    {/each}
</Dropdown>

<Button size="md" class="w-48 h-12 mt-4" onclick={handleNewConfig}>
    Create a new configuration
</Button>

{#if textspacedisplay}
    <Label for="textarea-id" class="mb-2">Votre script de configuration</Label>
    <Textarea id="textarea-id" placeholder="#!/bin/bash" rows={25} bind:value={text} class="w-full"/>
    <Label for="config-name" class="mb-2 mt-2">Nom de la configuration</Label>
    <Input id="config-name" type="text" placeholder="Configuration Name" class="mt-2 mb-2" bind:value={newconfigname} />
    {#if configid !== -1}
        <Button size="md" class="w-48 h-12 mt-2" onclick={handleupdateConfig}>Update Configuration</Button>
        <Button size="md" class="w-48 h-12 mt-2" onclick={handledeleteConfig}>Delete Configuration</Button>
    {:else}
        <Button size="md" class="w-48 h-12 mt-2" onclick={handlecreateConfig}>Save Configuration</Button>
        <Button size="md" class="w-48 h-12 mt-2" onclick={handledeleteConfig} disabled >Delete Configuration</Button>
    {/if}
{/if}