<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Drivers } from '#lib/api.ts';
	import InfoTip from '#lib/InfoTip.svelte';
	import LineChart from '#lib/LineChart.svelte';
	import MetricIcon from '#lib/MetricIcon.svelte';
	import PointsBars from '#lib/PointsBars.svelte';
	import PullToRefresh from '#lib/PullToRefresh.svelte';
	import TopBar from '#lib/TopBar.svelte';
	import { shortDate, signed } from '#lib/format.ts';

	/**
	 * The score over time and what drove it: each metric's points week by week.
	 * Raw health values are left to the health app; this page shows only what
	 * the score adds on top.
	 */
	let data = $state<Drivers | null>(null);
	let error = $state('');

	const ranges = [
		{ value: 13, label: '3M', words: 'over the last 3 months' },
		{ value: 26, label: '6M', words: 'over the last 6 months' },
		{ value: 52, label: '1Y', words: 'over the last year' },
		{ value: 0, label: 'All', words: '' }
	] as const;
	let range = $state<number>(26);

	async function load() {
		try {
			data = await api.drivers();
			error = '';
		} catch (e) {
			error = (e as Error).message;
		}
	}

	onMount(load);

	const weeks = $derived(data ? (range ? data.weeks.slice(-range) : data.weeks) : []);
	const first = $derived(weeks[0]);
	const words = $derived(ranges.find((r) => r.value === range)?.words || (first ? `since ${shortDate(first.week_end)}` : ''));
	const last = $derived(weeks.at(-1));

	const drivers = $derived(
		(data?.metrics ?? [])
			.map((m, i) => {
				const points = weeks.map((w) => w.points[i]);
				const avg = points.reduce((sum, p) => sum + p, 0) / Math.max(1, points.length);
				return { ...m, points, avg };
			})
			.sort((a, b) => b.avg - a.avg)
	);
	// Bars share a scale across rows so contributions compare at a glance.
	const scale = $derived(Math.max(8, ...drivers.flatMap((d) => d.points.map(Math.abs))));
	const total = $derived(drivers.reduce((sum, d) => sum + d.avg, 0));
	const aging = $derived(weeks.reduce((sum, w) => sum + w.aging, 0) / Math.max(1, weeks.length));
	// Metrics minus aging, from 1000, is roughly where the score levels off.
	const levelsOff = $derived(Math.round(1000 + total - aging));
</script>

<PullToRefresh onrefresh={load} />

<TopBar title="Trends" />

<div class="segmented ranges" role="tablist" aria-label="Period">
	{#each ranges as r (r.value)}
		<button type="button" role="tab" aria-selected={range === r.value} class:active={range === r.value} onclick={() => (range = r.value)}>
			{r.label}
		</button>
	{/each}
</div>

{#if error}
	<p class="error">{error}</p>
{:else if !data}
	<p class="muted">Loading…</p>
{:else if !first || !last}
	<p class="muted">No weeks scored yet.</p>
{:else}
	<section class="section" data-testid="trend-score">
		<p class="section-label">Score</p>
		<p class="summary" data-testid="trend-summary">
			From {Math.round(first.score)} to {Math.round(last.score)} {words}
			<span class="num" class:pos={last.score - first.score > 0.5} class:neg={last.score - first.score < -0.5}>({signed(last.score - first.score)})</span>
		</p>
		<LineChart points={weeks.map((w) => ({ x: shortDate(w.week_end), y: w.score }))} baseline={1000} height={140} />
	</section>

	<section class="section" data-testid="trend-drivers">
		<div class="head">
			<p class="section-label">
				What drove it
				<InfoTip
					label="How drivers are measured"
					id="info-drivers"
					text="Each bar is one week: how many points the metric added to that week's target (up) or cost it (down). The number is the average over the period. Aging takes points too. 1000 plus metrics minus aging is about where the score levels off; it climbs when it's below that and falls when it's above."
				/>
			</p>
			<span class="total num" class:pos={total > 0.05} class:neg={total < -0.05}>{signed(total)}</span>
		</div>
		{#each drivers as d (d.metric)}
			<div class="row" data-testid={`driver-${d.metric}`}>
				<MetricIcon metric={d.metric} />
				<div class="what">
					<span class="label">{d.label}</span>
					<span class="sub">
						{#if d.status === 'missing'}
							no data yet
						{:else}
							now {d.value_text} <span class="vs">vs</span> {d.reference_text}
						{/if}
					</span>
				</div>
				<div class="bars"><PointsBars points={d.points} {scale} /></div>
				<span class="points num" class:pos={d.avg > 0.05} class:neg={d.avg < -0.05}>{signed(d.avg)}</span>
			</div>
		{/each}
		<div class="net" data-testid="trend-net">
			<span>Metrics <span class="num">{signed(total)}</span>, aging <span class="num">−{aging.toFixed(1)}</span></span>
			<span>levels off near <strong class="num">{levelsOff}</strong></span>
		</div>
	</section>
{/if}

<style>
	.ranges {
		grid-template-columns: repeat(4, 1fr);
		margin: 0 0 1.25rem;
	}
	.summary {
		margin: 0 0 0.9rem;
		font-size: 0.95rem;
		color: var(--text-secondary);
	}
	.summary .num {
		font-size: 0.85rem;
	}
	.head {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		margin-bottom: 0.15rem;
	}
	.head .section-label {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		margin: 0;
	}
	.total {
		font-size: 0.75rem;
		color: var(--muted);
	}
	.row {
		display: grid;
		grid-template-columns: 30px 1fr 84px 3.2rem;
		align-items: center;
		gap: 0.75rem;
		padding: 0.7rem 0;
		border-top: 1px solid var(--border);
	}
	.head + .row {
		border-top: none;
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
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.vs {
		opacity: 0.7;
	}
	.net {
		display: flex;
		justify-content: space-between;
		gap: 0.75rem;
		padding: 0.8rem 0 0;
		border-top: 1px solid var(--border-strong);
		font-size: 0.82rem;
		color: var(--muted);
	}
	.net strong {
		font-weight: 600;
		color: var(--text-secondary);
	}
	.points {
		text-align: right;
		font-size: 0.88rem;
		font-weight: 600;
		color: var(--text-secondary);
	}
</style>
