<script lang="ts">
	import { onDestroy } from 'svelte';
	import { avatarColor, type ThemeMode } from '$lib/theme';
	import { themeStore } from '$lib/stores/theme';

	export let userId: number | string;
	export let name = '';
	export let avatarUrl: string | null = null;
	export let ariaLabel = '头像';

	let showFallback = false;
	let theme: ThemeMode = 'dark';
	let initials = '?';

	const unsubscribe = themeStore.subscribe((value) => {
		theme = value;
	});

	const refreshInitials = () => {
		const trimmed = name?.trim();
		initials = trimmed ? trimmed[0]?.toUpperCase() ?? '?' : '?';
	};

	refreshInitials();

	$: refreshInitials();

	const onError = () => {
		showFallback = true;
	};

	onDestroy(() => {
		unsubscribe();
	});
</script>

<div class="avatar" style={`background:${avatarColor(userId, theme)}`} role="img" aria-label={ariaLabel}>
	{#if avatarUrl && !showFallback}
		<img src={avatarUrl} alt={name} loading="lazy" on:error={onError} />
	{:else}
		<span>{initials}</span>
	{/if}
</div>

<style>
	.avatar {
		width: 46px;
		height: 46px;
		border-radius: 50%;
		display: grid;
		place-items: center;
		font-weight: 700;
		color: #0b0d12;
		overflow: hidden;
		border: 2px solid rgba(255, 255, 255, 0.08);
		box-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
	}

	span {
		font-size: 18px;
		letter-spacing: 0.5px;
	}

	img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		display: block;
	}
</style>
