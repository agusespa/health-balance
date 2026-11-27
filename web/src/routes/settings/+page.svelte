<script lang="ts">
	import { onMount } from 'svelte';
	import { version } from '$app/env';
	import { api, type Profile } from '#lib/api.ts';
	import TopBar from '#lib/TopBar.svelte';
	import { getTheme, setTheme, type ThemePref } from '#lib/theme.ts';

	let profile = $state<Profile>({ birth_date: '', sex: '', height_cm: 0 });
	let profileMsg = $state('');
	let profileError = $state('');
	let saving = $state(false);

	let theme = $state<ThemePref>('system');
	const themes: { value: ThemePref; label: string }[] = [
		{ value: 'system', label: 'System' },
		{ value: 'light', label: 'Light' },
		{ value: 'dark', label: 'Dark' }
	];

	function chooseTheme(pref: ThemePref) {
		theme = pref;
		setTheme(pref);
	}

	// SvelteKit's build version is the build's timestamp; it tells two builds of one release apart.
	const builtMs = Number(version);
	const builtAt = Number.isFinite(builtMs)
		? new Date(builtMs).toLocaleString(undefined, {
				day: 'numeric',
				month: 'short',
				year: 'numeric',
				hour: 'numeric',
				minute: '2-digit'
			})
		: null;

	onMount(async () => {
		theme = getTheme();
		try {
			profile = await api.profile();
		} catch (e) {
			profileError = (e as Error).message;
		}
	});

	async function saveProfile(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		profileMsg = profileError = '';
		try {
			profile = await api.saveProfile({ ...profile, height_cm: Number(profile.height_cm) });
			profileMsg = 'Saved';
		} catch (err) {
			profileError = (err as Error).message;
		} finally {
			saving = false;
		}
	}
</script>

<TopBar title="Settings" back="/profile" />

<form class="card" onsubmit={saveProfile} data-testid="profile-form">
	<h2>Profile</h2>
	<p class="muted lede">Used for the aging tax, the VO2 max baseline and waist-to-height.</p>
	<label for="birth">Birth date</label>
	<input id="birth" type="date" bind:value={profile.birth_date} required />
	<label for="sex">Sex</label>
	<select id="sex" bind:value={profile.sex} required>
		<option value="" disabled>Choose…</option>
		<option value="male">Male</option>
		<option value="female">Female</option>
	</select>
	<label for="height">Height (cm)</label>
	<input id="height" type="number" min="100" max="250" step="0.1" bind:value={profile.height_cm} required />
	<div class="actions">
		<button class="primary" disabled={saving}>Save profile</button>
		{#if profileMsg}<span class="pos" data-testid="profile-saved">{profileMsg}</span>{/if}
	</div>
	{#if profileError}<p class="error">{profileError}</p>{/if}
</form>

<section class="card" data-testid="appearance">
	<h2>Appearance</h2>
	<p class="muted lede">System follows your device's light or dark setting.</p>
	<div class="segmented" role="radiogroup" aria-label="Appearance">
		{#each themes as t (t.value)}
			<button type="button" role="radio" aria-checked={theme === t.value} class:active={theme === t.value} onclick={() => chooseTheme(t.value)}>
				{t.label}
			</button>
		{/each}
	</div>
</section>

<p class="muted version" data-testid="app-version">
	Version {__APP_VERSION__}{#if builtAt}<br />Built {builtAt}{/if}
</p>

<style>
	.lede {
		font-size: 0.88rem;
		line-height: 1.45;
		margin: 0.35rem 0 0.4rem;
	}
	.version {
		font-size: 0.8rem;
		text-align: center;
		margin: 2.5rem 0 0;
		font-variant-numeric: tabular-nums;
	}
	.actions {
		display: flex;
		align-items: center;
		gap: 1rem;
		margin-top: 1.4rem;
	}
</style>
