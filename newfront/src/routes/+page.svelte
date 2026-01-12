<script lang="ts">
  import {
    images,
    flavors,
    networks,
    servers,
    serverPools,
    configs
  } from "$lib/store";
	import { Button, modal, Modal, Textarea } from "flowbite-svelte";

  let modalssh = false;
  let sshkey = "";
  function handleSSHKey() {
    //search for pools with sshkey in Keypublist
  }
</script>

<Button class="mt-6 bg-option-500"
  onclick={() => modalssh = true}>
  Rechercher les pools disponibles
</Button>

{#if modalssh}
  <Modal
  bind:open={modalssh}
  focustrap>
  <Textarea
    placeholder="Entrez votre clé SSH publique"
    class="w-full h-20"
    bind:value={sshkey}/>
  <Button
    size="md"
    class="mt-4 bg-option-500"
    onclick={handleSSHKey}>
  </Button>
  </Modal>
{/if}
