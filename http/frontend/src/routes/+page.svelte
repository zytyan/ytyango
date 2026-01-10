<script lang="ts">
	import MessageCard from '$lib/MessageCard.svelte';
	import SearchBar from '$lib/SearchBar.svelte';
	import { type MeiliMsg } from '$lib/api';
	import { search, users } from '$lib/api/search';

	const limit = 20;

	let query = $state('');
	let page = $state(1);
	let hits: Array<MeiliMsg> = $state([]);
	let hasMore = $state(false);
	let isLoadingInitial = $state(false);
	let isLoadingMore = $state(false);
	let error = $state<string | null>(null);
	let searchSession = $state(0);
	let lastRequestedPage = $state<number | null>(null);
	let sentinelEl: HTMLDivElement | null = $state(null);
	let observer: IntersectionObserver | null = null;

	async function onSearch(q: string) {
		query = q.trim();
		searchSession += 1;
		lastRequestedPage = null;
		page = 1;
		hits = [];
		hasMore = false;
		error = null;

		if (!query) {
			return;
		}

		await runSearch(1, false, searchSession);
	}

	async function loadMore() {
		if (!query || !hasMore || isLoadingInitial || isLoadingMore || error) {
			return;
		}
		await runSearch(page + 1, true, searchSession);
	}

	async function runSearch(targetPage: number, append: boolean, sessionId: number) {
		lastRequestedPage = targetPage;
		const isFirstPage = targetPage === 1 && !append;
		if (isFirstPage) {
			isLoadingInitial = true;
		} else {
			isLoadingMore = true;
		}

		try {
			const ins_id = window?.Telegram?.WebApp?.initDataUnsafe?.chat_instance;
			if (!ins_id) {
				throw new Error('缺少 Telegram init data');
			}
			const res = await search({
				q: query,
				ins_id,
				page: targetPage,
				limit
			});

			if (sessionId !== searchSession) {
				return;
			}

			const incomingHits = res.hits ?? [];
			hits = append ? [...hits, ...incomingHits] : incomingHits;
			page = targetPage;

			const nextOffset = res.offset + incomingHits.length;
			const total = res.estimatedTotalHits ?? hits.length;
			hasMore = incomingHits.length > 0 && nextOffset < total;
			error = null;
		} catch (err) {
			if (sessionId !== searchSession) {
				return;
			}
			error = err instanceof Error ? err.message : '加载失败，请重试';
			if (!append) {
				hasMore = false;
			}
		} finally {
			if (sessionId === searchSession) {
				isLoadingInitial = false;
				isLoadingMore = false;
			}
		}
	}

	function retryLast() {
		if (!query || lastRequestedPage === null) {
			return;
		}
		const append = lastRequestedPage > 1;
		error = null;
		void runSearch(lastRequestedPage, append, searchSession);
	}

	$effect(() => {
		if (!sentinelEl) {
			return;
		}
		observer?.disconnect();
		observer = new IntersectionObserver(
			(entries) => {
				if (entries.some((entry) => entry.isIntersecting)) {
					void loadMore();
				}
			},
			{ threshold: 0.2 }
		);
		observer.observe(sentinelEl);
		return () => observer?.disconnect();
	});
</script>

<SearchBar {onSearch} />

{#if isLoadingInitial}
	<div class="loader initial">
		<div class="spinner" ></div>
		<span>正在搜索...</span>
	</div>
{/if}

{#if hits.length === 0 && !isLoadingInitial && query}
	<div class="end-text">没有更多了</div>
{/if}

<div class="results">
	{#each hits as result (result.mongo_id)}
		<MessageCard message={result} user={users.get(result?.from_id ?? 0)} />
	{/each}
</div>

<div class="list-footer">
	{#if error}
		<div class="error-row">
			<span>{error}</span>
			<button class="retry" on:click={retryLast}>重试</button>
		</div>
	{/if}

	{#if isLoadingMore}
		<div class="loader">
			<div class="spinner" />
			<span>加载中...</span>
		</div>
	{/if}

	{#if (!hasMore || hits.length === 0) && !isLoadingInitial && query && !error}
		<div class="end-text">没有更多了</div>
	{/if}
</div>

<div class="sentinel" bind:this={sentinelEl}></div>

<style>
	.results {
		display: flex;
		flex-direction: column;
		gap: 12px;
		margin-top: 12px;
	}

	.loader {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		color: var(--hint-color);
		padding: 8px 0;
		justify-content: center;
	}

	.loader.initial {
		margin: 16px 0;
	}

	.spinner {
		width: 16px;
		height: 16px;
		border-radius: 50%;
		border: 2px solid color-mix(in srgb, var(--hint-color) 40%, transparent);
		border-top-color: var(--link-color);
		animation: spin 0.9s linear infinite;
	}

	.list-footer {
		padding: 8px 0 16px;
		display: flex;
		flex-direction: column;
		gap: 8px;
		align-items: center;
	}

	.end-text {
		color: var(--hint-color);
		font-size: 14px;
		text-align: center;
	}

	.error-row {
		display: flex;
		align-items: center;
		gap: 10px;
		color: var(--destructive-text-color);
		background: color-mix(in srgb, var(--destructive-text-color) 10%, transparent);
		padding: 8px 12px;
		border-radius: 8px;
	}

	.retry {
		padding: 6px 10px;
		font-size: 14px;
	}

	.sentinel {
		height: 1px;
		width: 100%;
	}

	@keyframes spin {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}
</style>
