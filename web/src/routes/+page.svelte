<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Dashboard, type Insight } from '#lib/api.ts';
	import Collapsible from '#lib/Collapsible.svelte';
	import MetricCard from '#lib/MetricCard.svelte';
	import NutritionCard from '#lib/NutritionCard.svelte';
	import ScoreHeader from '#lib/ScoreHeader.svelte';
	import { daysAgo, shortDate } from '#lib/format.ts';

	let data = $state<Dashboard | null>(null);
	let error = $state('');
	let insight = $state<Insight | null | undefined>(undefined);
	let insightError = $state('');
	let regenerating = $state(false);

	onMount(async () => {
		try {
			data = await api.dashboard();
		} catch (e) {
			error = (e as Error).message;
		}
	});

	async function loadInsight() {
		if (insight !== undefined) return;
		try {
			insight = await api.insights();
		} catch (e) {
			insightError = (e as Error).message;
		}
	}

	async function regenerate() {
		regenerating = true;
		insightError = '';
		try {
			insight = await api.regenerateInsight();
		} catch (e) {
			insightError = (e as Error).message;
		} finally {
			regenerating = false;
		}
	}

	const score = $derived(data?.score ?? null);
	const markers = $derived(score?.metrics.filter((m) => m.kind === 'marker') ?? []);
	const behaviours = $derived(score?.metrics.filter((m) => m.kind === 'behaviour') ?? []);
	// The daily delivery includes today, so a newest day before the day before
	// yesterday means nothing new has arrived for at least 48 hours.
	const stale = $derived(!!data && data.latest_data < daysAgo(2));

</script>

<ScoreHeader eyebrow="This week" title={data?.week.range ?? ''} {score} loading={!data} {error}>
	{#snippet actions()}
		<a href="/trends" aria-label="Trends" class="icon-btn">
			<svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
				<path d="M4 17l5-5 4 3 7-8" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
			</svg>
		</a>
		<a href="/profile" aria-label="Profile" class="icon-btn">
			<svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
				<circle cx="12" cy="8.5" r="3.6" fill="none" stroke="currentColor" stroke-width="1.8" />
				<path d="M5 19.5c1.2-3.2 3.9-4.8 7-4.8s5.8 1.6 7 4.8" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
			</svg>
		</a>
	{/snippet}
</ScoreHeader>

{#if stale && data}
	<div class="alert" role="alert" data-testid="stale-banner">
		{#if data.latest_data}
			<strong>No new health data since {shortDate(data.latest_data)}.</strong>
			Check whatever sends your health data, and include every day since then in the next delivery.
		{:else}
			<strong>No health data has arrived yet.</strong>
			Set up a daily delivery of your health data to start scoring.
		{/if}
	</div>
{/if}

{#if data && !data.profile_complete}
	<section class="card">
		<h2>Set up your profile</h2>
		<p class="muted">The score needs your birth date, sex and height.</p>
		<a href="/settings">Open Settings</a>
	</section>
{/if}

{#if score}
	<MetricCard title="Markers" info="Not measured every week. When a week has no new reading, the last one is reused for 2 to 16 weeks depending on the marker, then fades by a quarter each week." metrics={markers} />
	<MetricCard title="Behaviours" info="Each week scores the average of the last 4 weeks, so a steady habit counts more than one good or bad week." metrics={behaviours} />
{/if}

{#if data?.nutrition}
	<NutritionCard nutrition={data.nutrition} />
{/if}

<Collapsible title="AI insights" onopen={loadInsight}>
	{#if insight === undefined && !insightError}
		<p class="muted">Loading…</p>
	{:else if regenerating}
		<p class="muted" data-testid="insight-writing">Writing… this can take a minute.</p>
	{:else if insight}
		<p class="section-label">Week ending {shortDate(insight.week_end)}</p>
		<div class="insight" data-testid="insight">{@html insight.html}</div>
	{:else if !insightError}
		<p class="muted">No insights yet. Write them when you want a summary of the latest week.</p>
	{/if}
	{#if insightError}
		<p class="error" data-testid="insight-error">{insightError}</p>
	{/if}
	{#if insight !== undefined || insightError}
		<button class="regenerate" onclick={regenerate} disabled={regenerating}>
			{insight ? 'Regenerate' : 'Write insights now'}
		</button>
	{/if}
</Collapsible>

<style>
	.alert {
		border-left: 2px solid var(--warning);
		background: color-mix(in srgb, var(--warning) 8%, transparent);
		color: var(--text-secondary);
		border-radius: 0 12px 12px 0;
		padding: 0.75rem 0.95rem;
		margin: 0 0 0.75rem;
		font-size: 0.85rem;
		line-height: 1.45;
	}
	.alert strong {
		display: block;
		color: var(--warning);
		font-weight: 600;
		margin-bottom: 0.15rem;
	}
	.regenerate {
		margin-top: 0.6rem;
	}
	.insight :global(p),
	.insight :global(li) {
		line-height: 1.55;
		color: var(--text-secondary);
	}
	.insight :global(strong) {
		color: var(--text);
	}
</style>
