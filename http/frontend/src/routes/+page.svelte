<script lang="ts">
	import Avatar from "$lib/Avatar.svelte";
	import SearchBar from "$lib/SearchBar.svelte";
	let queries :Array<string>= $state([]);
	function onSearch(query:string){
		if (query === ''){
			queries.push("<empty query>");
			return;
		}
		queries.push(query);
	}
</script>

<SearchBar {onSearch} />
<Avatar src="https://avatar.iran.liara.run/public/9" name="Test" fallbackKey="fjieo" />
<Avatar src="none" name="Test" fallbackKey="fjieo" />

{#each queries as query}
	<p>{query}</p>
{/each}
