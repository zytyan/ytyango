<script lang="ts">
	import dayjs from 'dayjs';
	import Avatar from './Avatar.svelte';
	import type { MeiliMsg } from './api';
	let prop = $props();
	let message: MeiliMsg = prop.message;
	let auth = window?.Telegram.WebApp.initData || 'noInitData';
	let time = dayjs((message?.date ?? 0) * 1000).format('YYYY-MM-DD HH:mm:ss');
	// TODO: 使用API获取人名
</script>

<div class="message-card">
	<div class="message-header">
		<div class="header-left">
			<Avatar
				src="/users/{message.from_id}/avatar?tgauth={auth}"
				name="U{message.from_id}"
				fallbackKey={message.from_id}
			/>
			<div>
				<div class="sender-name">U{message.from_id}</div>
				<div class="send-time">{time}</div>
			</div>
		</div>

		<div style="display:flex; gap:10px; align-items:center;">
			<div class="more-button">⋮</div>
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

	/* 顶部行：头像 + 名字 + 时间 + 星标 + 三点 */
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

	/* 三点菜单 */
	.more-button {
		color: var(--hint-color);
		font-size: 20px;
		cursor: pointer;
		padding: 4px;
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
