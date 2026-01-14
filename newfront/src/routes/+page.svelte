<script lang="ts">
  import { Button, Modal, Textarea } from "flowbite-svelte";
  import { returnPoolsWithKey, attribVMinPool } from "$lib/grpc/attribVMService/attribVMService";

  let modalssh = false;
  let sshkey = "";
  let availablePools: { pool_id: string; user_id: string }[] = [];
  let selectedPool: { pool_id: string; user_id: string } | null = null;
  let vmIp = "";
  let loading = false;
  let errorMsg = "";

  // Recherche des pools correspondant à la clé SSH
  async function handleSSHKey() {
    if (!sshkey.trim()) return;

    loading = true;
    errorMsg = "";
    availablePools = [];
    selectedPool = null;
    vmIp = "";

    try {
      availablePools = await returnPoolsWithKey(sshkey);
      if (availablePools.length === 0) {
        errorMsg = "Aucun pool disponible pour cette clé";
      }
    } catch (err) {
      console.error(err);
      errorMsg = "Erreur lors de la récupération des pools";
    } finally {
      loading = false;
    }
  }

  // Attribution d'une VM dans le pool choisi
  async function assignVM(pool: { pool_id: string; user_id: string }) {
    selectedPool = pool;
    loading = true;
    errorMsg = "";
    vmIp = "";

    try {
      vmIp = await attribVMinPool(pool.pool_id, pool.user_id, sshkey);
    } catch (err: any) {
      console.error(err);
      errorMsg = err?.message || "Erreur lors de l'attribution de la VM";
    } finally {
      loading = false;
    }
  }
</script>

<!-- Bouton pour ouvrir le modal -->
<Button class="mt-6 bg-option-500" onclick={() => modalssh = true}>
  Rechercher les pools disponibles
</Button>

<!-- Modal -->
{#if modalssh}
  <Modal bind:open={modalssh} focustrap>
    <h2 class="text-lg font-bold mb-2">Entrez votre clé SSH publique</h2>
    
    <Textarea
      placeholder="SSH Key"
      class="w-full h-20"
      bind:value={sshkey}
    />

    <Button
      class="mt-4 bg-option-500"
      onclick={handleSSHKey}
      disabled={loading}
    >
      {loading ? "Recherche en cours..." : "Rechercher"}
    </Button>

    {#if errorMsg}
      <p class="text-red-500 mt-2">{errorMsg}</p>
    {/if}

    {#if availablePools.length > 0}
      <h3 class="mt-4 font-semibold">Pools disponibles :</h3>
      <ul>
        {#each availablePools as pool}
          <li class="flex justify-between items-center mt-2">
            <span>{pool.pool_id} ({pool.user_id})</span>
            <Button
              size="sm"
              class="ml-2"
              onclick={() => assignVM(pool)}
              disabled={loading && selectedPool === pool}
            >
              {loading && selectedPool === pool ? "Attribution..." : "Attribuer VM"}
            </Button>
          </li>
        {/each}
      </ul>
    {/if}

    {#if vmIp}
      <p class="mt-4 font-bold text-green-600">
        VM attribuée ! Adresse IP : {vmIp}
      </p>
    {/if}
  </Modal>
{/if}
