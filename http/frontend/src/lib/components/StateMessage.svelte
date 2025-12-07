<script lang="ts">
	export let title = '';
	export let description = '';
</script>

<div class="state glass">
	<div class="dot" aria-hidden="true"></div>
	<div class="text">
		<div class="title">{title}</div>
		{#if description}
			<p>{description}</p>
		{/if}
	</div>
</div>

<style>
	.state {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 14px 16px;
		background: var(--card);
		border: 1px solid var(--surface-border);
		border-radius: var(--radius-lg);
		box-shadow: var(--shadow);
	}

	.dot {
		width: 12px;
		height: 12px;
		border-radius: 50%;
		background: var(--accent);
		box-shadow: 0 0 0 6px rgba(255, 255, 255, 0.05);
	}

	.title {
		font-weight: 700;
		color: var(--text);
	}

	p {
		margin: 2px 0 0;
		color: var(--muted);
		font-size: 14px;
	}
</style>
