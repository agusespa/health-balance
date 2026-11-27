// See https://svelte.dev/docs/kit/types#app.d.ts
declare global {
	// The release, set at build time from APP_VERSION.
	const __APP_VERSION__: string;

	namespace App {}
}

export {};
