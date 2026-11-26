<script lang="ts">
	import TopBar from '#lib/TopBar.svelte';
</script>

<TopBar title="How it works" />

<section class="card">
	<h2>Biological Reserve Score</h2>
	<p>
		A slow-moving estimate of your long-term health reserve, updated once a week (Saturday to Friday) from your health
		data and gym log. Nothing is entered by hand. History starts at 1000 in the first week of 2024.
	</p>
</section>

<section class="card">
	<h2>Each week</h2>
	<ol>
		<li><strong>Aging tax.</strong> The score loses (age² / 8000) / 52 of itself, the natural decline with age.</li>
		<li><strong>Target.</strong> 1000 plus the points of every metric below.</li>
		<li><strong>Slow adjustment.</strong> The score moves 12% of the way toward the target, so one good or bad week barely moves it.</li>
	</ol>
	<p class="muted">Points are capped, and falling short costs more than doing well earns.</p>
</section>

<section class="card">
	<h2>Markers</h2>
	<p class="muted">Slow-changing measures. When a week has none, the last value is reused for a while, then fades toward neutral.</p>
	<ul>
		<li><strong>VO2 max</strong> vs the typical value for your age and sex (up to +16 / −22).</li>
		<li><strong>Waist / height</strong> vs 0.48 (up to +10 / −15).</li>
		<li><strong>Resting heart rate</strong> vs your own 90-day average (up to +7 / −10).</li>
		<li><strong>HRV</strong> vs your own 60-day average, in percent (up to +5 / −8).</li>
		<li><strong>Blood pressure</strong> vs 120/80 (up to +9 / −14); measured a few mornings each quarter and kept for 16 weeks.</li>
		<li><strong>Cardio recovery</strong>, the heart-rate drop one minute after a workout, vs 25 bpm (up to +4 / −6).</li>
	</ul>
</section>

<section class="card">
	<h2>Behaviours</h2>
	<p class="muted">Habits, scored on a 4-week average so consistency counts more than one week.</p>
	<ul>
		<li><strong>Sleep score</strong> (0–100) vs 85: duration (50 points, full from 8 h), bedtime consistency (30) and interruptions (20).</li>
		<li><strong>Steps</strong> vs 8,000 a day.</li>
		<li><strong>Exercise minutes</strong> vs 150 a week.</li>
		<li><strong>Working sets</strong> from the gym log vs 10 a week.</li>
		<li><strong>Mindfulness sessions</strong> vs 3 a week.</li>
	</ul>
</section>

<section class="card">
	<h2>Nutrition</h2>
	<p class="muted">Shown on the home page but not part of the score.</p>
	<p>
		Daily averages over the last 7 full days: energy eaten, protein, carbs, fat, fiber and water from your food diary, and
		the balance against active plus resting energy burned from the Watch. A day counts once it has 1,000 kcal logged,
		so an unfinished diary doesn't show up as a big deficit. Water counts on any day it was logged.
	</p>
</section>

<section class="card">
	<h2>Missing data</h2>
	<p>
		A daily measure needs at least 3 days in a week to count. Labels on each metric show when a value is the
		<em>last value</em> reused, <em>fading</em> toward neutral, or has <em>no data</em> yet (0 points).
	</p>
</section>

<section class="card">
	<h3>Disclaimer</h3>
	<p class="muted">A personal tracking tool, not medical advice.</p>
</section>

<style>
	p,
	li {
		line-height: 1.6;
		color: var(--text-secondary);
		font-size: 0.94rem;
	}
	h2 {
		margin-bottom: 0.2rem;
	}
	strong {
		color: var(--text);
		font-weight: 600;
	}
	ul,
	ol {
		padding-left: 1.15rem;
		margin: 0.6rem 0 0;
	}
	li + li {
		margin-top: 0.35rem;
	}
	li::marker {
		color: var(--positive);
		font-family: var(--font-mono);
		font-size: 0.85em;
	}
	h3 {
		margin: 0;
		font-family: var(--font-mono);
		font-size: 0.68rem;
		font-weight: 400;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--muted);
	}
	.card:last-of-type p {
		margin-bottom: 0;
	}
</style>
