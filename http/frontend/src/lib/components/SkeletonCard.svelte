<div class="card glass">
	<div class="avatar shimmer"></div>
	<div class="body">
		<div class="line wide shimmer"></div>
		<div class="line shimmer"></div>
		<div class="line short shimmer"></div>
	</div>
</div>

<style>
	.card {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 12px;
		padding: 16px;
		background: var(--card);
		border-radius: var(--radius-lg);
		border: 1px solid var(--surface-border);
		overflow: hidden;
	}

	.avatar {
		width: 46px;
		height: 46px;
		border-radius: 50%;
		background: rgba(255, 255, 255, 0.06);
	}

	.body {
		display: grid;
		gap: 10px;
	}

	.line {
		height: 12px;
		border-radius: 999px;
		background: rgba(255, 255, 255, 0.08);
	}

	.line.wide {
		width: 60%;
	}

	.line.short {
		width: 40%;
	}

	.shimmer {
		background: linear-gradient(
			90deg,
			rgba(255, 255, 255, 0.06) 0%,
			rgba(255, 255, 255, 0.14) 50%,
			rgba(255, 255, 255, 0.06) 100%
		);
		background-size: 200% 100%;
		animation: shimmer 1.4s infinite;
	}

	@keyframes shimmer {
		0% {
			background-position: 200% 0;
		}
		100% {
			background-position: -200% 0;
		}
	}
</style>
