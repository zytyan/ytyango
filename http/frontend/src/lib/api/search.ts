import type { SearchRequest } from './models/SearchRequest';
import type { SearchResult } from './models/SearchResult';
import type { UserInfoRequest } from './models/UserInfoRequest';
import type { UserInfoResponse } from './models/UserInfoResponse';

export interface Deferred<T> {
	promise: Promise<T>;
	resolve: (value: T | PromiseLike<T>) => void;
	reject: (reason?: any) => void;
}

function createDeferred<T>(): Deferred<T> {
	let resolve!: (value: T | PromiseLike<T>) => void;
	let reject!: (reason?: any) => void;

	const promise = new Promise<T>((res, rej) => {
		resolve = res;
		reject = rej;
	});

	return { promise, resolve, reject };
}
export const users = new Map<number, Deferred<string>>();
export const baseUrl = 'https://tgapi.zchan.moe';
export async function search(req: SearchRequest) {
	let headers = {
		'X-Telegram-Init-Data': `${window?.Telegram.WebApp.initData || 'noInitData'}`,
		'Content-Type': 'application/json; charset=utf8'
	};
	let resp = await fetch(`${baseUrl}/search`, {
		method: 'POST',
		body: JSON.stringify(req),
		headers
	});
	let result: SearchResult = await resp.json();
	let usersReq: UserInfoRequest = { user_ids: [] };
	for (let hit of result.hits) {
		if (hit.from_id && !users.has(hit.from_id)) {
			usersReq.user_ids.push(hit.from_id);
			users.set(hit.from_id, createDeferred());
		}
	}
	fetch(`${baseUrl}/users/info`, {
		method: 'POST',
		body: JSON.stringify(usersReq),
		headers
	}).then(async (body) => {
		let uResp: UserInfoResponse = await body.json();
		for (let i = 0; i < usersReq.user_ids.length; i++) {
			let r = uResp.users[i];
			if (r.name) {
				users.get(r.id)?.resolve(r.name);
			} else if (r.error) {
				users.get(r.id)?.reject(r.error);
			}
		}
	});
	return result;
}
