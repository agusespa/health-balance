<script lang="ts">
	/**
	 * One bar per week around a zero line: up and green when the metric added
	 * points, down and red when it cost them. `scale` is the points value that
	 * reaches the top or bottom, shared across rows so they compare.
	 */
	let { points, scale }: { points: number[]; scale: number } = $props();

	const H = 28;
	const mid = H / 2;
	const w = $derived(100 / Math.max(1, points.length));
	const h = (p: number) => (Math.min(1, Math.abs(p) / scale) * (mid - 1)).toFixed(2);
</script>

<svg viewBox={`0 0 100 ${H}`} preserveAspectRatio="none" aria-hidden="true">
	<line x1="0" x2="100" y1={mid} y2={mid} class="zero" />
	{#each points as p, i (i)}
		{#if Math.abs(p) > 0.05}
			<rect
				x={(i * w + w * 0.15).toFixed(2)}
				width={(w * 0.7).toFixed(2)}
				y={p > 0 ? (mid - Number(h(p))).toFixed(2) : mid}
				height={h(p)}
				class:neg={p < 0}
			/>
		{/if}
	{/each}
</svg>

<style>
	svg {
		display: block;
		width: 100%;
		height: 28px;
	}
	.zero {
		stroke: var(--border-strong);
		stroke-width: 1;
		vector-effect: non-scaling-stroke;
	}
	rect {
		fill: var(--positive-fill);
	}
	rect.neg {
		fill: var(--negative-fill);
	}
</style>
