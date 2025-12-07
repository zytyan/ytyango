import { browser } from '$app/environment';

export type ThemeMode = 'light' | 'dark';

const darkPalette = [
	'#ffe066',
	'#ff8fa3',
	'#64d2ff',
	'#6ee7b7',
	'#f7b801',
	'#c792ea',
	'#4cc9f0',
	'#ff9770'
];

const lightPalette = [
	'#ffb703',
	'#ff477e',
	'#4cc9f0',
	'#6a4c93',
	'#118ab2',
	'#06d6a0',
	'#f07167',
	'#f9c74f'
];

const hashString = (value: string): number => {
	let hash = 0;
	for (let i = 0; i < value.length; i += 1) {
		hash = (hash << 5) - hash + value.charCodeAt(i);
		hash |= 0;
	}
	return hash;
};

export const avatarColor = (seed: string | number, mode: ThemeMode = 'dark'): string => {
	const palette = mode === 'light' ? lightPalette : darkPalette;
	const hashed = hashString(String(seed || '0'));
	const index = Math.abs(hashed) % palette.length;
	return palette[index];
};

export const detectTheme = (): ThemeMode => {
	if (!browser) return 'dark';
	const tgColor = window.Telegram?.WebApp?.colorScheme;
	if (tgColor === 'light' || tgColor === 'dark') return tgColor;
	if (window.matchMedia?.('(prefers-color-scheme: light)').matches) return 'light';
	return 'dark';
};

type ThemeHandler = (mode: ThemeMode) => void;

export const subscribeTelegramTheme = (onChange: ThemeHandler): (() => void) | undefined => {
	if (!browser) return undefined;
	const tg = window.Telegram?.WebApp;
	if (!tg?.onEvent) return undefined;

	const handler = () => {
		const next = tg.colorScheme === 'light' ? 'light' : 'dark';
		onChange(next);
	};

	tg.onEvent('themeChanged', handler);

	return () => {
		tg.offEvent?.('themeChanged', handler);
	};
};

export const applyDocumentTheme = (mode: ThemeMode): void => {
	if (!browser) return;
	document.documentElement.dataset.theme = mode;
	document.documentElement.style.setProperty('color-scheme', mode);
};
