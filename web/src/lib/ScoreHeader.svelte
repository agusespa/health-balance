<script lang="ts">
	import type { Snippet } from 'svelte';
	import ReserveGauge from '#lib/ReserveGauge.svelte';
	import { signed } from '#lib/format.ts';

	/**
	 * The coloured score header of the home and week pages. It is fixed to the
	 * top and collapses as the page scrolls: the week label and gauge fold away
	 * and the score shrinks into the bar, so the score is always on screen.
	 * Everything is driven by one CSS variable, --p (0 = expanded, 1 =
	 * collapsed); a spacer of the expanded height keeps the page from jumping.
	 */
	let {
		eyebrow,
		title,
		score,
		loading = false,
		error = '',
		deltaLabel = 'this week',
		actions
	}: {
		eyebrow: string;
		title: string;
		score: { score: number; change: number; settle: number } | null;
		loading?: boolean;
		error?: string;
		deltaLabel?: string;
		actions?: Snippet;
	} = $props();

	let y = $state(0);
	let innerH = $state(0);
	let expandedH = $state(0);
	let row = $state<HTMLElement>();

	// The expanded height is measured while nothing is collapsed.
	$effect(() => {
		if (y <= 0 && innerH) expandedH = innerH;
	});

	// Collapsed, the header is just the bar row (plus a little padding).
	const collapsedH = $derived(row ? row.offsetTop + row.offsetHeight + 8 : 0);
	const distance = $derived(Math.max(1, expandedH - collapsedH));
	const p = $derived(Math.min(1, Math.max(0, y / distance)));
</script>

<svelte:window bind:scrollY={y} />

<div class="spacer" style:height={`${expandedH + 16}px`}></div>

<header class="score-header band" data-testid="score-card" style:--p={p}>
	<div class="inner" bind:offsetHeight={innerH}>
		<div class="row" bind:this={row}>
			<h1 class="week">
				{#if title}
					<span class="eyebrow">{eyebrow}</span>
					{title}
				{/if}
			</h1>
			<div class="actions">
				{@render actions?.()}
			</div>
		</div>

		{#if error}
			<p class="error">{error}</p>
		{:else if loading}
			<div class="figure"><p class="value muted">…</p></div>
		{:else if score}
			<div class="figure">
				<p class="value" data-testid="score-value">{Math.round(score.score)}</p>
				<span class="delta num" class:up={score.change > 0} class:down={score.change < 0}>
					{signed(score.change)}<small>{deltaLabel}</small>
				</span>
				<div class="gauge" aria-hidden={p > 0.5}>
					<ReserveGauge score={score.score} settle={score.settle} compact />
				</div>
			</div>
		{:else}
			<div class="figure"><p class="value muted">—</p></div>
		{/if}
	</div>
</header>

<style>
	.spacer {
		/* Full-bleed parent column is fine: this only reserves height. */
		pointer-events: none;
	}
	.score-header {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		z-index: 20;
		background: var(--header-bg);
		box-shadow: 0 6px 18px rgba(0, 0, 0, calc(0.18 * var(--p)));
	}
	.inner {
		max-width: 560px;
		margin: 0 auto;
		/* Clear of the iOS status bar blur, like the other pages' bars. */
		padding: calc(env(safe-area-inset-top) + var(--top-gap)) 1.1rem calc(1.1rem - 0.6rem * var(--p));
		min-height: calc(env(safe-area-inset-top) + var(--top-gap) + var(--bar-h) + 8px);
		overflow: hidden;
	}
	.row {
		height: var(--bar-h);
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
	}
	.week {
		margin: 0;
		display: flex;
		flex-direction: column;
		font-size: 0.98rem;
		font-weight: 600;
		letter-spacing: -0.01em;
		white-space: nowrap;
		opacity: calc(1 - 2.5 * var(--p));
	}
	.week .eyebrow {
		margin: 0 0 1px;
		font-size: 0.6rem;
		font-weight: 400;
	}
	.actions {
		display: flex;
		gap: 0.4rem;
		/* Above the score as it slides into the row. */
		position: relative;
		z-index: 1;
	}

	/* Expanded: below the row. Collapsed: pulled up and centred in it. */
	.figure {
		display: flex;
		align-items: flex-end;
		gap: calc(0.9rem - 0.4rem * var(--p));
		margin: calc(-42px * var(--p)) 0 calc(0.5rem * (1 - var(--p)));
		position: relative;
		width: calc(100% - 100px * var(--p));
	}
	.value {
		font-size: calc(3.8rem - 1.4rem * var(--p));
		font-weight: 600;
		line-height: 0.85;
		letter-spacing: -0.06em;
		margin: 0;
		font-variant-numeric: lining-nums tabular-nums;
	}
	/* "−1.3" over "this week"; the label folds away when collapsed. */
	.delta {
		display: flex;
		flex-direction: column;
		padding-bottom: calc(0.25rem * (1 - var(--p)) + 3px * var(--p));
		font-size: 1rem;
		font-weight: 600;
		color: var(--text-secondary);
		white-space: nowrap;
	}
	.delta.up {
		color: var(--positive);
	}
	.delta.down {
		color: var(--negative);
	}
	.delta small {
		overflow: hidden;
		max-height: calc(1rem * (1 - var(--p)));
		margin-top: calc(2px * (1 - var(--p)));
		opacity: calc(1 - 2 * var(--p));
		font-family: var(--font-sans);
		font-size: 0.72rem;
		font-weight: 400;
		color: var(--muted);
	}
	/* Fills the space right of the score. */
	.gauge {
		flex: 1;
		min-width: 0;
		max-width: 240px;
		margin-left: auto;
		overflow: hidden;
		max-height: calc(80px * (1 - var(--p)));
		opacity: calc(1 - 2 * var(--p));
	}
	.error {
		margin: 0.4rem 0 0.6rem;
	}
</style>
