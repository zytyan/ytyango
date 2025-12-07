<svelte:options runes={false} />

<script lang="ts">
	import '@fontsource-variable/manrope';
	import '$lib/styles/global.css';
	import { onMount, onDestroy } from 'svelte';
	import {
		applyDocumentTheme,
		detectTheme,
		subscribeTelegramTheme,
		type ThemeMode
	} from '$lib/theme';
	import { setTheme, themeStore } from '$lib/stores/theme';
	import favicon from '$lib/assets/favicon.svg';

	let theme: ThemeMode = 'dark';
	let unsub: (() => void) | undefined;

	onMount(() => {
		const initial = detectTheme();
		theme = initial;
		applyDocumentTheme(initial);
		setTheme(initial);

		if (window.Telegram?.WebApp) {
			window.Telegram.WebApp.ready();
			window.Telegram.WebApp.expand?.();
		}

		unsub = subscribeTelegramTheme((next) => {
			theme = next;
			applyDocumentTheme(next);
			setTheme(next);
		});
	});

	onDestroy(() => {
		unsub?.();
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<meta name="theme-color" content={theme === 'light' ? '#f6f8fb' : '#0f1117'} />
</svelte:head>

<div class="app-shell" data-theme={$themeStore}>
	<slot />
</div>
