<script lang="ts">
	interface Point {
		x: string;
		y: number;
	}

	let {
		points,
		baseline,
		height = 110,
		format = (v: number) => v.toFixed(0)
	}: { points: Point[]; baseline?: number; height?: number; format?: (v: number) => string } = $props();

	const uid = $props.id();
	const width = 320;
	const pad = 6;

	const ys = $derived(points.map((p) => p.y).concat(baseline === undefined ? [] : [baseline]));
	const min = $derived(Math.min(...ys));
	const max = $derived(Math.max(...ys));
	const span = $derived(max - min || 1);

	const sx = (i: number) => pad + (i / Math.max(1, points.length - 1)) * (width - 2 * pad);
	const sy = (v: number) => pad + (1 - (v - min) / span) * (height - 2 * pad);

	const path = $derived(points.map((p, i) => `${i ? 'L' : 'M'}${sx(i).toFixed(1)},${sy(p.y).toFixed(1)}`).join(' '));
	const area = $derived(`${path} L${sx(points.length - 1).toFixed(1)},${height} L${sx(0).toFixed(1)},${height} Z`);
	const last = $derived(points.at(-1));
	const lo = $derived(Math.min(...points.map((p) => p.y)));
	const hi = $derived(Math.max(...points.map((p) => p.y)));
</script>

{#if points.length > 1}
	<figure>
		<div class="readout">
			<span class="now num">{format(last?.y ?? 0)}</span>
			<span class="range num">{format(lo)}–{format(hi)}</span>
		</div>
		<svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label="trend">
			<defs>
				<linearGradient id={`fade-${uid}`} x1="0" x2="0" y1="0" y2="1">
					<stop offset="0" style="stop-color: var(--chart-fill)" stop-opacity="0.32" />
					<stop offset="1" style="stop-color: var(--chart-fill)" stop-opacity="0" />
				</linearGradient>
			</defs>
			{#if baseline !== undefined}
				<line x1="0" x2={width} y1={sy(baseline)} y2={sy(baseline)} class="base" />
			{/if}
			<path d={area} fill={`url(#fade-${uid})`} />
			<path d={path} class="line" />
			{#if last}
				<circle cx={sx(points.length - 1)} cy={sy(last.y)} r="5" class="halo" />
				<circle cx={sx(points.length - 1)} cy={sy(last.y)} r="2.6" class="dot" />
			{/if}
		</svg>
		<figcaption class="num">
			<span>{points[0].x}</span>
			<span>{last?.x}</span>
		</figcaption>
	</figure>
{:else}
	<p class="muted">Not enough data yet.</p>
{/if}

<style>
	figure {
		margin: 0;
	}
	.readout {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		margin: -0.2rem 0 0.5rem;
	}
	.now {
		font-family: var(--font-display);
		font-size: 1.6rem;
		font-weight: 600;
		letter-spacing: -0.03em;
		line-height: 1;
	}
	.range {
		font-size: 0.7rem;
		color: var(--muted);
	}
	svg {
		width: 100%;
		height: auto;
		display: block;
		overflow: visible;
	}
	.line {
		fill: none;
		stroke: var(--chart-line);
		stroke-width: 1.75;
		stroke-linejoin: round;
		stroke-linecap: round;
	}
	.dot {
		fill: var(--chart-fill);
		stroke: var(--chart-line);
		stroke-width: 1.5;
	}
	.halo {
		fill: var(--chart-fill);
		opacity: 0.3;
	}
	.base {
		stroke: var(--text-secondary);
		stroke-opacity: 0.5;
		stroke-dasharray: 2 4;
		stroke-width: 1;
	}
	figcaption {
		display: flex;
		justify-content: space-between;
		gap: 0.5rem;
		font-size: 0.66rem;
		letter-spacing: 0.04em;
		color: var(--muted);
		margin-top: 0.45rem;
		text-transform: uppercase;
	}
</style>
