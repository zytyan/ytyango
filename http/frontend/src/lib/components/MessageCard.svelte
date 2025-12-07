<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import AvatarCircle from '$lib/components/AvatarCircle.svelte';
	import { formatMessageTime } from '$lib/utils/dates';

	type RenderedMessage = {
		id: string;
		userId: number;
		displayName: string;
		username?: string | null;
		text: string;
		timestamp: number;
		avatarUrl?: string | null;
		starred?: boolean;
	};

	export let message: RenderedMessage;
	export let showMenu = true;

	const dispatch = createEventDispatcher<{ toggleStar: string }>();

	const toggleStar = () => {
		dispatch('toggleStar', message.id);
	};

	const datetime = formatMessageTime(message.timestamp);
	const initials = message.displayName?.trim()?.[0]?.toUpperCase() ?? '?';
	const avatarAlt = `${initials} 的头像`;
	let menuOpen = false;
	const toggleMenu = () => {
		menuOpen = !menuOpen;
	};
</script>

<article class="card glass">
	<div class="left">
		<AvatarCircle
			userId={message.userId}
			name={message.displayName}
			avatarUrl={message.avatarUrl}
			ariaLabel={avatarAlt}
		/>
	</div>
	<div class="body">
		<header class="top">
			<div class="meta">
				<div class="name">{message.displayName}</div>
				{#if message.username}
					<div class="username">@{message.username}</div>
				{/if}
				<div class="dot" aria-hidden="true"></div>
				<time datetime={new Date(message.timestamp).toISOString()}>{datetime}</time>
			</div>
			<div class="actions">
				<button
					class:active={message.starred}
					type="button"
					aria-label={message.starred ? '取消星标' : '添加星标'}
					on:click={toggleStar}
				>
					<svg viewBox="0 0 24 24" stroke="currentColor" fill={message.starred ? 'currentColor' : 'none'}>
						<path
							d="M12 3.6 9.7 9H4.6a.6.6 0 0 0-.36 1.08L8.7 12.9l-2.1 5.4a.6.6 0 0 0 .9.72L12 15.8l4.5 3.22a.6.6 0 0 0 .9-.72l-2.1-5.4 4.46-2.82A.6.6 0 0 0 19.4 9h-5.1z"
						/>
					</svg>
				</button>
				<button class="ghost" type="button" aria-label="更多操作" on:click={toggleMenu}>
					<svg viewBox="0 0 24 24" fill="currentColor">
						<circle cx="5" cy="12" r="1.6" />
						<circle cx="12" cy="12" r="1.6" />
						<circle cx="19" cy="12" r="1.6" />
					</svg>
				</button>
				{#if showMenu && menuOpen}
					<div class="menu glass" role="menu">
						<button type="button" role="menuitem">复制</button>
						<button type="button" role="menuitem">分享</button>
					</div>
				{/if}
			</div>
		</header>
		<p class="text">{message.text}</p>
	</div>
</article>

<style>
	.card {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 12px;
		padding: 16px;
		background: var(--card);
		border-radius: var(--radius-lg);
		border: 1px solid var(--surface-border);
		box-shadow: var(--shadow);
		position: relative;
		overflow: hidden;
	}

	.left {
		display: grid;
		align-items: flex-start;
	}

	.body {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.top {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 10px;
	}

	.meta {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
	}

	.name {
		font-weight: 700;
	}

	.username {
		color: var(--muted);
		font-size: 14px;
	}

	time {
		color: var(--muted);
		font-size: 13px;
	}

	.dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--muted);
		opacity: 0.5;
	}

	.actions {
		display: flex;
		align-items: center;
		gap: 6px;
		position: relative;
	}

	button {
		border: none;
		background: none;
		color: var(--muted);
		cursor: pointer;
		width: 36px;
		height: 36px;
		border-radius: var(--radius-sm);
		display: grid;
		place-items: center;
		transition: background 0.15s ease, color 0.15s ease;
	}

	button:hover {
		background: rgba(255, 255, 255, 0.05);
		color: var(--text);
	}

	button.active {
		color: var(--accent-strong);
	}

	svg {
		width: 18px;
		height: 18px;
	}

	.text {
		margin: 0;
		line-height: 1.55;
		color: var(--text);
	}

	.menu {
		position: absolute;
		top: 42px;
		right: 0;
		background: var(--card);
		border: 1px solid var(--surface-border);
		border-radius: var(--radius-md);
		padding: 6px;
		display: grid;
		gap: 4px;
		min-width: 140px;
		box-shadow: var(--shadow);
	}

	.menu button {
		width: 100%;
		justify-content: flex-start;
		padding: 8px 10px;
		color: var(--text);
	}

	.menu button:hover {
		background: rgba(255, 255, 255, 0.06);
	}
</style>
