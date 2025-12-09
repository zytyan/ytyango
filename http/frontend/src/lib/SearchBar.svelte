<script lang="ts">
	let { onSearch } = $props();
	const placeholder: string = 'Search...';
	let value: string = $state('');
	function search() {
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
		border: 1px solid var(--section-separator-color);
		border-radius: 8px;
		background-color: var(--secondary-bg-color);
	}

	/* 输入框 */
	.search-bar input {
		flex-grow: 1;
		padding: 8px 12px;
		border: 1px solid var(--section-separator-color);
		border-radius: 4px;
		font-size: 16px;
		background: var(--bg-color);
		color: var(--text-color);
	}

	/* 输入框 focus 状态 */
	.search-bar input:focus {
		outline: none;
		border-color: var(--link-color);
		box-shadow: 0 0 0 2px color-mix(in srgb, var(--link-color) 30%, transparent);
	}

	/* 按钮 */
	.search-bar button {
		background-color: var(--button-color);
		color: var(--button-text-color);
		border: none;
		border-radius: 4px;
		padding: 8px 14px;
		cursor: pointer;
		font-size: 15px;
	}

	/* 按钮 hover（用更深的颜色但保持主题） */
	.search-bar button:hover {
		background-color: color-mix(in srgb, var(--button-color) 80%, black);
	}
</style>
