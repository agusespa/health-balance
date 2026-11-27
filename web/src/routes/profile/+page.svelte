<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api } from '#lib/api.ts';
	import { noNewData, shortDate } from '#lib/format.ts';
	import HelpView from '#lib/HelpView.svelte';
	import TopBar from '#lib/TopBar.svelte';
	import ScoreHistory from '#lib/ScoreHistory.svelte';

	const views = [
		{ value: 'history', label: 'Score history' },
		{ value: 'help', label: 'How it works' }
	] as const;

	// The view lives in the URL so a link (or Back from Settings) lands on the same tab.
	const view = $derived(page.url.searchParams.get('view') === 'help' ? 'help' : 'history');

	// null until known; a failed check shows nothing rather than a false alarm.
	let latest = $state<string | null>(null);
	onMount(() => {
		api.latestData().then(
			(r) => (latest = r.latest_data),
			() => {}
		);
	});

	function choose(value: string) {
		goto(value === 'history' ? '/profile' : `/profile?view=${value}`, { replaceState: true });
	}
</script>

<TopBar title="Profile">
	{#snippet actions()}
		<a href="/settings" aria-label="Settings" class="icon-btn">
			<svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
				<path d="M4 7h10M18 7h2M4 17h2M10 17h10" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
				<circle cx="16" cy="7" r="2.2" fill="none" stroke="currentColor" stroke-width="1.8" />
				<circle cx="8" cy="17" r="2.2" fill="none" stroke="currentColor" stroke-width="1.8" />
			</svg>
		</a>
	{/snippet}
</TopBar>

{#if latest !== null && noNewData(latest)}
	<div class="alert" role="alert" data-testid="stale-banner">
		{#if latest}
			<strong>No new health data since {shortDate(latest)}.</strong>
			Check whatever sends your health data, and include every day since then in the next delivery.
		{:else}
			<strong>No health data has arrived yet.</strong>
			Set up a daily delivery of your health data to start scoring.
		{/if}
	</div>
{/if}

<div class="segmented views" role="tablist" aria-label="Profile views">
	{#each views as v (v.value)}
		<button type="button" role="tab" aria-selected={view === v.value} class:active={view === v.value} onclick={() => choose(v.value)}>
			{v.label}
		</button>
	{/each}
</div>

<div role="tabpanel">
	{#if view === 'help'}
		<HelpView />
	{:else}
		<ScoreHistory />
	{/if}
</div>

<style>
	.alert {
		border-left: 2px solid var(--warning);
		background: color-mix(in srgb, var(--warning) 8%, transparent);
		color: var(--text-secondary);
		border-radius: 0 12px 12px 0;
		padding: 0.75rem 0.95rem;
		margin: 0 0 1rem;
		font-size: 0.85rem;
		line-height: 1.45;
	}
	.alert strong {
		display: block;
		color: var(--warning);
		font-weight: 600;
		margin-bottom: 0.15rem;
	}
	.views {
		grid-template-columns: repeat(2, 1fr);
		margin: 0 0 1rem;
	}
</style>
