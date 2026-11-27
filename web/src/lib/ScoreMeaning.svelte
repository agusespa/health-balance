<script lang="ts">
	import type { MetricScore, WeeklyScore } from '#lib/api.ts';
	import InfoTip from '#lib/InfoTip.svelte';

	/**
	 * The score in plain words: where it stands against the 1000 start, where
	 * weeks like this one would take it, and what costs the most points.
	 */
	let { score }: { score: WeeklyScore } = $props();

	const START = 1000;
	const pts = (n: number) => Math.abs(n).toFixed(1);

	const standing = $derived.by(() => {
		const d = score.score - START;
		if (d >= 40) return 'Well above your starting point. Your health is clearly outpacing aging.';
		if (d >= 10) return "Above your starting point. You're building reserve.";
		if (d > -10) return 'Close to your starting point. Roughly holding even with aging.';
		if (d > -40) return "Below your starting point. You've lost some reserve.";
		return 'Well below your starting point. Several areas are falling short.';
	});

	const settle = $derived(Math.round(score.settle));
	const heading = $derived.by(() => {
		const d = settle - Math.round(score.score);
		if (Math.abs(d) <= 3) return 'Weeks like this one keep it about where it is.';
		return d > 0 ? `Weeks like this one would take it to about ${settle}.` : `Weeks like this one would bring it down to about ${settle}.`;
	});

	const scored = $derived(score.metrics.filter((m) => m.status !== 'missing'));
	const drags = $derived(scored.filter((m) => m.points < -0.05).sort((a, b) => a.points - b.points));
	const room = (m: MetricScore) => m.max_points - m.points;
	const grow = $derived(scored.filter((m) => room(m) > 0.05).sort((a, b) => room(b) - room(a))[0]);
</script>

<section class="card meaning" data-testid="card-meaning">
	<p class="section-label">
		What your score means
		<InfoTip
			label="How the score works"
			id="info-meaning"
			text="1000 is where your score started in January 2024. Each week it moves 12% of the way toward a target set by your metrics, after a small cost for aging. That's why a good week only adds a few points. Watch the direction, not the exact number."
		/>
	</p>
	<p class="standing" data-testid="meaning-standing">{standing}</p>
	<p class="line" data-testid="meaning-heading">{heading}</p>
	<p class="line" data-testid="meaning-improve">
		{#if drags.length}
			{@const top = drags[0]}
			Biggest drag: <strong>{top.label}</strong> ({top.value_text} vs {top.reference_text}), costing {pts(top.points)} points.
			{#if drags.length > 1}
				Then {drags
					.slice(1, 3)
					.map((m) => `${m.label} (−${pts(m.points)})`)
					.join(' and ')}.
			{/if}
		{:else if grow}
			Biggest room to grow: <strong>{grow.label}</strong>, up to {pts(room(grow))} more points.
		{:else}
			Every metric is at its best. Keep it up.
		{/if}
	</p>
</section>

<style>
	/* A card, unlike the flat sections: it's the summary to read first. */
	.meaning {
		margin-bottom: 2rem;
	}
	.meaning p:last-child {
		margin-bottom: 0;
	}
	.section-label {
		display: flex;
		align-items: center;
		gap: 0.25rem;
	}
	p.standing {
		margin: 0 0 0.4rem;
		font-size: 1.05rem;
		line-height: 1.4;
		color: var(--text);
	}
	p.line {
		margin: 0 0 0.3rem;
		font-size: 0.9rem;
		line-height: 1.45;
		color: var(--text-secondary);
	}
	strong {
		font-weight: 600;
		color: var(--text);
	}
</style>
