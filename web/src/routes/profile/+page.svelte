<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import HelpView from '#lib/HelpView.svelte';
	import TopBar from '#lib/TopBar.svelte';
	import ScoreHistory from '#lib/ScoreHistory.svelte';

	const views = [
		{ value: 'history', label: 'Score history' },
		{ value: 'help', label: 'How it works' }
	] as const;

	// The view lives in the URL so a link (or Back from Settings) lands on the same tab.
	const view = $derived(page.url.searchParams.get('view') === 'help' ? 'help' : 'history');

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
	.views {
		grid-template-columns: repeat(2, 1fr);
		margin: 0 0 1rem;
	}
</style>
