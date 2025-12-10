<script lang="ts">
	import MessageCard from '$lib/MessageCard.svelte';
	import SearchBar from '$lib/SearchBar.svelte';
	import { type MeiliMsg, type SearchResult } from '$lib/api';
	import { search, users } from '$lib/api/search';

	let query = $state('');
	let page = $state(1);
	let searchResult = $state<SearchResult | null>(null);
	let results: Array<MeiliMsg> = $derived(searchResult?.hits ?? []);
	async function onSearch(q: string) {
		query = q;
		let ins_id = window?.Telegram.WebApp.initDataUnsafe.chat_instance;
		if (!ins_id) {
			console.error('No ins_id');
			return;
		}
		searchResult = await search({
			q,
			ins_id,
			page,
			limit: 20
		});
	}
</script>

<SearchBar {onSearch} />

{#each results as result (result.mongo_id)}
	<MessageCard message={result} user={users.get(result?.from_id ?? 0)} />
{/each}
