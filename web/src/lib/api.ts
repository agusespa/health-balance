// Types and calls for the Go JSON API under /api.

export type Kind = 'marker' | 'behaviour';
export type Status = 'measured' | 'carried' | 'imputed' | 'missing';

export interface MetricScore {
	metric: string;
	label: string;
	kind: Kind;
	status: Status;
	value: number;
	reference: number;
	value_text: string;
	reference_text: string;
	points: number;
}

export interface WeeklyScore {
	week_end: string;
	score: number;
	change: number;
	aging_tax: number;
	target: number;
	metrics: MetricScore[];
}

export type ScoreSummary = Omit<WeeklyScore, 'metrics'>;

export interface IngestRun {
	id: number;
	received_at: string;
	kind: string;
	rows: number;
	accepted: number;
	rejected: number;
	from_date: string;
	to_date: string;
}

/** Daily averages over the logged days of the last 7 full days. Not scored. */
export interface Nutrition {
	from: string;
	to: string;
	days: number;
	logged_days: number;
	energy_in_kcal: number;
	protein_g: number;
	carbs_g: number;
	fat_g: number;
	fiber_g: number;
	protein_per_kg: number;
	balance_days: number;
	burned_kcal: number;
	balance_kcal: number;
	water_days: number;
	water_ml: number;
}

export interface Dashboard {
	profile_complete: boolean;
	week: { end: string; range: string };
	score: WeeklyScore | null;
	last_sync: IngestRun | null;
	nutrition: Nutrition | null;
}

export interface DailyMetric {
	date: string;
	metric: string;
	value: number;
	samples: number;
	source: string;
}

export interface Profile {
	birth_date: string;
	sex: string;
	height_cm: number;
}

export interface Insight {
	week_end: string;
	model: string;
	created_at: string;
	html: string;
}

export class ApiError extends Error {
	constructor(
		readonly status: number,
		message: string
	) {
		super(message);
	}
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
	const res = await fetch(path, {
		method,
		headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
		body: body === undefined ? undefined : JSON.stringify(body)
	});
	const text = await res.text();
	const data = text ? JSON.parse(text) : null;
	if (!res.ok) throw new ApiError(res.status, data?.error ?? res.statusText);
	return data as T;
}

export const api = {
	dashboard: () => request<Dashboard>('GET', '/api/dashboard'),
	scores: () => request<ScoreSummary[]>('GET', '/api/scores'),
	week: (date: string) => request<WeeklyScore>('GET', `/api/weeks/${date}`),
	daily: (from: string, to: string, metrics: string[]) =>
		request<DailyMetric[]>('GET', `/api/daily?from=${from}&to=${to}&metrics=${metrics.join(',')}`),
	profile: () => request<Profile>('GET', '/api/profile'),
	saveProfile: (p: Profile) => request<Profile>('PUT', '/api/profile', p),
	insights: () => request<Insight | null>('GET', '/api/insights'),
	regenerateInsight: () => request<Insight>('POST', '/api/insights')
};
