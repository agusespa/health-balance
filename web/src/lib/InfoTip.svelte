<script lang="ts">
	// An ⓘ button beside a section title; it opens a short hint in place.
	let { label, text, id }: { label: string; text: string; id: string } = $props();

	let open = $state(false);
	let wrap: HTMLElement | undefined = $state();

	function closeOutside(e: PointerEvent) {
		if (open && wrap && !wrap.contains(e.target as Node)) open = false;
	}
</script>

<svelte:window onpointerdown={closeOutside} onkeydown={(e) => e.key === 'Escape' && (open = false)} />

<span class="info-wrap" bind:this={wrap}>
	<button
		type="button"
		class="info"
		aria-label={label}
		aria-expanded={open}
		aria-describedby={open ? id : undefined}
		onclick={() => (open = !open)}
	>
		<svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
			<circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="1.8" />
			<path d="M12 11v5.5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
			<circle cx="12" cy="7.8" r="1.1" fill="currentColor" />
		</svg>
	</button>
	{#if open}
		<span class="tip" role="tooltip" {id}>{text}</span>
	{/if}
</span>

<style>
	.info-wrap {
		position: relative;
		display: inline-flex;
	}
	/* Small visible icon, larger tap target. */
	.info {
		appearance: none;
		border: none;
		padding: 0;
		background: none;
		cursor: pointer;
		display: grid;
		place-items: center;
		width: 32px;
		height: 32px;
		margin: -8px 0;
		border-radius: 999px;
		color: var(--muted);
	}
	.tip {
		position: absolute;
		top: calc(100% + 6px);
		left: -10px;
		z-index: 5;
		width: max-content;
		max-width: min(260px, 70vw);
		padding: 0.55rem 0.7rem;
		border: 1px solid var(--border-strong);
		border-radius: 10px;
		background: var(--surface);
		box-shadow: var(--shadow);
		font-family: var(--font-sans);
		font-size: 0.8rem;
		line-height: 1.45;
		letter-spacing: normal;
		text-transform: none;
		color: var(--text-secondary);
	}
	.info:hover,
	.info[aria-expanded='true'] {
		color: var(--text-secondary);
		background: var(--surface-muted);
	}
</style>
