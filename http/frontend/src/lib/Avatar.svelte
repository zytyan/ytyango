<script lang="ts">
	let { src, name, fallbackKey } = $props();
	const avatarColors = [
		{ bg: '#3B82F6', fg: '#FFFFFF' }, // blue
		{ bg: '#2563EB', fg: '#FFFFFF' },
		{ bg: '#1D4ED8', fg: '#FFFFFF' },

		{ bg: '#8B5CF6', fg: '#FFFFFF' }, // purple
		{ bg: '#7C3AED', fg: '#FFFFFF' },

		{ bg: '#06B6D4', fg: '#FFFFFF' }, // cyan
		{ bg: '#0EA5E9', fg: '#FFFFFF' },

		{ bg: '#22C55E', fg: '#FFFFFF' }, // green
		{ bg: '#16A34A', fg: '#FFFFFF' },

		{ bg: '#F59E0B', fg: '#FFFFFF' }, // amber
		{ bg: '#D97706', fg: '#FFFFFF' },

		{ bg: '#EF4444', fg: '#FFFFFF' }, // red
		{ bg: '#DC2626', fg: '#FFFFFF' },

		{ bg: '#475569', fg: '#FFFFFF' }, // slate dark
		{ bg: '#334155', fg: '#FFFFFF' },

		{ bg: '#E5E7EB', fg: '#1F2937' } // light gray + dark text
	];

	function simpleHash(str: string): number {
		str = str.toString();
		let hash = 0;
		for (let i = 0; i < str.length; i++) {
			const char = str.charCodeAt(i);
			hash = (hash << 5) - hash + char;
			hash = hash & hash; // Convert to 32bit integer
		}
		return Math.abs(hash);
	}
	let { bg, fg } = $derived(avatarColors[simpleHash(fallbackKey) % avatarColors.length]);
	const segmenter = new Intl.Segmenter('en', { granularity: 'grapheme' });

	function firstGrapheme(str: string): string {
		const it = segmenter.segment(str)[Symbol.iterator]().next();
		let res = it.value?.segment ?? '';
		if (/^a-z/.test(res)) {
			return res.toUpperCase();
		}
		return res;
	}

	let initial = $derived(firstGrapheme(name));
	let showFallback = $state(true);
	let size = 48;
	function onLoad() {
		showFallback = false;
	}
	function onError() {
		showFallback = true;
	}
</script>

<div class="avatar-wrapper" style="width:{size}px; height:{size}px;">
	{#if src}
		<img
			{src}
			alt={name}
			onload={onLoad}
			onerror={onError}
			style="display: {showFallback ? 'none' : 'block'};"
		/>
	{/if}

	<div
		class="fallback"
		style="
      background:{bg};
      color:{fg};
      font-size:{size * 0.42}px;
      display:{showFallback ? 'flex' : 'none'};
    "
	>
		{initial}
	</div>
</div>

<style>
	.avatar-wrapper {
		position: relative;
		border-radius: 50%;
		overflow: hidden;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		font-weight: 600;
	}

	img {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.fallback {
		position: absolute;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
	}
</style>
