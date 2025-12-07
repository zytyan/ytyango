import js from '@eslint/js';
import tsParser from '@typescript-eslint/parser';
import globals from 'globals';
import svelte from 'eslint-plugin-svelte';
import svelteParser from 'svelte-eslint-parser';
import tseslint from 'typescript-eslint';

const tsRecommendedRules = tseslint.configs.recommended.reduce(
	(acc, config) => ({ ...acc, ...(config.rules || {}) }),
	{}
);

export default [
	{
		ignores: [
			'build',
			'dist',
			'.svelte-kit',
			'node_modules',
			'src/lib/api/**',
			'eslint.config.js',
			'svelte.config.js',
			'vite.config.ts'
		]
	},
	{
		files: ['**/*.svelte'],
		plugins: {
			svelte
		},
		languageOptions: {
			parser: svelteParser,
			globals: {
				...globals.browser,
				Telegram: 'readonly'
			},
			parserOptions: {
				extraFileExtensions: ['.svelte'],
				parser: tsParser
			}
		},
		rules: {
			...svelte.configs['flat/recommended'].rules,
			'no-console': ['warn', { allow: ['warn', 'error'] }]
		}
	},
	{
		files: ['**/*.{ts,js}'],
		plugins: {
			'@typescript-eslint': tseslint.plugin
		},
		languageOptions: {
			parser: tsParser,
			globals: {
				...globals.browser,
				...globals.node,
				Telegram: 'readonly'
			}
		},
		rules: {
			...js.configs.recommended.rules,
			...tsRecommendedRules,
			'no-console': ['warn', { allow: ['warn', 'error'] }]
		}
	}
];
