<script lang="ts">
	import type { Snippet } from 'svelte';

	let { title, onopen, children }: { title: string; onopen?: () => void; children: Snippet } = $props();
	let open = $state(false);

	function toggle() {
		open = !open;
		if (open) onopen?.();
	}
</script>

<section class="card" class:open>
	<button class="toggle" onclick={toggle} aria-expanded={open}>
		<h2>{title}</h2>
		<svg class="icon" viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
			<path d="M6 9l6 6 6-6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
		</svg>
	</button>
	{#if open}
		<div class="content">{@render children()}</div>
	{/if}
</section>

<style>
	.toggle {
		display: flex;
		justify-content: space-between;
		align-items: center;
		width: 100%;
		background: none;
		border: none;
		border-radius: 0;
		padding: 0;
		text-align: left;
	}
	.icon {
		color: var(--muted);
		transition: transform 0.2s;
	}
	.open .icon {
		transform: rotate(180deg);
	}
	.content {
		margin-top: 0.9rem;
		padding-top: 0.2rem;
	}
</style>
