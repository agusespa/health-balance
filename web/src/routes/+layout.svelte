<script lang="ts">
	import '@fontsource-variable/geist';
	import '@fontsource-variable/geist-mono';
	import '../app.css';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
</script>

<main>
	{@render children()}
</main>

<style>
	main {
		max-width: 560px;
		margin: 0 auto;
		/* The sticky TopBar on every page carries the top safe-area inset. */
		padding: 0 0.9rem calc(env(safe-area-inset-bottom) + 2.5rem);
	}
</style>
