<script lang="ts">
	import type { Nutrition } from '#lib/api.ts';

	let { nutrition: n }: { nutrition: Nutrition } = $props();

	const kcal = (v: number) => Math.round(v).toLocaleString('en-US');
	const balanceText = $derived(n.balance_kcal > 0 ? `+${kcal(n.balance_kcal)}` : n.balance_kcal < 0 ? `−${kcal(-n.balance_kcal)}` : '0');
	const balanceWord = $derived(n.balance_kcal > 0 ? 'surplus' : n.balance_kcal < 0 ? 'deficit' : 'even');

	// Share of the logged energy from each macro (4 / 4 / 9 kcal per gram).
	const macroKcal = $derived(n.protein_g * 4 + n.carbs_g * 4 + n.fat_g * 9);
	const pct = (kcalFromMacro: number) => (macroKcal > 0 ? Math.round((kcalFromMacro / macroKcal) * 100) : 0);
	const macros = $derived([
		{ key: 'protein', label: 'Protein', grams: n.protein_g, pct: pct(n.protein_g * 4), extra: n.protein_per_kg > 0 ? `${n.protein_per_kg.toFixed(1)} g/kg` : '' },
		{ key: 'carbs', label: 'Carbs', grams: n.carbs_g, pct: pct(n.carbs_g * 4), extra: '' },
		{ key: 'fat', label: 'Fat', grams: n.fat_g, pct: pct(n.fat_g * 9), extra: '' }
	]);
</script>

<section class="card" data-testid="card-nutrition">
	<div class="head">
		<p class="section-label">Nutrition <span class="hint">· last {n.days} days</span></p>
		<span class="logged" data-testid="nutrition-logged">{n.logged_days}/{n.days} logged</span>
	</div>

	{#if n.logged_days === 0}
		<p class="muted empty">No full days of food logged.</p>
	{:else}
		<div class="balance" data-testid="nutrition-balance">
			{#if n.balance_days > 0}
				<p class="stat num">
					{balanceText} <span class="unit">kcal/day</span>
				</p>
				<p class="note">{balanceWord} · {kcal(n.energy_in_kcal)} eaten vs {kcal(n.burned_kcal)} burned</p>
			{:else}
				<p class="stat num">{kcal(n.energy_in_kcal)} <span class="unit">kcal/day</span></p>
				<p class="note">eaten · no energy burned data for these days</p>
			{/if}
		</div>

		<div class="bar" aria-hidden="true">
			{#each macros as m (m.key)}
				<span class="seg" data-macro={m.key} style:width={`${m.pct}%`}></span>
			{/each}
		</div>

		{#each macros as m (m.key)}
			<div class="row" data-testid={`nutrition-${m.key}`}>
				<span class="label"><span class="dot" data-macro={m.key}></span>{m.label}</span>
				<span class="sub">{m.pct}%{m.extra ? ` · ${m.extra}` : ''}</span>
				<span class="value num">{Math.round(m.grams)} g</span>
			</div>
		{/each}
		<div class="row" data-testid="nutrition-fiber">
			<span class="label">Fiber</span>
			<span class="sub">part of carbs</span>
			<span class="value num">{Math.round(n.fiber_g)} g</span>
		</div>
	{/if}

	{#if n.water_days > 0}
		<div class="row" data-testid="nutrition-water">
			<span class="label">Water</span>
			<span class="sub">{n.water_days} days</span>
			<span class="value num">{(n.water_ml / 1000).toFixed(1)} L</span>
		</div>
	{/if}

	<p class="foot">Daily averages over days with at least 1,000 kcal logged. Not part of the score.</p>
</section>

<style>
	.head {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		margin-bottom: 0.5rem;
	}
	.head .section-label {
		margin: 0;
	}
	.logged {
		font-size: 0.75rem;
		color: var(--muted);
	}
	.empty {
		margin: 0.4rem 0;
		font-size: 0.88rem;
	}
	.balance {
		margin-bottom: 0.8rem;
	}
	.stat {
		margin: 0;
		font-size: 1.45rem;
		font-weight: 600;
		letter-spacing: -0.03em;
		line-height: 1.1;
	}
	.foot {
		margin: 0.5rem 0 0;
		font-size: 0.72rem;
		color: var(--muted);
	}
	.unit {
		font-size: 0.8rem;
		font-weight: 500;
		letter-spacing: 0;
		color: var(--muted);
	}
	.note {
		margin: 0.2rem 0 0;
		font-size: 0.75rem;
		color: var(--muted);
	}
	.bar {
		display: flex;
		height: 6px;
		border-radius: 999px;
		overflow: hidden;
		background: var(--surface-muted);
		gap: 2px;
		margin-bottom: 0.25rem;
	}
	.row {
		display: grid;
		grid-template-columns: 1fr auto 4rem;
		align-items: baseline;
		gap: 0.9rem;
		padding: 0.6rem 0;
		border-top: 1px solid var(--border);
	}
	.bar + .row {
		border-top: none;
	}
	.label {
		font-size: 0.95rem;
	}
	.sub {
		color: var(--muted);
		font-size: 0.78rem;
		font-variant-numeric: tabular-nums;
	}
	.value {
		text-align: right;
		font-size: 0.88rem;
		font-weight: 600;
		color: var(--text-secondary);
	}
	.dot {
		display: inline-block;
		width: 8px;
		height: 8px;
		border-radius: 50%;
		margin-right: 0.45rem;
		vertical-align: 1px;
	}
	[data-macro='protein'] {
		background: var(--positive-fill);
	}
	[data-macro='carbs'] {
		background: #e0b44a;
	}
	[data-macro='fat'] {
		background: #6b9bd8;
	}
</style>
