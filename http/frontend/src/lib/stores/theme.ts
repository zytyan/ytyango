import { writable } from 'svelte/store';
import type { ThemeMode } from '$lib/theme';

export const themeStore = writable<ThemeMode>('dark');

export const setTheme = (mode: ThemeMode) => {
	themeStore.set(mode);
};
