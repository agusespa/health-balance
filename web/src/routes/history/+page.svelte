<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type DailyMetric, type ScoreSummary } from '#lib/api.ts';
	import LineChart from '#lib/LineChart.svelte';
	import TopBar from '#lib/TopBar.svelte';
	import { daysAgo, isoDate, shortDate, signed } from '#lib/format.ts';

	let scores = $state<ScoreSummary[] | null>(null);
	let daily = $state<DailyMetric[]>([]);
	let error = $state('');

	const trends = [
		{ metric: 'rhr', label: 'Resting HR', days: 90, format: (v: number) => `${v.toFixed(0)}` },
		{ metric: 'hrv_sdnn', label: 'HRV (ms)', days: 90, format: (v: number) => `${v.toFixed(0)}` },
		{ metric: 'sleep_hours', label: 'Sleep (hours)', days: 90, format: (v: number) => v.toFixed(1) },
		{ metric: 'steps', label: 'Steps', days: 90, format: (v: number) => `${(v / 1000).toFixed(1)}k` },
		{ metric: 'vo2_max', label: 'VO2 max', days: 730, format: (v: number) => v.toFixed(1) },
		{ metric: 'waist_cm', label: 'Waist (cm)', days: 730, format: (v: number) => v.toFixed(1) }
	];

	onMount(async () => {
		try {
			const [s, d] = await Promise.all([
				api.scores(),
				api.daily(daysAgo(730), isoDate(new Date()), trends.map((t) => t.metric))
			]);
			scores = s;
			daily = d;
		} catch (e) {
			error = (e as Error).message;
		}
	});

	const scorePoints = $derived((scores ?? []).toReversed().map((s) => ({ x: shortDate(s.week_end), y: s.score })));

	/** Daily values; the 90-day ones as a 7-day rolling average, which reads better than raw day-to-day noise. */
	function series(metric: string, days: number) {
		const from = daysAgo(days);
		const window = days <= 90 ? 7 : 1;
		// Include the days before the window so its first points average a full week too.
		const rows = daily.filter((d) => d.metric === metric && d.date >= daysAgo(days + window - 1));
		return rows
			.map((d, i) => {
				const slice = rows.slice(Math.max(0, i - window + 1), i + 1);
				return { date: d.date, x: shortDate(d.date), y: slice.reduce((sum, r) => sum + r.value, 0) / slice.length };
			})
			.filter((p) => p.date >= from);
	}
</script>

<TopBar title="History" />

{#if error}
	<p class="error">{error}</p>
{:else if !scores}
	<p class="muted">Loading…</p>
{:else}
	<section class="card">
		<p class="section-label">Score since {shortDate(scores.at(-1)?.week_end ?? '')}</p>
		<LineChart points={scorePoints} baseline={1000} format={(v) => v.toFixed(0)} />
	</section>

	{#each trends as t (t.metric)}
		<section class="card" data-testid={`trend-${t.metric}`}>
			<p class="section-label">{t.label} <span class="hint">· {t.days > 365 ? 'last 2 years' : '7-day avg · 90 days'}</span></p>
			<LineChart points={series(t.metric, t.days)} format={t.format} />
		</section>
	{/each}

	<section class="card">
		<p class="section-label">Weekly scores</p>
		<ul class="week-list">
			{#each scores as s (s.week_end)}
				<li>
					<a href={`/week/${s.week_end}`}>
						<span>{shortDate(s.week_end)}</span>
						<span class="num">{s.score.toFixed(1)}</span>
						<span class="num" class:pos={s.change > 0} class:neg={s.change < 0}>{signed(s.change)}</span>
					</a>
				</li>
			{/each}
		</ul>
	</section>
{/if}
