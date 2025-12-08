<script lang="ts">

    let {onSearch} = $props();
    const placeholder: string = 'Search...';
    let value: string = $state('');
    function search(){
        onSearch(value);
    }
    function handleKeyPress(event: KeyboardEvent) {
        if (event.key === 'Enter') {
            search();
        }
    }
</script>

<div class="search-bar">
    <input type="text" {placeholder} bind:value onkeypress={handleKeyPress} />
    <button onclick={search}>Search</button>
</div>

<style>
    .search-bar {
        display: flex;
        gap: 8px;
        padding: 10px;
        border: 1px solid #eee;
        border-radius: 8px;
        background-color: #f9f9f9;
    }

    .search-bar input {
        flex-grow: 1;
        padding: 8px 12px;
        border: 1px solid #ccc;
        border-radius: 4px;
        font-size: 16px;
    }

    .search-bar input:focus {
        outline: none;
        border-color: #007bff;
        box-shadow: 0 0 0 2px rgba(0, 123, 255, 0.25);
    }

    .search-bar button {
        padding: 8px 16px;
        background-color: #007bff;
        color: white;
        border: none;
        border-radius: 4px;
        cursor: pointer;
        font-size: 16px;
        transition: background-color 0.2s ease;
    }

    .search-bar button:hover {
        background-color: #0056b3;
    }
</style>
