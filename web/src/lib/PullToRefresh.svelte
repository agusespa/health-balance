<script lang="ts">
	import { onMount } from 'svelte';

	/**
	 * Pull down from the top of the page to fetch its data again without reloading, so newly
	 * delivered health data shows up in the home-screen app, which has no reload button. Only in
	 * the home-screen app: browser tabs have their own gesture, and two would refresh twice.
	 */

	let { onrefresh }: { onrefresh: () => Promise<unknown> } = $props();

	/** How far (px) the indicator has to come down before letting go refreshes. */
	const THRESHOLD = 72;
	const MAX = 110;
	/** Finger travel (px) ignored before a pull starts, so taps and small jitters don't count. */
	const SLOP = 6;
	/**
	 * The spinner stays at least this long (ms). The data usually comes back almost at once, and a
	 * spinner that only flashes looks like nothing happened.
	 */
	const MIN_SPIN = 700;

	let pull = $state(0);
	let dragging = $state(false);
	let refreshing = $state(false);
	const ready = $derived(pull >= THRESHOLD);

	onMount(() => {
		const standalone =
			(navigator as Navigator & { standalone?: boolean }).standalone === true ||
			matchMedia('(display-mode: standalone)').matches;
		if (!standalone) return;

		let startY: number | null = null;
		let startX = 0;

		const reset = () => {
			startY = null;
			dragging = false;
			pull = 0;
		};

		const start = (e: TouchEvent) => {
			const atTop = scrollY <= 0;
			const busy = refreshing || e.touches.length !== 1;
			startY = atTop && !busy ? e.touches[0].clientY : null;
			startX = e.touches[0].clientX;
		};

		const move = (e: TouchEvent) => {
			if (startY === null) return;
			const dy = e.touches[0].clientY - startY;
			if (!dragging) {
				// Moving up, or the page already scrolled: this is a normal scroll.
				if (dy < 0 || scrollY > 0) return void (startY = null);
				// Mostly sideways: a horizontal scroll, not a pull.
				if (Math.abs(e.touches[0].clientX - startX) > dy) return void (startY = null);
				if (dy < SLOP) return;
				dragging = true;
			}
			// Stops the iOS rubber band so the page stays put while the indicator comes down.
			e.preventDefault();
			pull = Math.min(MAX, Math.max(0, dy - SLOP) * 0.5);
		};

		const end = async () => {
			if (!dragging) return void (startY = null);
			if (pull < THRESHOLD) return reset();
			startY = null;
			dragging = false;
			pull = THRESHOLD;
			refreshing = true;
			try {
				await Promise.all([onrefresh(), new Promise((r) => setTimeout(r, MIN_SPIN))]);
			} finally {
				refreshing = false;
				reset();
			}
		};

		addEventListener('touchstart', start, { passive: true });
		addEventListener('touchmove', move, { passive: false });
		addEventListener('touchend', end);
		addEventListener('touchcancel', reset);
		return () => {
			removeEventListener('touchstart', start);
			removeEventListener('touchmove', move);
			removeEventListener('touchend', end);
			removeEventListener('touchcancel', reset);
		};
	});
</script>

<div
	class="ptr"
	class:dragging
	class:ready
	class:refreshing
	class:idle={pull === 0 && !refreshing}
	style:--pull="{pull}px"
	style:--progress={Math.min(pull / THRESHOLD, 1)}
	data-testid="pull-to-refresh"
	aria-hidden="true"
>
	<svg viewBox="0 0 24 24">
		<path d="M20 12a8 8 0 1 1-2.34-5.66" />
		<path d="M20 4.5v4h-4" />
	</svg>
</div>

<style>
	.ptr {
		position: fixed;
		top: env(safe-area-inset-top);
		left: 50%;
		/* Above the sticky top bar. */
		z-index: 30;
		display: grid;
		place-items: center;
		width: 40px;
		height: 40px;
		border-radius: 999px;
		background: var(--surface);
		border: 1px solid var(--border-strong);
		box-shadow: var(--shadow);
		color: var(--muted);
		opacity: var(--progress);
		transform: translate(-50%, calc(var(--pull) - 44px));
		transition:
			transform 0.3s cubic-bezier(0.3, 1.3, 0.5, 1),
			opacity 0.2s;
		pointer-events: none;
	}

	/* Not painted at rest; hidden only once it has finished easing back up. */
	.ptr.idle {
		visibility: hidden;
		transition:
			transform 0.3s cubic-bezier(0.3, 1.3, 0.5, 1),
			opacity 0.2s,
			visibility 0s 0.3s;
	}

	/* Follow the finger exactly while pulling; ease back only on release. */
	.ptr.dragging {
		transition: none;
	}

	.ptr.ready,
	.ptr.refreshing {
		color: var(--text);
	}

	svg {
		width: 22px;
		height: 22px;
		fill: none;
		stroke: currentColor;
		stroke-width: 2.2;
		stroke-linecap: round;
		stroke-linejoin: round;
		transform: rotate(calc(var(--progress) * 270deg));
	}

	.refreshing svg {
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(1turn);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.ptr {
			transition: none;
		}

		.refreshing svg {
			animation: none;
		}
	}
</style>
