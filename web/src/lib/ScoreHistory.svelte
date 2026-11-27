<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type ScoreSummary } from '#lib/api.ts';
	import { shortDate, signed } from '#lib/format.ts';

	let history = $state<ScoreSummary[] | null>(null);
	let error = $state('');

	onMount(async () => {
		try {
			history = await api.scores();
		} catch (e) {
			error = (e as Error).message;
		}
	});
</script>

<section class="card" data-testid="score-history">
	{#if error}
		<p class="error">{error}</p>
	{:else if !history}
		<p class="muted">Loading…</p>
	{:else if history.length === 0}
		<p class="muted">No weeks scored yet.</p>
	{:else}
		<ul class="week-list">
			{#each history as h (h.week_end)}
				<li>
					<a href={`/week/${h.week_end}`}>
						<span>{shortDate(h.week_end)}</span>
						<span class="num">{h.score.toFixed(1)}</span>
						<span class="num" class:pos={h.change > 0} class:neg={h.change < 0}>{signed(h.change)}</span>
					</a>
				</li>
			{/each}
		</ul>
	{/if}
</section>
