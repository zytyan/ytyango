<script lang="ts">
	import dayjs from 'dayjs';
	import Avatar from './Avatar.svelte';
	import type { MeiliMsg } from './api';
	import { baseUrl, type Deferred } from './api/search';
	type Props = {
		message: MeiliMsg;
		user: Deferred<string> | undefined;
	};
	let { message, user }: Props = $props();
	let name = $state('');
	$effect(() => {
		if (name == '') {
			name = `U${message?.from_id ?? 0}`;
		}
		user?.promise.then((s) => {
			name = s;
		});
	});

	let auth = encodeURIComponent(window?.Telegram.WebApp.initData || 'noInitData');
	let time = $derived(dayjs((message?.date ?? 0) * 1000).format('YYYY-MM-DD HH:mm:ss'));
</script>

<div class="message-card">
	<div class="message-header">
		<div class="header-left">
			<Avatar
				src="{baseUrl}/users/{message.from_id}/avatar?tgauth={auth}"
				{name}
				fallbackKey={message.from_id}
			/>
			<div>
				<div class="sender-name">{name}</div>
				<div class="send-time">{time}</div>
			</div>
		</div>

		<div style="display:flex; gap:10px; align-items:center;">
			<symbol id="material-symbols--chat-paste-go-outline-rounded" viewBox="0 0 24 24"
				><path
					fill="currentColor"
					d="M18.175 18H15q-.425 0-.712-.288T14 17t.288-.712T15 16h3.175l-.875-.875q-.275-.3-.288-.712t.288-.713t.7-.3t.7.3l2.6 2.6q.3.3.3.7t-.3.7l-2.6 2.6q-.3.3-.7.3t-.7-.3t-.288-.712t.288-.713zM6 18l-2.15 2.15q-.25.25-.55.125T3 19.8V6q0-.825.588-1.412T5 4h12q.825 0 1.413.588T19 6v4q0 .425-.288.713T18 11t-.712-.288T17 10V6H5v10h6q.425 0 .713.288T12 17t-.288.713T11 18zm2-8h6q.425 0 .713-.288T15 9t-.288-.712T14 8H8q-.425 0-.712.288T7 9t.288.713T8 10m0 4h3q.425 0 .713-.288T12 13t-.288-.712T11 12H8q-.425 0-.712.288T7 13t.288.713T8 14m-3 2V6z"
				/></symbol
			>
		</div>
	</div>

	<div class="message-body">{message.message}</div>

	{#if message.image_text}
		<div class="message-ocr">{message.image_text}</div>
	{/if}
</div>

<style>
	.message-card {
		background: var(--secondary-bg-color);
		border-radius: 12px;
		padding: 14px 16px;
		box-shadow: 0 2px 6px rgba(0, 0, 0, 0.25);
		color: var(--text-color);
		font-size: 14px;
		margin: 12px 0;
		position: relative;
	}

	/* 顶部行：头像 + 名字 + 时间 + 三点 */
	.message-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 8px;
	}

	/* 左区：头像 + 名称+时间 */
	.header-left {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.sender-name {
		font-size: 15px;
		font-weight: 600;
		color: var(--text-color);
	}

	.send-time {
		font-size: 12px;
		color: var(--hint-color);
		margin-top: 2px;
	}
	/* 消息正文 */
	.message-body {
		margin: 6px 0;
		line-height: 1.45;
		font-size: 15px;
		color: var(--text-color);
	}

	/* OCR 文本（灰色次级） */
	.message-ocr {
		margin-top: 4px;
		font-size: 13px;
		color: var(--hint-color);
	}
</style>
