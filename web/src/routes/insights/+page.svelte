<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Insight } from '#lib/api.ts';
	import TopBar from '#lib/TopBar.svelte';
	import { shortDate } from '#lib/format.ts';

	// Insights are written only when asked for: each one is a Claude run.
	let insight = $state<Insight | null | undefined>(undefined);
	let error = $state('');
	let writing = $state(false);

	onMount(async () => {
		try {
			insight = await api.insights();
		} catch (e) {
			error = (e as Error).message;
		}
	});

	async function write() {
		writing = true;
		error = '';
		try {
			insight = await api.regenerateInsight();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			writing = false;
		}
	}
</script>

<TopBar title="AI insights" />

<section class="card">
	{#if insight === undefined && !error}
		<p class="muted">Loading…</p>
	{:else if writing}
		<p class="muted" data-testid="insight-writing">Writing… this can take a minute.</p>
	{:else if insight}
		<p class="section-label">Week ending {shortDate(insight.week_end)}</p>
		<div class="insight" data-testid="insight">{@html insight.html}</div>
	{:else if !error}
		<p class="muted">No insights yet. Write them when you want a summary of the latest week.</p>
	{/if}
	{#if error}
		<p class="error" data-testid="insight-error">{error}</p>
	{/if}
	{#if insight !== undefined || error}
		<button class="write" onclick={write} disabled={writing}>
			{insight ? 'Regenerate' : 'Write insights now'}
		</button>
	{/if}
</section>

<style>
	.write {
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
