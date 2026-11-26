// Appearance preference. The inline script in app.html applies it before first
// paint (and follows the system setting live); this module reads and changes it.

export type ThemePref = 'system' | 'light' | 'dark';

const KEY = 'theme';

declare global {
	interface Window {
		__applyTheme?: () => void;
	}
}

export function getTheme(): ThemePref {
	const v = localStorage.getItem(KEY);
	return v === 'light' || v === 'dark' ? v : 'system';
}

export function setTheme(pref: ThemePref): void {
	if (pref === 'system') localStorage.removeItem(KEY);
	else localStorage.setItem(KEY, pref);
	window.__applyTheme?.();
}
