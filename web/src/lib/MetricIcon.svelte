<script lang="ts" module>
	type Category = 'heart' | 'body' | 'sleep' | 'activity' | 'mind' | 'nutrition';
	type Glyph = 'heart' | 'pulse' | 'gauge' | 'wind' | 'stopwatch' | 'ruler' | 'moon' | 'steps' | 'flame' | 'dumbbell' | 'flower' | 'apple';

	// Each metric gets its own glyph, tinted by the health category it belongs to.
	const icons: Record<string, { glyph: Glyph; category: Category }> = {
		rhr: { glyph: 'heart', category: 'heart' },
		hrv: { glyph: 'pulse', category: 'heart' },
		hrv_sdnn: { glyph: 'pulse', category: 'heart' },
		bp: { glyph: 'gauge', category: 'heart' },
		vo2_max: { glyph: 'wind', category: 'heart' },
		cardio_recovery: { glyph: 'stopwatch', category: 'heart' },
		whtr: { glyph: 'ruler', category: 'body' },
		waist_cm: { glyph: 'ruler', category: 'body' },
		sleep_score: { glyph: 'moon', category: 'sleep' },
		sleep_hours: { glyph: 'moon', category: 'sleep' },
		steps: { glyph: 'steps', category: 'activity' },
		exercise_min: { glyph: 'flame', category: 'activity' },
		working_sets: { glyph: 'dumbbell', category: 'activity' },
		mindfulness: { glyph: 'flower', category: 'mind' },
		nutrition: { glyph: 'apple', category: 'nutrition' }
	};
</script>

<script lang="ts">
	let { metric, size = 30 }: { metric: string; size?: number } = $props();

	const icon = $derived(icons[metric]);
</script>

{#if icon}
	<span class="icon" data-category={icon.category} style:--size={`${size}px`} aria-hidden="true">
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
			{#if icon.glyph === 'heart'}
				<path d="M12 20s-8-4.6-8-10.2A4.3 4.3 0 0 1 12 7.4a4.3 4.3 0 0 1 8 2.4C20 15.4 12 20 12 20Z" fill="currentColor" />
			{:else if icon.glyph === 'pulse'}
				<path d="M2 12h4l2.5-6 4 12 2.5-6h7" />
			{:else if icon.glyph === 'gauge'}
				<path d="M3.5 17a8.5 8.5 0 1 1 17 0" />
				<path d="m12 17 4-5" />
				<circle cx="12" cy="17" r="1" fill="currentColor" />
			{:else if icon.glyph === 'wind'}
				<path d="M3 8h9.5a2.5 2.5 0 1 0-2.5-2.5" />
				<path d="M3 12h15.5a2.5 2.5 0 1 1-2.5 2.5" />
				<path d="M3 16h7" />
			{:else if icon.glyph === 'stopwatch'}
				<circle cx="12" cy="14" r="7" />
				<path d="M12 14V10.5" />
				<path d="M10 3h4" />
				<path d="m18 7 1.5-1.5" />
			{:else if icon.glyph === 'ruler'}
				<rect x="2.5" y="8" width="19" height="8" rx="1.5" />
				<path d="M6.5 8v3M10.5 8v4M14.5 8v3M18.5 8v4" />
			{:else if icon.glyph === 'moon'}
				<path d="M20 14.5A8.5 8.5 0 1 1 9.5 4 6.8 6.8 0 0 0 20 14.5Z" fill="currentColor" />
			{:else if icon.glyph === 'steps'}
				<g fill="currentColor" stroke="none">
					<ellipse cx="8" cy="7.5" rx="3" ry="4.5" />
					<circle cx="8" cy="15.5" r="2" />
					<ellipse cx="16" cy="10.5" rx="3" ry="4.5" />
					<circle cx="16" cy="18.5" r="2" />
				</g>
			{:else if icon.glyph === 'flame'}
				<path d="M12 22a7 7 0 0 0 7-7c0-4.5-3.5-7-5-11-1 2.5-2 3.5-3.5 4C10 5.5 9.5 4 8 2.5 8 7 5 9 5 15a7 7 0 0 0 7 7Z" fill="currentColor" />
			{:else if icon.glyph === 'dumbbell'}
				<path d="M6.5 6.5v11M17.5 6.5v11M3 9.5v5M21 9.5v5M6.5 12h11" stroke-width="2.4" />
			{:else if icon.glyph === 'flower'}
				<g fill="currentColor" stroke="none" opacity="0.55">
					<circle cx="12" cy="7" r="3.6" />
					<circle cx="17" cy="12" r="3.6" />
					<circle cx="12" cy="17" r="3.6" />
					<circle cx="7" cy="12" r="3.6" />
				</g>
				<circle cx="12" cy="12" r="2.6" fill="currentColor" stroke="none" />
			{:else if icon.glyph === 'apple'}
				<path d="M12 7.5c-1.5-1-3-1.5-4.5-1C5 7.3 4 9.7 4 12.5 4 17 7 21 9 21c1.2 0 1.8-.6 3-.6s1.8.6 3 .6c2 0 5-4 5-8.5 0-2.8-1-5.2-3.5-6-1.5-.5-3 0-4.5 1Z" fill="currentColor" />
				<path d="M12 7.5c0-2 1-3.8 3-4.5" />
			{/if}
		</svg>
	</span>
{/if}

<style>
	.icon {
		--tint: var(--muted);
		flex: none;
		display: inline-grid;
		place-items: center;
		width: var(--size);
		height: var(--size);
		border-radius: calc(var(--size) * 0.28);
		color: var(--tint);
		background: color-mix(in srgb, var(--tint) 18%, transparent);
	}
	.icon svg {
		width: 60%;
		height: 60%;
	}
	.icon[data-category='heart'] {
		--tint: #ff375f;
	}
	.icon[data-category='body'] {
		--tint: #af52de;
	}
	.icon[data-category='sleep'] {
		--tint: #32ade6;
	}
	.icon[data-category='activity'] {
		--tint: #ff9500;
	}
	.icon[data-category='mind'] {
		--tint: #00b8a9;
	}
	.icon[data-category='nutrition'] {
		--tint: #34c759;
	}
</style>
