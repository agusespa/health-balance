<script lang="ts">
	import '@fontsource-variable/geist';
	import '@fontsource-variable/geist-mono';
	import '../app.css';
	import { replaceState } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { sessionExpired } from '#lib/api.ts';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	onMount(() => {
		// Back from the Access login: drop the marker that let the launch skip the cached shell.
		if (location.search === '?signin') replaceState(location.pathname, {});
	});
</script>

<main>
	{@render children()}
</main>

{#if $sessionExpired}
	<p class="expired" role="alert">
		Your session expired. <a href={`${page.url.pathname}?signin`} data-sveltekit-reload>Sign in again</a>
	</p>
{/if}

<style>
	main {
		max-width: 560px;
		margin: 0 auto;
		/* The sticky TopBar on every page carries the top safe-area inset. */
		padding: 0 0.9rem calc(env(safe-area-inset-bottom) + 2.5rem);
	}

	.expired {
		position: fixed;
		left: 50%;
		bottom: calc(env(safe-area-inset-bottom) + 1rem);
		transform: translateX(-50%);
		width: max-content;
		max-width: calc(100% - 2rem);
		margin: 0;
		padding: 0.7rem 1rem;
		border-radius: var(--radius);
		background: var(--surface);
		border: 1px solid var(--border-strong);
		box-shadow: var(--shadow);
		color: var(--text);
		font-size: 0.9rem;
	}

	.expired a {
		color: var(--link);
		font-weight: 600;
	}
</style>
