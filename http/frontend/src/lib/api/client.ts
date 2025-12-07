import { browser } from '$app/environment';
import { OpenAPI } from './index';

const fallbackBase = 'http://localhost:4173';
const baseUrl = import.meta.env.VITE_API_BASE_URL || '';
const generatedBase = OpenAPI.BASE;
let cachedAuth = import.meta.env.VITE_TG_AUTH || '';

export const resolveTelegramAuth = (): string => {
	if (browser) {
		const tgAuth = window.Telegram?.WebApp?.initData;
		if (tgAuth) {
			cachedAuth = tgAuth;
		}
	}
	return cachedAuth;
};

export const configureOpenApi = (): void => {
	OpenAPI.BASE = baseUrl || generatedBase || '';
	OpenAPI.HEADERS = async () => {
		const auth = resolveTelegramAuth();
		const headers: Record<string, string> = {};
		if (auth) {
			headers['X-Telegram-Init-Data'] = auth;
		}
		return headers;
	};
};

configureOpenApi();

export const withRetry = async <T>(fn: () => Promise<T>, retries = 1): Promise<T> => {
	let lastError: unknown;
	for (let attempt = 0; attempt <= retries; attempt += 1) {
		try {
			return await fn();
		} catch (error) {
			lastError = error;
		}
	}
	throw lastError;
};

export const avatarUrlForUser = (userId: number | string): string => {
	const auth = resolveTelegramAuth();
	const base = baseUrl || generatedBase || (browser ? window.location.origin : fallbackBase);
	const url = new URL(`/users/${userId}/avatar`, base);
	if (auth) {
		url.searchParams.set('tgauth', auth);
	}
	return url.toString();
};

export const shouldUseMock = (): boolean =>
	(import.meta.env.VITE_USE_MOCK || '').toString().toLowerCase() === 'true';
