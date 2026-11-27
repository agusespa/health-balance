<script lang="ts">
	import { page } from '$app/state';
	import { api, type WeeklyScore } from '#lib/api.ts';
	import MetricCard from '#lib/MetricCard.svelte';
	import ScoreHeader from '#lib/ScoreHeader.svelte';
	import ScoreMeaning from '#lib/ScoreMeaning.svelte';
	import { weekRange } from '#lib/format.ts';

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

<ScoreHeader eyebrow="Week" title={weekRange(page.params.date ?? '')} score={week} loading={!week && !error} {error} deltaLabel="vs prior week">
	{#snippet actions()}
		<a href="/profile" aria-label="Back" class="icon-btn">
			<svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
				<path d="M6 6l12 12M18 6L6 18" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
			</svg>
		</a>
	{/snippet}
</ScoreHeader>

{#if week}
	<p class="facts num">target {week.target.toFixed(1)}&ensp;·&ensp;aging tax −{week.aging_tax.toFixed(1)}&ensp;·&ensp;levels off at {week.settle.toFixed(1)}</p>
	<ScoreMeaning score={week} />
	<MetricCard title="Markers" info="Not measured every week. When a week has no new reading, the last one is reused for 2 to 16 weeks depending on the marker, then fades by a quarter each week." metrics={week.metrics.filter((m) => m.kind === 'marker')} />
	<MetricCard title="Behaviours" info="Each week scores the average of the last 4 weeks, so a steady habit counts more than one good or bad week." metrics={week.metrics.filter((m) => m.kind === 'behaviour')} />
{/if}

<style>
	.facts {
		font-size: 0.72rem;
		color: var(--muted);
		margin: 0 0.2rem 0.75rem;
	}
</style>
