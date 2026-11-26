<script lang="ts">
	import type { MetricScore } from '#lib/api.ts';
	import { signed } from '#lib/format.ts';

	let { title, hint, metrics }: { title: string; hint: string; metrics: MetricScore[] } = $props();

	const sorted = $derived([...metrics].sort((a, b) => b.points - a.points));
	// Bars share a scale within the card so contributions compare at a glance.
	const scale = $derived(Math.max(8, ...metrics.map((m) => Math.abs(m.points))));
	const total = $derived(metrics.reduce((sum, m) => sum + m.points, 0));

	const statusText: Record<string, string> = {
		carried: 'last value',
		imputed: 'fading',
		missing: 'no data'
	};
</script>

<section class="card" data-testid={`card-${title.toLowerCase()}`}>
	<div class="head">
		<p class="section-label">{title} <span class="hint">· {hint}</span></p>
		<span class="total num" class:pos={total > 0.05} class:neg={total < -0.05}>{signed(total)}</span>
	</div>
	{#each sorted as m (m.metric)}
		<div class="row" data-testid={`metric-${m.metric}`} data-status={m.status}>
			<div class="what">
				<span class="label">
					{m.label}
					{#if m.status !== 'measured'}
						<span class="badge" data-status={m.status}>{statusText[m.status]}</span>
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
		margin: 0;
	}
	.total {
		font-size: 0.75rem;
		color: var(--muted);
	}
	.row {
		display: grid;
		grid-template-columns: 1fr 56px 3.2rem;
		align-items: center;
		gap: 0.9rem;
		padding: 0.7rem 0;
		border-top: 1px solid var(--border);
	}
	.head + .row {
		border-top: none;
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
	.badge {
		font-family: var(--font-mono);
		font-size: 0.6rem;
		letter-spacing: 0.05em;
		text-transform: uppercase;
		color: var(--muted);
		border: 1px solid var(--border-strong);
		border-radius: 999px;
		padding: 1px 6px;
		margin-left: 5px;
		vertical-align: 2px;
	}
	.badge[data-status='imputed'] {
		color: var(--warning);
		border-color: color-mix(in srgb, var(--warning) 40%, transparent);
	}
</style>
