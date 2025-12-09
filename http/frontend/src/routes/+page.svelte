<script lang="ts">
	import Avatar from '$lib/Avatar.svelte';
	import MessageCard from '$lib/MessageCard.svelte';
	import SearchBar from '$lib/SearchBar.svelte';
	import { type MeiliMsg, type SearchResult, type SearchRequest } from '$lib/api';

	let query = $state('');
	let page = $state(1);
	let searchResult = $state<SearchResult | null>(null);
	let results: Array<MeiliMsg> = $derived(searchResult?.hits ?? []);
	async function search(req: SearchRequest) {
		let resp = await fetch('https://tgapi.zchan.moe/api/v1/tg/search', {
			method: 'POST',
			body: JSON.stringify(req),
			headers: {
				// TOOD: 这里要换成自定义头来验证，但现在先将就一下
				Authorization: `Telegram ${window?.Telegram.WebApp.initData || 'noInitData'}`,
				'Content-Type': 'application/json; charset=utf8'
			}
		});
		return await resp.json();
	}
	async function onSearch(q: string) {
		query = q;
		searchResult = await search({
			q,
			ins_id: '8485712724326358069',
			page,
			limit: 20
		});
	}
</script>

<SearchBar {onSearch} />
{#each results as result (result.mongo_id)}
	<MessageCard message={result} />
{/each}
