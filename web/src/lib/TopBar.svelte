<script lang="ts">
	import type { Snippet } from 'svelte';

	/**
	 * The coloured, sticky top of every page. It reaches up under the iOS status
	 * bar (which blurs whatever is behind it) and keeps its controls a safe
	 * distance below it. On the home page a `.band-hero` continues the colour
	 * underneath and scrolls away beneath this bar; past `threshold` the `lead`
	 * content (e.g. the week) gives way to a compact summary (`compact`).
	 */
	let {
		title = '',
		back = '/',
		home = false,
		threshold = 220,
		lead,
		compact,
		actions
	}: {
		title?: string;
		back?: string;
		home?: boolean;
		threshold?: number;
		lead?: Snippet;
		compact?: Snippet;
		actions?: Snippet;
	} = $props();

	let y = $state(0);
	const raised = $derived(y > 4);
	const condensed = $derived(y > threshold);
</script>

<svelte:window bind:scrollY={y} />

<header class="appbar band" class:raised class:sub={!home} class:home>
	<div class="inner">
		<div class="lead">
			{#if home}
				<div class="swap">
					<div class="layer" class:out={condensed} aria-hidden={condensed}>{@render lead?.()}</div>
					<div class="layer" class:out={!condensed} aria-hidden={!condensed}>{@render compact?.()}</div>
				</div>
			{:else}
				<a class="back" href={back}>
					<svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true">
						<path d="M15 5l-7 7 7 7" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
					</svg>
					Back
				</a>
			{/if}
		</div>
		{#if !home}
			<h1 class="mid">{title}</h1>
		{/if}
		<div class="trail">{@render actions?.()}</div>
	</div>
</header>

<style>
	.appbar {
		position: sticky;
		top: 0;
		z-index: 20;
		/* Full-bleed even though it lives inside the centred column. */
		width: 100vw;
		margin-left: calc(50% - 50vw);
		/* Solid colour under the status bar, controls well below it: iOS blurs
		   the top edge of the screen, so nothing interactive sits there. */
		padding-top: calc(env(safe-area-inset-top) + var(--top-gap));
		padding-bottom: 0.4rem;
		background: var(--header-bg);
		color: var(--text);
		transition: box-shadow 0.2s;
	}
	.appbar.sub {
		margin-bottom: 1rem;
	}
	.appbar.raised {
		box-shadow: 0 6px 18px rgba(0, 0, 0, 0.16);
	}
	.inner {
		max-width: 560px;
		height: var(--bar-h);
		margin: 0 auto;
		padding: 0 0.9rem;
		display: grid;
		grid-template-columns: 1fr auto 1fr;
		align-items: center;
		gap: 0.5rem;
	}
	.home .inner {
		grid-template-columns: 1fr auto;
	}
	.lead {
		justify-self: start;
		min-width: 0;
	}
	.swap {
		display: grid;
		align-items: center;
	}
	.layer {
		grid-area: 1 / 1;
		transition:
			opacity 0.2s,
			transform 0.2s;
	}
	.layer.out {
		opacity: 0;
		transform: translateY(6px);
		pointer-events: none;
	}
	.trail {
		justify-self: end;
		display: flex;
		gap: 0.4rem;
	}
	.mid {
		margin: 0;
		font-family: var(--font-sans);
		font-weight: 600;
		font-size: 1rem;
		letter-spacing: -0.01em;
		white-space: nowrap;
	}
	.back {
		display: inline-flex;
		align-items: center;
		gap: 0.1rem;
		margin-left: -0.35rem;
		padding: 0.4rem;
		font-size: 0.98rem;
		color: var(--text);
	}
</style>
