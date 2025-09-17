<script lang="ts">
  export let showModal: boolean;

  let dialog: HTMLDialogElement;
  let closeButton: HTMLButtonElement;

  // Ouvre la modal et focus sur le bouton fermer
  $: if (showModal && dialog) {
    dialog.showModal();
    closeButton?.focus();
  }

  // Fermer en cliquant sur le backdrop
  function handleBackdropClick(e: MouseEvent) {
    if (e.target === dialog) dialog.close();
  }
</script>

<dialog
  bind:this={dialog}
  on:close={() => (showModal = false)}
  on:click={handleBackdropClick}
  class="fixed inset-0 w-full max-w-lg p-0 bg-transparent"
>
  <div class="flex min-h-full items-center justify-center p-4 text-center sm:p-0">
    <div class="relative transform overflow-hidden rounded-lg bg-gray-800 text-left shadow-xl transition-all w-full sm:max-w-lg">
      <div class="px-6 pt-5 pb-4 sm:p-6 sm:pb-4">
        <slot name="header" class="text-white text-xl font-bold mb-4" />
        <slot />
      </div>
      <div class="bg-gray-700/25 px-6 py-3 flex justify-end gap-2">
        <button
          bind:this={closeButton}
          type="button"
          on:click={() => dialog.close()}
          class="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-400"
        >
          Fermer
        </button>
      </div>
    </div>
  </div>
</dialog>

<!-- <dialog
  bind:this={dialog}
  on:close={() => (showModal = false)}
  on:click={handleBackdropClick}
  class="fixed inset-0 flex items-center justify-center bg-black/50 p-4"
>
  <div
    class="relative w-full max-w-lg transform overflow-hidden rounded-lg bg-gray-800 text-left shadow-xl transition-all"
  >
    Contenu principal -->
    <!-- <div class="px-6 pt-5 pb-4 sm:p-6 sm:pb-4">
      <slot name="header" class="text-white text-xl font-bold mb-4" />
      <slot />
    </div> -->

    <!-- Footer -->
    <!-- <div class="bg-gray-700/25 px-6 py-3 flex justify-end gap-2">
      <button
        bind:this={closeButton}
        type="button"
        on:click={() => dialog.close()}
        class="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-400"
      >
        Fermer
      </button>
    </div>
  </div> -->
<!-- </dialog> -->