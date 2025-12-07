// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}

	interface TelegramWebApp {
		colorScheme?: 'light' | 'dark';
		initData?: string;
		ready: () => void;
		expand?: () => void;
		onEvent?: (_event: 'themeChanged', _callback: () => void) => void;
		offEvent?: (_event: 'themeChanged', _callback: () => void) => void;
	}

	interface TelegramNamespace {
		WebApp?: TelegramWebApp;
	}

	interface Window {
		Telegram?: TelegramNamespace;
	}
}

export {};
