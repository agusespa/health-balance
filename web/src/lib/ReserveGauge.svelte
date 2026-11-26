<script lang="ts">
	/**
	 * A ruler centred on the 1000 baseline. It zooms to the score's neighbourhood
	 * (±50 at least) so week-to-week movement is visible, and marks the target the
	 * score is drifting toward. `compact` draws a narrower ruler with fewer
	 * ticks and larger labels, for sitting beside the score.
	 */
	let { score, target, compact = false }: { score: number; target: number; compact?: boolean } = $props();

	const BASE = 1000;
	const W = $derived(compact ? 200 : 320);
	const PAD = 12;
	const AXIS = 30;

	const half = $derived(Math.max(50, Math.ceil((Math.max(Math.abs(score - BASE), Math.abs(target - BASE)) + 10) / 25) * 25));
	const lo = $derived(BASE - half);
	const hi = $derived(BASE + half);
	const x = (v: number) => PAD + ((Math.min(hi, Math.max(lo, v)) - lo) / (hi - lo)) * (W - 2 * PAD);

	const steps = $derived(compact ? 10 : 20);
	const ticks = $derived(Array.from({ length: steps + 1 }, (_, i) => lo + (i * (hi - lo)) / steps));
	const below = $derived(score < BASE);
</script>

<svg viewBox={`0 0 ${W} 56`} class:compact role="img" aria-label={`Score ${Math.round(score)} against a baseline of ${BASE}, target ${Math.round(target)}`}>
	{#each ticks as t, i (i)}
		<line x1={x(t)} x2={x(t)} y1={AXIS - (i % 5 === 0 ? 10 : 5)} y2={AXIS} class="tick" class:major={i % 5 === 0} />
	{/each}
	<line x1={PAD} x2={W - PAD} y1={AXIS} y2={AXIS} class="axis" />

	<rect
		x={Math.min(x(BASE), x(score))}
		y={AXIS - 2}
		width={Math.max(1.5, Math.abs(x(score) - x(BASE)))}
		height="4"
		rx="2"
		class="band"
		class:below
	/>

	<path d={`M${x(target)} ${AXIS + 4} l4 6 h-8 z`} class="target" />

	<line x1={x(score)} x2={x(score)} y1={AXIS - 18} y2={AXIS + 2} class="needle" class:below />
	<circle cx={x(score)} cy={AXIS - 18} r="4" class="head" class:below />

	<text x={PAD} y="52" class="lbl" text-anchor="start">{lo}</text>
	<text x={x(BASE)} y="52" class="lbl base" text-anchor="middle">{BASE}</text>
	<text x={W - PAD} y="52" class="lbl" text-anchor="end">{hi}</text>
</svg>

<style>
	svg {
		display: block;
		width: 100%;
		height: auto;
		overflow: visible;
	}
	.tick {
		stroke: var(--border-strong);
		stroke-width: 1;
	}
	.tick.major {
		stroke: color-mix(in srgb, var(--text) 35%, transparent);
	}
	.axis {
		stroke: var(--border-strong);
		stroke-width: 1;
	}
	.band {
		fill: var(--positive);
		opacity: 0.45;
	}
	.band.below {
		fill: var(--negative);
	}
	.needle {
		stroke: var(--positive);
		stroke-width: 2;
		stroke-linecap: round;
	}
	.head {
		fill: var(--positive);
	}
	.needle.below {
		stroke: var(--negative);
	}
	.head.below {
		fill: var(--negative);
	}
	.target {
		fill: none;
		stroke: var(--text-secondary);
		stroke-width: 1.2;
		stroke-linejoin: round;
	}
	.lbl {
		font-family: var(--font-mono);
		font-size: 9px;
		fill: var(--muted);
		letter-spacing: 0.04em;
	}
	.compact .lbl {
		font-size: 11px;
	}
	.lbl.base {
		fill: var(--text-secondary);
	}
</style>
