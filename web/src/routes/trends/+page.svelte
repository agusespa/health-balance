<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type DailyMetric } from '#lib/api.ts';
	import LineChart from '#lib/LineChart.svelte';
	import MetricIcon from '#lib/MetricIcon.svelte';
	import TopBar from '#lib/TopBar.svelte';
	import { daysAgo, isoDate, shortDate } from '#lib/format.ts';

	let daily = $state<DailyMetric[] | null>(null);
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
			daily = await api.daily(daysAgo(730), isoDate(new Date()), trends.map((t) => t.metric));
		} catch (e) {
			error = (e as Error).message;
		}
	});

	/** Daily values; the 90-day ones as a 7-day rolling average, which reads better than raw day-to-day noise. */
	function series(metric: string, days: number) {
		const from = daysAgo(days);
		const window = days <= 90 ? 7 : 1;
		// Include the days before the window so its first points average a full week too.
		const rows = (daily ?? []).filter((d) => d.metric === metric && d.date >= daysAgo(days + window - 1));
		return rows
			.map((d, i) => {
				const slice = rows.slice(Math.max(0, i - window + 1), i + 1);
				return { date: d.date, x: shortDate(d.date), y: slice.reduce((sum, r) => sum + r.value, 0) / slice.length };
			})
			.filter((p) => p.date >= from);
	}
</script>

<TopBar title="Trends" />

{#if error}
	<p class="error">{error}</p>
{:else if !daily}
	<p class="muted">Loading…</p>
{:else}
	{#each trends as t (t.metric)}
		<section class="card" data-testid={`trend-${t.metric}`}>
			<p class="section-label"><MetricIcon metric={t.metric} size={22} /> {t.label} <span class="hint">· {t.days > 365 ? 'last 2 years' : '7-day avg · 90 days'}</span></p>
			<LineChart points={series(t.metric, t.days)} format={t.format} />
		</section>
	{/each}
{/if}
