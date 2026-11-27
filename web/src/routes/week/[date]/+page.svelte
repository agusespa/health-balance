<script lang="ts">
	import { page } from '$app/state';
	import { api, type WeeklyScore } from '#lib/api.ts';
	import MetricCard from '#lib/MetricCard.svelte';
	import ReserveGauge from '#lib/ReserveGauge.svelte';
	import TopBar from '#lib/TopBar.svelte';
	import { shortDate, signed } from '#lib/format.ts';

	let week = $state<WeeklyScore | null>(null);
	let error = $state('');

	$effect(() => {
		const date = page.params.date ?? '';
		week = null;
		error = '';
		api.week(date).then(
			(w) => (week = w),
			(e) => (error = (e as Error).message)
		);
	});
</script>

<TopBar title={`Week of ${shortDate(page.params.date ?? '')}`} />

{#if error}
	<p class="error">{error}</p>
{:else if !week}
	<p class="muted">Loading…</p>
{:else}
	<section class="band band-hero summary">
		<div class="wrap">
		<div class="figure">
			<span class="score">{week.score.toFixed(1)}</span>
			<span class="num delta" class:pos={week.change > 0} class:neg={week.change < 0}>{signed(week.change)}</span>
		</div>
		<ReserveGauge score={week.score} target={week.target} />
		<p class="facts num">target {week.target.toFixed(1)}&ensp;·&ensp;aging tax −{week.aging_tax.toFixed(1)}</p>
		</div>
	</section>
	<MetricCard title="Markers" hint="carried forward" metrics={week.metrics.filter((m) => m.kind === 'marker')} />
	<MetricCard title="Behaviours" hint="4-week average" metrics={week.metrics.filter((m) => m.kind === 'behaviour')} />
{/if}

<style>
	.figure {
		display: flex;
		align-items: baseline;
		gap: 0.8rem;
		margin-bottom: 0.8rem;
	}
	.score {
		font-family: var(--font-display);
		font-size: 3.2rem;
		font-weight: 600;
		line-height: 1;
		letter-spacing: -0.05em;
	}
	.delta {
		font-weight: 600;
		color: var(--text-secondary);
	}
	.delta.pos {
		color: var(--positive);
	}
	.delta.neg {
		color: var(--negative);
	}
	.facts {
		font-size: 0.72rem;
		color: var(--muted);
		margin: 0.6rem 0 0;
	}
</style>
