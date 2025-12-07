<script lang="ts">
	import { createEventDispatcher } from 'svelte';

	const dispatch = createEventDispatcher<{
		submit: string;
		input: string;
		clear: void;
	}>();

	export let value = '';
	export let placeholder = '搜索';
	export let loading = false;

	const handleSubmit = () => {
		dispatch('submit', value.trim());
	};

	const clear = () => {
		value = '';
		dispatch('clear');
		dispatch('input', value);
	};
</script>

<form class="search" on:submit|preventDefault={handleSubmit}>
	<div class="icon" aria-hidden="true">
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<circle cx="11" cy="11" r="7" />
			<line x1="16.65" y1="16.65" x2="21" y2="21" />
		</svg>
	</div>
	<input
		type="search"
		bind:value
		name="q"
		autocomplete="off"
		placeholder={placeholder}
		on:input={(event) => dispatch('input', (event.target as HTMLInputElement).value)}
	/>
	<div class="right">
		{#if value}
			<button class="ghost" type="button" aria-label="清除" on:click={clear}>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<line x1="18" y1="6" x2="6" y2="18" />
					<line x1="6" y1="6" x2="18" y2="18" />
				</svg>
			</button>
		{/if}
		<button class="submit" type="submit" aria-label="搜索" disabled={loading}>
			{#if loading}
				<span class="spinner" aria-hidden="true"></span>
			{:else}
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="11" cy="11" r="7" />
					<line x1="16.65" y1="16.65" x2="21" y2="21" />
				</svg>
			{/if}
		</button>
	</div>
</form>

<style>
	.search {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 12px 14px;
		background: var(--card);
		border-radius: var(--radius-lg);
		box-shadow: var(--shadow);
		border: 1px solid var(--surface-border);
	}

	.icon {
		width: 20px;
		height: 20px;
		color: var(--muted);
	}

	input {
		flex: 1;
		background: transparent;
		border: none;
		outline: none;
		color: var(--text);
		font-size: 16px;
	}

	input::placeholder {
		color: var(--muted);
	}

	.right {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	button {
		border: none;
		cursor: pointer;
		color: var(--text);
		background: none;
		display: grid;
		place-items: center;
		border-radius: var(--radius-sm);
	}

	button:hover {
		background: rgba(255, 255, 255, 0.05);
	}

	.ghost {
		width: 28px;
		height: 28px;
		color: var(--muted);
	}

	.submit {
		width: 40px;
		height: 40px;
		background: linear-gradient(135deg, var(--accent), var(--accent-strong));
		color: #0d1017;
	}

	.submit:disabled {
		opacity: 0.7;
		cursor: wait;
	}

	svg {
		width: 20px;
		height: 20px;
	}

	.spinner {
		width: 16px;
		height: 16px;
		border-radius: 50%;
		border: 2px solid rgba(0, 0, 0, 0.2);
		border-top-color: rgba(0, 0, 0, 0.7);
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}
</style>
