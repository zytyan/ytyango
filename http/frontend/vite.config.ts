// @ts-nocheck
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';
import type { PluginOption } from 'vite';

const ensureEnvironments: PluginOption = {
	name: 'ensure-svelte-environments',
	apply: 'serve',
	configureServer(server) {
		if (!(server as any).environments) {
			(server as any).environments = {
				client: {
					config: { consumer: 'client' }
				}
			};
		}
	}
};

export default defineConfig({
	plugins: [ensureEnvironments, sveltekit() as PluginOption],
	test: {
		environment: 'jsdom',
		globals: true,
		setupFiles: ['src/test/setup.ts'],
		include: ['src/**/*.{test,spec}.{js,ts}'],
		deps: {
			inline: ['@testing-library/svelte', '@sveltejs/kit', 'svelte', '@sveltejs/vite-plugin-svelte']
		}
	}
});
