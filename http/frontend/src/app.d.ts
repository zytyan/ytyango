// See https://svelte.dev/docs/kit/types#app.d.ts

import type { Telegram } from 'telegram-web-app';

// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
	interface Window {
		Telegram: Telegram;
	}
}

export {};
