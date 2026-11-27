<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Dashboard } from '#lib/api.ts';
	import MetricCard from '#lib/MetricCard.svelte';
	import NutritionCard from '#lib/NutritionCard.svelte';
	import PullToRefresh from '#lib/PullToRefresh.svelte';
	import ScoreHeader from '#lib/ScoreHeader.svelte';
	import ScoreMeaning from '#lib/ScoreMeaning.svelte';
	import { noNewData } from '#lib/format.ts';

	let data = $state<Dashboard | null>(null);
	let error = $state('');

	async function load() {
		try {
			data = await api.dashboard();
			error = '';
		} catch (e) {
			error = (e as Error).message;
		}
	}

	onMount(load);


	const score = $derived(data?.score ?? null);
	const markers = $derived(score?.metrics.filter((m) => m.kind === 'marker') ?? []);
	const behaviours = $derived(score?.metrics.filter((m) => m.kind === 'behaviour') ?? []);
	// The notice itself is on the profile page; here the profile icon gets a badge.
	const stale = $derived(!!data && noNewData(data.latest_data));

</script>

<PullToRefresh onrefresh={load} />

<ScoreHeader eyebrow="Last week" deltaLabel="vs prior week" title={data?.week.range ?? ''} {score} loading={!data} {error}>
	{#snippet actions()}
		<a href="/insights" aria-label="AI insights" class="icon-btn">
			<svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
				<path d="M12 3.5l1.9 5.1 5.1 1.9-5.1 1.9L12 17.5l-1.9-5.1L5 10.5l5.1-1.9z" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round" />
				<path d="M18.5 15.5l.8 2.2 2.2.8-2.2.8-.8 2.2-.8-2.2-2.2-.8 2.2-.8z" fill="currentColor" />
			</svg>
		</a>
		<a href="/trends" aria-label="Trends" class="icon-btn">
			<svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
				<path d="M4 4v16h16" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" opacity="0.5" />
				<path d="M7.5 16c2.6 0 3.2-4.5 5.8-4.5s3-4.2 5.2-5" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
				<circle cx="18.5" cy="6.5" r="1.9" fill="currentColor" />
			</svg>
		</a>
		<a href="/profile" aria-label={stale ? 'Profile: no new health data' : 'Profile'} class="icon-btn badged">
			<svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
				<circle cx="12" cy="8.5" r="3.6" fill="none" stroke="currentColor" stroke-width="1.8" />
				<path d="M5 19.5c1.2-3.2 3.9-4.8 7-4.8s5.8 1.6 7 4.8" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
			</svg>
			{#if stale}
				<span class="badge" data-testid="stale-badge"></span>
			{/if}
		</a>
	{/snippet}
</ScoreHeader>

{#if data && !data.profile_complete}
	<section class="card">
		<h2>Set up your profile</h2>
		<p class="muted">The score needs your birth date, sex and height.</p>
		<a href="/settings">Open Settings</a>
	</section>
{/if}

{#if score}
	<ScoreMeaning {score} />
	<MetricCard title="Markers" info="Not measured every week. When a week has no new reading, the last one is reused for 2 to 16 weeks depending on the marker, then fades by a quarter each week." metrics={markers} />
	<MetricCard title="Behaviours" info="Each week scores the average of the last 4 weeks, so a steady habit counts more than one good or bad week." metrics={behaviours} />
{/if}

{#if data?.nutrition}
	<NutritionCard nutrition={data.nutrition} />
{/if}


<style>
	.badged {
		position: relative;
	}
	/* --warning is a light yellow on the coloured header. */
	.badge {
		position: absolute;
		top: 1px;
		right: 1px;
		width: 10px;
		height: 10px;
		border-radius: 999px;
		background: var(--warning);
		box-shadow: 0 0 0 2px rgba(0, 0, 0, 0.18);
	}
</style>
