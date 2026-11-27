export function signed(n: number, digits = 1): string {
	const v = n.toFixed(digits);
	if (Number(v) === 0) return (0).toFixed(digits);
	return n > 0 ? `+${v}` : `−${v.slice(1)}`;
}

/** "6 h ago", "3 days ago". */
export function ago(iso: string, now = Date.now()): string {
	const mins = Math.max(0, Math.round((now - new Date(iso).getTime()) / 60000));
	if (mins < 60) return `${mins} min ago`;
	const hours = Math.round(mins / 60);
	if (hours < 48) return `${hours} h ago`;
	return `${Math.round(hours / 24)} days ago`;
}

/** "Nov 21" or "Nov 21, 2025" outside the current year. */
export function shortDate(date: string): string {
	const d = new Date(`${date}T12:00:00`);
	const sameYear = d.getFullYear() === new Date().getFullYear();
	return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: sameYear ? undefined : 'numeric' });
}

export function isoDate(d: Date): string {
	const y = d.getFullYear();
	const m = String(d.getMonth() + 1).padStart(2, '0');
	const day = String(d.getDate()).padStart(2, '0');
	return `${y}-${m}-${day}`;
}

export function daysAgo(n: number): string {
	const d = new Date();
	d.setDate(d.getDate() - n);
	return isoDate(d);
}
