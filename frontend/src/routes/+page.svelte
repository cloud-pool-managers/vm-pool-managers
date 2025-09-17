<script lang="ts">
	import { user } from '$lib/stores/user';

	let features = [
		{
			title: "Gestion centralisée",
			description: "Administrez toutes vos piscines en un seul endroit."
		},
		{
			title: "Suivi en temps réel",
			description: "Surveillez les données et performances de vos équipements."
		},
		{
			title: "Alertes intelligentes",
			description: "Recevez des notifications automatiques en cas de problème."
		}
	];
</script>

<section class="text-center py-20 px-6 bg-gradient-to-r from-blue-500 to-indigo-600 text-white">
	{#if $user}
		<h2 class="text-4xl font-bold mb-4">Bienvenue, {$user.name} !</h2>
		<p class="text-lg mb-6">Vous êtes connecté et pouvez accéder à votre dashboard.</p>
		<a href="/dashboard" class="bg-white text-blue-600 px-6 py-3 rounded-lg shadow hover:bg-gray-100">
			Accéder au dashboard
		</a>
	{:else}
		<h2 class="text-4xl font-bold mb-4">Bienvenue sur CloudPoolManager</h2>
		<p class="text-lg mb-6">La solution cloud pour gérer vos piscines, équipements et alertes.</p>
		<a href="/signup" class="bg-white text-blue-600 px-6 py-3 rounded-lg shadow hover:bg-gray-100">
			Créer un compte
		</a>
	{/if}
</section>

{#if !$user}
	<section class="py-16 px-6 max-w-5xl mx-auto">
		<h3 class="text-2xl font-bold text-center mb-12">Fonctionnalités principales</h3>
		<div class="grid grid-cols-1 md:grid-cols-3 gap-8">
			{#each features as f}
				<div class="bg-white p-6 rounded-xl shadow hover:shadow-lg transition">
					<h4 class="text-xl font-semibold mb-2 text-blue-600">{f.title}</h4>
					<p class="text-gray-600">{f.description}</p>
				</div>
			{/each}
		</div>
	</section>
{/if}
