<script lang="ts">
	import { onMount } from 'svelte';
	import SearchBar from '$lib/components/SearchBar.svelte';
	import MessageCard from '$lib/components/MessageCard.svelte';
	import SkeletonCard from '$lib/components/SkeletonCard.svelte';
	import StateMessage from '$lib/components/StateMessage.svelte';
	import {
		SearchService,
		UsersService,
		type MeiliMsg,
		type UserInfo
	} from '$lib/api';
	import { avatarUrlForUser, shouldUseMock, withRetry } from '$lib/api/client';
	import { ensureMillis } from '$lib/utils/dates';
	import { loadStarred, persistStarred } from '$lib/utils/storage';
	import { sampleHits, sampleUsers } from '$lib/mocks/search';
	import { browser } from '$app/environment';

	type ViewMessage = {
		id: string;
		userId: number;
		displayName: string;
		username?: string | null;
		text: string;
		timestamp: number;
		avatarUrl?: string | null;
		starred: boolean;
	};

	const DEFAULT_INS_ID = '0';
	const PAGE_SIZE = 10;

	let query = '测试';
	let isLoading = false;
	let error = '';
	let hasMore = false;
	let page = 1;
	let messages: ViewMessage[] = [];
	let starred = loadStarred();
	const useMock = shouldUseMock();

	const mapToView = (hit: MeiliMsg, users: Map<number, UserInfo>): ViewMessage => {
		const userId = Number(hit.from_id || 0);
		const user = users.get(userId);
		const text = hit.message || hit.image_text || hit.qr_result || '暂无正文';

		return {
			id: hit.mongo_id || `${hit.msg_id}-${hit.peer_id}-${hit.from_id}`,
			userId,
			displayName: user?.name || `用户 ${userId || '未知'}`,
			username: user?.username,
			text,
			timestamp: ensureMillis(hit.date),
			avatarUrl: userId ? avatarUrlForUser(userId) : null,
			starred: starred.has(hit.mongo_id || `${hit.msg_id}-${hit.peer_id}-${hit.from_id}`)
		};
	};

	const fetchUsers = async (userIds: number[]): Promise<Map<number, UserInfo>> => {
		const idList = userIds.filter((id) => id > 0);
		if (!idList.length) return new Map();

		const data = await withRetry(() =>
			UsersService.getUsersInfo({
				requestBody: {
					user_ids: idList
				}
			})
		);

		const map = new Map<number, UserInfo>();
		data.users?.forEach((user) => map.set(Number(user.id), user));
		return map;
	};

	const runSearch = async (reset = false) => {
		if (!query.trim()) {
			error = '请输入关键字';
			messages = [];
			return;
		}

		if (useMock) {
			messages = sampleHits.map((hit) =>
				mapToView(
					hit,
					new Map(
						Object.entries(sampleUsers).map(([id, value]) => [Number(id), { id: Number(id), ...value }])
					)
				)
			);
			hasMore = false;
			error = '';
			return;
		}

		isLoading = true;
		error = '';

		const targetPage = reset ? 1 : page;

		try {
			const result = await withRetry(() =>
				SearchService.searchMessages({
					requestBody: {
						q: query.trim(),
						ins_id: DEFAULT_INS_ID,
						page: targetPage,
						limit: PAGE_SIZE
					}
				})
			);

			const hits = result.hits || [];
			const userIds = Array.from(new Set(hits.map((hit) => Number(hit.from_id || 0)).filter(Boolean)));
			const users = await fetchUsers(userIds);
			const mapped = hits.map((hit) => mapToView(hit, users));

			const total = result.estimatedTotalHits ?? mapped.length;
			const offset = result.offset ?? (targetPage - 1) * (result.limit ?? PAGE_SIZE);

			messages = reset ? mapped : [...messages, ...mapped];
			hasMore = offset + mapped.length < total;
			page = targetPage + 1;
		} catch (err) {
			console.error(err);
			error = err instanceof Error ? err.message : '请求失败，请稍后重试';
			hasMore = false;
		} finally {
			isLoading = false;
		}
	};

	const toggleStar = (id: string) => {
		if (!browser) return;
		if (starred.has(id)) {
			starred.delete(id);
		} else {
			starred.add(id);
		}
		persistStarred(starred);
		messages = messages.map((msg) => (msg.id === id ? { ...msg, starred: starred.has(id) } : msg));
	};

	onMount(() => {
		runSearch(true);
	});
</script>

<svelte:head>
	<title>消息搜索</title>
	<meta name="description" content="SvelteKit + Telegram WebApp 风格的搜索界面" />
</svelte:head>

<main class="stack">
	<section class="header">
		<h1>搜索</h1>
		<p>输入关键字，查看最近的消息结果。界面将匹配 Telegram 主题。</p>
	</section>

	<SearchBar
		bind:value={query}
		loading={isLoading}
		placeholder="搜索"
		on:submit={() => runSearch(true)}
		on:clear={() => {
			messages = [];
			error = '';
			hasMore = false;
		}}
	/>

	<section class="result-stack">
		{#if error}
			<StateMessage title="请求异常" description={error} />
		{:else if !isLoading && !messages.length}
			<StateMessage title="暂无内容" description="尝试输入其他关键词或调整时间范围" />
		{/if}

		{#if isLoading}
			<div class="list">
				{#each Array(3) as _}
					<SkeletonCard />
				{/each}
			</div>
		{:else}
			<div class="list">
				{#each messages as message (message.id)}
					<MessageCard message={message} on:toggleStar={(event) => toggleStar(event.detail)} />
				{/each}
			</div>
		{/if}

		{#if hasMore && !isLoading}
			<button class="more" type="button" on:click={() => runSearch(false)}>加载更多</button>
		{/if}
	</section>
</main>

<style>
	.stack {
		display: grid;
		gap: 18px;
	}

	.header h1 {
		margin: 0 0 6px;
		font-size: 26px;
	}

	.header p {
		margin: 0;
		color: var(--muted);
	}

	.result-stack {
		display: grid;
		gap: 12px;
	}

	.list {
		display: grid;
		gap: 12px;
	}

	.more {
		width: 100%;
		padding: 12px 14px;
		border-radius: var(--radius-md);
		border: 1px solid var(--surface-border);
		background: var(--card);
		color: var(--text);
		cursor: pointer;
		font-weight: 600;
		transition: transform 0.1s ease, box-shadow 0.1s ease;
	}

	.more:hover {
		box-shadow: var(--shadow);
		transform: translateY(-1px);
	}

	@media (max-width: 640px) {
		.stack {
			gap: 14px;
		}

		.header h1 {
			font-size: 22px;
		}
	}
</style>
