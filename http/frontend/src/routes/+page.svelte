<script lang="ts">
	import Avatar from "$lib/Avatar.svelte";
    import SearchBar from "$lib/SearchBar.svelte";
    function onSearch(query: string){
        console.log(query);
    }
</script>

<SearchBar {onSearch}></SearchBar>
<Avatar src="https://i.pravatar.cc/300" name="Zest" fallbackKey="f12w23" />
