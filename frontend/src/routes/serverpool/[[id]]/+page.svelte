<script lang="ts">
import {
  Button,
  Dropdown,
  DropdownItem,
  Modal,
  Label,
  Input,
  Select,
  MultiSelect,
  Clipboard,
  Textarea,
} from 'flowbite-svelte';
import { CheckOutline, ChevronDownOutline } from 'flowbite-svelte-icons';
import {
  rebuildServer,
  RebuildServerRequestSchema,
  CreatePoolRequestSchema,
  DeletePoolRequestSchema,
  deletePool,
  createPool,
  addServer,
  addSSHKeys,
} from '$lib/index';
import type {
  ServerPool,
  Server,
  CreatePoolRequest,
  DeletePoolRequest,
  RebuildServerRequest,
  Image
} from '$lib/type';
import {
  authStore,
  serverPools,
  servers,
  configs,
  images,
  flavors,
  networks
} from '$lib/store';
import { onMount } from 'svelte';
import { page } from '$app/state';
import { create } from '@bufbuild/protobuf';
import {
	ListSSHPublicKeysRequestSchema,
  type DeletePoolResponse,
  type RebuildServerResponse,
  
} from '$lib/grpc/frontcontrol_pb';
import CreateServerPoolModal from '$lib/components/CreateServerPoolModal.svelte';
import AddSSHKeys from '$lib/components/AddSSHKeys.svelte';



let token: string | null = null;
let selectedsp: string = 'Choisissez le serverpool';
let serversp: Server[] = [];

let selectedNetwork: string = "";
let selectedFlavor: string = "";
let selectedConfigFile: string = "";
let createspModal: boolean = false;
let createsshModal: boolean = false;
let sshkeys: string = "";
let createError: string = "";
let createSuccess = false;
let scheduleDay: string = "";
let scheduleTime: string = "";
let scheduleWindowHours: number | undefined = undefined;

let offDays = {
  monday: false,
  tuesday: false,
  wednesday: false,
  thursday: false,
  friday: false,
  saturday: true,
  sunday: true
};

let ListStudentModalOpen: boolean = false;

type CreateServerPoolForm = {
    name: string;
    image: string;
    flavor: string;
    networks: string;
    minVm: number;
    maxVm: number;
    config: string;
};

onMount(async() => {
	if (!token) {
		window.location.href = '/';
	}
	selectedsp = page.params.id || 'Choisissez le serverpool';
});

const handleClick = async (e: Event) => {
	e.preventDefault();
	const target = e.target as HTMLButtonElement;
	selectedsp = target.name;
};

$: token = $authStore?.token ?? null;
$: selectedPool = $serverPools.find(p => p.name === selectedsp);


$: networkOptions = $networks.map(net => ({
    value: net.id,
    name: net.name,
  }));
  
  $: sortedFlavors = [...$flavors].sort((a, b) =>
  a.name.localeCompare(b.name, undefined, {numeric: true, sensitivity:"base"})
);

async function handleRebuildServer(serv: Server) {
	if (!confirm(`Voulez-vous rebuild le serveur ${serv.name} ?`)) {
		return;
	}
	const req: RebuildServerRequest = create(RebuildServerRequestSchema,{
    user: $authStore?.email,
    poolId: serv.metadata?.serverpool_id,
    serverId: serv.name
  });
	console.log("Rebuild request: ", req);
  try {
		const res: RebuildServerResponse = await rebuildServer(req);
		if (!res.success) {
      console.error("Erreur rebuild server");
		}
	} catch (err) {
		console.error("Erreur rebuild server: ", err);
		throw err;
	}
}

async function handleDeleteServerpool(sp: ServerPool) {
	if (!confirm(`Voulez-vous supprimer le serveur ${sp.name} ?`)) {
		return;
	}
	const req: DeletePoolRequest = create(DeletePoolRequestSchema,{
    user: $authStore?.email,
    poolId: sp.name
  });
	try {
		const res: DeletePoolResponse = await deletePool(req);
		if (res.success) {
      selectedsp = "Choisissez le serverpool";
      const { loadServerPools } = await import('$lib/store/serverpoolStore');
      await loadServerPools($authStore?.email ?? "");
		}
	} catch (err) {
		console.error("Erreur lors de la suppression du pool: ", err);
		throw err;
	}
}

async function handleCreateServer(sp: ServerPool) {
  if (!confirm(`Voulez-vous ajouter un serveur au serverpool ${sp.name} ?`)) {
    return;
  }
  const req: CreatePoolRequest = create(CreatePoolRequestSchema, {
    user: $authStore?.email,
    name: sp.name,
    image: sp.image,
    flavor: sp.flavor,
    network: sp.network,
    minVm: String(sp.minVm),
    maxVm: String(sp.maxVm),
    config: sp.config,
  });

  try {
    const res: RebuildServerResponse = await addServer(req);
    if (res.success) {
      console.log("Serveur ajouté avec succès au serverpool.");
    } else {
      console.error("Erreur lors de l'ajout du serveur au serverpool.");
    }
  } catch (err) {
    console.error("Impossible d'ajouter le serveur au serverpool.", err);
  }
}

export function getUniqueFirstAlphaBlocks(images: Image[]): string[] {
  const prefixes = images
    .map(img => {
      const match = img.name.match(/^[A-Za-z]+/);
      return match ? match[0] : null;
    })
    .filter((x): x is string => x !== null);

  return Array.from(new Set(prefixes));
}

export function filterImagesByPrefix(images: Image[], prefix:string): Image[] {
  return images.filter(img => img.name.startsWith(prefix));
}

let selectedGroupImage: string | null = null;
let selectedImage: string | null = null;


async function handleCreateServerpool(event: Event) {
    event.preventDefault();

    const form = event.target as HTMLFormElement;
    const formData = new FormData(form);

    const data: CreateServerPoolForm = {
        name: formData.get("namesp") as string,
        image: selectedImage ?? "",
        flavor: selectedFlavor,
        networks: selectedNetwork,
        minVm: Number(formData.get("min_vm")),
        maxVm: Number(formData.get("max_vm")),
        config: selectedConfigFile,
    };

    
    if (!data.name?.trim()) {
      createError = "Le nom du serverpool est obligatoire.";
      return;
    }
    if (!data.image || !data.flavor || !data.networks) {
      createError = "Veuillez sélectionner une image, un flavor et un réseau.";
      return;
    }

    const enabledOffDays = Object.entries(offDays)
      .filter(([, enabled]) => enabled)
      .map(([day]) => day);

    const hasSchedule = Boolean(scheduleDay && scheduleTime);
    if ((scheduleDay && !scheduleTime) || (!scheduleDay && scheduleTime)) {
      createError = "Pour le planning, renseignez le jour et l'heure, ou laissez les deux vides.";
      return;
    }

    console.log(" Creating pool:", data);

    const reqPayload: CreatePoolRequest = {
        user: $authStore?.email ?? "",
        name: data.name,
        image: data.image,
        flavor: data.flavor,
        network: data.networks,
        minVm: String(data.minVm),
        maxVm: String(data.maxVm),
        config: data.config ?? "",
        metadata: enabledOffDays.length > 0 ? { off_days: enabledOffDays.join(",") } : {},
        timeWindow: 0,
    };

    if (hasSchedule) {
      const startDate = computeNextSchedule(Number(scheduleDay), scheduleTime);
      reqPayload.startTime = {
        seconds: BigInt(Math.floor(startDate.getTime() / 1000)),
        nanos: (startDate.getTime() % 1000) * 1_000_000,
      };
      if (scheduleWindowHours != null && scheduleWindowHours > 0) {
        reqPayload.timeWindow = scheduleWindowHours;
      }
    }

    const req: CreatePoolRequest = create(CreatePoolRequestSchema, reqPayload);

    console.log(req)

    try {
        createError = "";
        const res = await createPool(req);

        if (res.success) {
            createSuccess = true;
            const { loadServerPools } = await import('$lib/store/serverpoolStore');
            await loadServerPools($authStore?.email ?? "");
            setTimeout(() => (createspModal = false), 1200);
        } else {
            createError = "Erreur lors de la création du serverpool.";
        }
    } catch (err) {
        console.error(err);
        createError = "Impossible de créer le serverpool.";
    }
}

function computeNextSchedule(dayOfWeek: number, time: string): Date {
  const [hours, minutes] = time.split(":").map(Number);
  const now = new Date();

  const target = new Date(now);
  target.setHours(hours, minutes, 0, 0);

  let delta = dayOfWeek - now.getDay();
  if (delta < 0 || (delta === 0 && target < now)) {
    // Si le jour est déjà passé cette semaine, on ajoute 7 jours
    delta += 7;
  }

  target.setDate(now.getDate() + delta);
  return target;
}

async function handleSendSSHKeys() {
  console.log("Sending SSH keys:", sshkeys);
  const req = create(ListSSHPublicKeysRequestSchema, {
    userId: $authStore?.email,
    serverpoolId: selectedPool?.name ?? "",
    pubkeys: sshkeys.split("\n").map(k => k.trim()).filter(k => k.length > 0),
  });
  try {
    const res = await addSSHKeys(req);
    if (!res.success) {
      console.error("Erreur lors de l'ajout des clés SSH");
    }
  }
  catch (err) {
    console.error("Erreur lors de l'ajout des clés SSH: ", err);
    throw err;
  }
  createsshModal = false;
}

</script>

<!-- Header Actions -->
<div class="flex justify-between items-center mb-6 mt-4">
  <div class="flex gap-4">
    <Button size="md" class="w-64 h-12 bg-tertiary-400 hover:bg-tertiary-500 text-white shadow-md">
      {selectedsp}<ChevronDownOutline class="ms-2 h-6 text-white" />
    </Button>
    <Dropdown simple isOpen={false} class="mt-2 bg-tertiary-300 border-tertiary-200">
      {#each $serverPools as sp}
      <DropdownItem name={sp.name} onclick={handleClick} class="hover:bg-tertiary-400 text-white">{sp.name}</DropdownItem>
      {/each}
    </Dropdown>
  </div>
  
  <Button
    size="lg"
    class="bg-option-500 hover:bg-option-600 shadow-lg px-6 py-2.5 font-semibold transition-transform hover:scale-105"
    onclick={() => createspModal = true}>
      + Créer un serverpool
  </Button>
</div>

<!-- Main Dashboard Area -->
{#if selectedPool}
  <div class="mt-8 bg-tertiary-300 rounded-xl p-8 shadow-xl border border-tertiary-200">
    <!-- Header Card -->
    <div class="flex justify-between items-center mb-8 pb-6 border-b border-tertiary-200">
      <div>
        <h2 class="text-3xl font-bold text-white tracking-wide mb-2">Configuration : <span class="text-option-500">{selectedPool.name}</span></h2>
        <p class="text-gray-400">Gérez les paramètres et les instances de ce pool.</p>
      </div>
      <div class="bg-tertiary-400 px-6 py-3 rounded-lg shadow-inner text-sm text-gray-300 border border-tertiary-200 flex flex-col items-center">
        <span class="text-xs uppercase tracking-widest text-gray-400 mb-1">Objectif VMs</span>
        <span class="text-xl font-bold text-white">{selectedPool.minVm} - {selectedPool.maxVm}</span>
      </div>
    </div>

    <!-- Properties Grid -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-10">
      <div class="bg-tertiary-400 p-6 rounded-xl border border-tertiary-200 shadow-sm transition-all hover:border-option-500">
        <div class="flex items-center gap-3 mb-2">
          <div class="w-8 h-8 rounded-full bg-tertiary-500 flex items-center justify-center">🚀</div>
          <p class="text-xs text-gray-400 uppercase tracking-widest font-semibold">Flavor</p>
        </div>
        <p class="text-lg text-white font-medium break-all mt-2 pl-11">
          {$flavors.find(img => img.id === selectedPool?.flavor)?.name ?? selectedPool?.flavor}
        </p>
      </div>

      <div class="bg-tertiary-400 p-6 rounded-xl border border-tertiary-200 shadow-sm transition-all hover:border-option-500">
        <div class="flex items-center gap-3 mb-2">
          <div class="w-8 h-8 rounded-full bg-tertiary-500 flex items-center justify-center">💿</div>
          <p class="text-xs text-gray-400 uppercase tracking-widest font-semibold">Image</p>
        </div>
        <p class="text-lg text-white font-medium break-all mt-2 pl-11">
          {$images.find(img => img.id === selectedPool?.image)?.name ?? selectedPool?.image}
        </p>
      </div>

      <div class="bg-tertiary-400 p-6 rounded-xl border border-tertiary-200 shadow-sm transition-all hover:border-option-500">
        <div class="flex items-center gap-3 mb-2">
          <div class="w-8 h-8 rounded-full bg-tertiary-500 flex items-center justify-center">🌐</div>
          <p class="text-xs text-gray-400 uppercase tracking-widest font-semibold">Réseau</p>
        </div>
        <p class="text-lg text-white font-medium break-all mt-2 pl-11">
          {$networks.find(img => img.id === selectedPool?.network)?.name ?? selectedPool?.network}
        </p>
      </div>
    </div>

    <!-- Actions Bar -->
    <div class="flex flex-wrap gap-4 bg-tertiary-400/50 p-6 rounded-xl border border-tertiary-200">
      <Button
        size="lg"
        class="bg-option-500 hover:bg-option-600 transition-colors shadow-md"
        onclick={() => handleCreateServer(selectedPool)}>
        Ajouter un serveur au pool
      </Button>
      <Button
        size="lg"
        class="bg-tertiary-500 hover:bg-tertiary-600 transition-colors shadow-md text-white border border-tertiary-200"
        onclick={() => ListStudentModalOpen = true}>
        Liste des étudiants
      </Button>
      <div class="flex-grow"></div>
      <Button
        size="lg"
        class="bg-red-600/90 hover:bg-red-600 transition-colors shadow-md text-white"
        onclick={() => handleDeleteServerpool(selectedPool)}>
        Supprimer le serverpool
      </Button>
    </div>
  </div>
{:else}
  <div class="mt-12 bg-tertiary-300 rounded-2xl p-16 text-center border border-tertiary-200 shadow-lg flex flex-col items-center justify-center h-96">
    <div class="text-6xl mb-6 opacity-50">🎛️</div>
    <h3 class="text-2xl font-bold text-white mb-3">Aucun serverpool sélectionné</h3>
    <p class="text-gray-400 text-lg max-w-md">Sélectionnez un serverpool existant via le menu déroulant ci-dessus, ou créez-en un nouveau pour commencer le déploiement de vos instances.</p>
  </div>
{/if}

<!-- Modals -->
{#if createspModal}
<CreateServerPoolModal
  bind:open={createspModal}
  images={$images}
  flavors={sortedFlavors}
  networks={$networks}
  configs={$configs}

  bind:selectedGroupImage
  bind:selectedImage
  bind:selectedFlavor
  bind:selectedNetwork
  bind:selectedConfigFile
  bind:scheduleDay
  bind:scheduleTime
  bind:scheduleWindowHours
  bind:offDays

  {createError}
  {createSuccess}

  {handleCreateServerpool}
  {getUniqueFirstAlphaBlocks}
  {filterImagesByPrefix}
/>
{/if}

{#if ListStudentModalOpen && selectedPool}
  <AddSSHKeys
  bind:open={ListStudentModalOpen}
  bind:poolname={selectedPool.name}
  />
{/if}