<script lang="ts">
	import type { MetricScore } from '#lib/api.ts';
	import { signed } from '#lib/format.ts';
	import MetricIcon from '#lib/MetricIcon.svelte';

	// `info` is a one-line explanation of how this card is scored, shown in a tooltip.
	let { title, info, metrics }: { title: string; info: string; metrics: MetricScore[] } = $props();

	let open = $state(false);
	let infoEl: HTMLElement | undefined = $state();
	const tipId = $derived(`info-${title.toLowerCase()}`);

	function closeOutside(e: PointerEvent) {
		if (open && infoEl && !infoEl.contains(e.target as Node)) open = false;
	}

	const sorted = $derived([...metrics].sort((a, b) => b.points - a.points));
	// Bars share a scale within the card so contributions compare at a glance.
	const scale = $derived(Math.max(8, ...metrics.map((m) => Math.abs(m.points))));
	const total = $derived(metrics.reduce((sum, m) => sum + m.points, 0));
</script>

<svelte:window onpointerdown={closeOutside} onkeydown={(e) => e.key === 'Escape' && (open = false)} />

<section class="section" data-testid={`card-${title.toLowerCase()}`}>
	<div class="head">
		<p class="section-label">
			{title}
			<span class="info-wrap" bind:this={infoEl}>
				<button
					type="button"
					class="info"
					aria-label={`How ${title.toLowerCase()} are scored`}
					aria-expanded={open}
					aria-describedby={open ? tipId : undefined}
					onclick={() => (open = !open)}
				>
					<svg viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">
						<circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="1.8" />
						<path d="M12 11v5.5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
						<circle cx="12" cy="7.8" r="1.1" fill="currentColor" />
					</svg>
				</button>
				{#if open}
					<span class="tip" role="tooltip" id={tipId}>{info}</span>
				{/if}
			</span>
		</p>
		<span class="total num" class:pos={total > 0.05} class:neg={total < -0.05}>{signed(total)}</span>
	</div>
	{#each sorted as m (m.metric)}
		<div class="row" data-testid={`metric-${m.metric}`} data-status={m.status}>
			<MetricIcon metric={m.metric} />
			<div class="what">
				<span class="label">
					{m.label}
					{#if m.weeks_old}
						<span class="age" title="Last measured {m.weeks_old} {m.weeks_old === 1 ? 'week' : 'weeks'} ago">{m.weeks_old}w old</span>
					{/if}
				</span>
				<span class="sub">
					{#if m.status === 'missing'}
						no data yet
					{:else}
						{m.value_text} <span class="vs">vs</span> {m.reference_text}
					{/if}
				</span>
			</div>
			<div class="track" aria-hidden="true">
				<span
					class="fill"
					class:neg={m.points < 0}
					style:width={`${(Math.min(1, Math.abs(m.points) / scale) * 50).toFixed(1)}%`}
				></span>
			</div>
			<span class="points num" class:pos={m.points > 0.05} class:neg={m.points < -0.05}>{signed(m.points)}</span>
		</div>
	{/each}
</section>

<style>
	.head {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		margin-bottom: 0.15rem;
	}
	.head .section-label {
		display: flex;
		align-items: center;
		gap: 0.15rem;
		margin: 0;
	}
	.info-wrap {
		position: relative;
		display: inline-flex;
	}
	/* Small visible icon, larger tap target. */
	.info {
		appearance: none;
		border: none;
		padding: 0;
		background: none;
		cursor: pointer;
		display: grid;
		place-items: center;
		width: 28px;
		height: 28px;
		margin: -8px 0;
		border-radius: 999px;
		color: var(--muted);
	}
	.tip {
		position: absolute;
		top: calc(100% + 6px);
		left: -10px;
		z-index: 5;
		width: max-content;
		max-width: min(260px, 70vw);
		padding: 0.55rem 0.7rem;
		border: 1px solid var(--border-strong);
		border-radius: 10px;
		background: var(--surface);
		box-shadow: var(--shadow);
		font-family: var(--font-sans);
		font-size: 0.8rem;
		line-height: 1.45;
		letter-spacing: normal;
		text-transform: none;
		color: var(--text-secondary);
	}
	.info:hover,
	.info[aria-expanded='true'] {
		color: var(--text-secondary);
		background: var(--surface-muted);
	}
	.total {
		font-size: 0.75rem;
		color: var(--muted);
	}
	.row {
		display: grid;
		grid-template-columns: 30px 1fr 56px 3.2rem;
		align-items: center;
		gap: 0.75rem;
		padding: 0.7rem 0;
		border-top: 1px solid var(--border);
	}
	.head + .row {
		border-top: none;
	}
	.row[data-status='missing'] :global(.icon) {
		opacity: 0.5;
	}
	.row[data-status='missing'] .label {
		color: var(--text-secondary);
	}
	.what {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}
	.label {
		font-size: 0.95rem;
	}
	.sub {
		color: var(--muted);
		font-size: 0.78rem;
		margin-top: 3px;
		font-variant-numeric: tabular-nums;
	}
	.vs {
		opacity: 0.7;
	}
	.track {
		position: relative;
		height: 6px;
		border-radius: 999px;
		background: var(--surface-muted);
	}
	.track::before {
		content: '';
		position: absolute;
		left: 50%;
		top: -3px;
		bottom: -3px;
		width: 1px;
		background: var(--border-strong);
	}
	.fill {
		position: absolute;
		left: 50%;
		top: 0;
		bottom: 0;
		border-radius: 0 999px 999px 0;
		background: var(--positive-fill);
	}
	.fill.neg {
		left: auto;
		right: 50%;
		border-radius: 999px 0 0 999px;
		background: var(--negative-fill);
	}
	.points {
		text-align: right;
		font-size: 0.88rem;
		font-weight: 600;
		color: var(--text-secondary);
	}
	/* Short and never split: the label column is narrow on a phone. */
	.age {
		display: inline-block;
		white-space: nowrap;
		font-family: var(--font-mono);
		font-size: 0.6rem;
		letter-spacing: 0.03em;
		color: var(--muted);
		border: 1px solid var(--border-strong);
		border-radius: 999px;
		padding: 0 6px;
		margin-left: 4px;
		vertical-align: 2px;
	}
</style>
