// Generates the app icons from the design in scripts/icon.svg: npm run icons
// Writes content-hashed favicons and PNGs to web/static/icons (rendered with Playwright's Chromium).
import { mkdirSync, readdirSync, readFileSync, unlinkSync, writeFileSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { chromium } from '@playwright/test';

const source = readFileSync('scripts/icon.svg', 'utf8').trim();

/** rounded: transparent corners (favicon, manifest "any"); otherwise the full square (iOS, maskable). */
function svg(rounded) {
	if (!rounded) return source + '\n';
	return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
<clipPath id="corners"><rect width="512" height="512" rx="112"/></clipPath>
<g clip-path="url(#corners)">${source.replace('<svg ', '<svg width="512" height="512" ')}</g>
</svg>
`;
}

// New artwork gets new URLs, including the icon Safari saves during installation.
const hash = createHash('sha256')
	.update(source)
	.digest('hex')
	.slice(0, 12);
const directory = 'web/static/icons';
mkdirSync(directory, { recursive: true });
for (const file of readdirSync(directory)) {
	if (/^(favicon|apple-touch-icon|icon-192|icon-512|icon-maskable-512)-[a-f0-9]{12}\.(png|svg)$/.test(file)) {
		unlinkSync(`${directory}/${file}`);
	}
}
const name = (base, extension = 'png') => `${base}-${hash}.${extension}`;
const outputs = [
	{ file: `${directory}/${name('favicon')}`, size: 32, rounded: true },
	{ file: `${directory}/${name('icon-192')}`, size: 192, rounded: true },
	{ file: `${directory}/${name('icon-512')}`, size: 512, rounded: true },
	{ file: `${directory}/${name('icon-maskable-512')}`, size: 512, rounded: false },
	{ file: `${directory}/${name('apple-touch-icon')}`, size: 180, rounded: false }
];

writeFileSync(`${directory}/${name('favicon', 'svg')}`, svg(true));
const app = 'web/src/app.html';
const links = [
	['rel="icon"', 'type="image/png"', 'sizes="32x32"', `href="%sveltekit.assets%/icons/${name('favicon')}"`],
	[
		'rel="icon"',
		'type="image/svg+xml"',
		'sizes="any"',
		`href="%sveltekit.assets%/icons/${name('favicon', 'svg')}"`
	],
	['rel="apple-touch-icon"', 'sizes="180x180"', `href="%sveltekit.assets%/icons/${name('apple-touch-icon')}"`]
]
	.map((attributes) => `<link\n\t\t\t${attributes.join('\n\t\t\t')}\n\t\t/>`)
	.join('\n\t\t');
writeFileSync(
	app,
	readFileSync(app, 'utf8').replace(
		/\t\t<!-- App icons -->[\s\S]*?\t\t<!-- End app icons -->/,
		`\t\t<!-- App icons -->\n\t\t${links}\n\t\t<!-- End app icons -->`
	)
);
// The README shows the 192px icon; point it at the new file.
const readme = 'README.md';
writeFileSync(
	readme,
	readFileSync(readme, 'utf8').replace(
		/web\/static\/(?:icons\/)?icon-192(?:-[a-f0-9]{12})?\.png/g,
		`web/static/icons/${name('icon-192')}`
	)
);
const manifestPath = 'web/static/manifest.webmanifest';
const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
manifest.icons = [
	{ src: `/icons/${name('icon-192')}`, sizes: '192x192', type: 'image/png' },
	{ src: `/icons/${name('icon-512')}`, sizes: '512x512', type: 'image/png' },
	{ src: `/icons/${name('icon-maskable-512')}`, sizes: '512x512', type: 'image/png', purpose: 'maskable' }
];
writeFileSync(manifestPath, JSON.stringify(manifest, null, '\t') + '\n');

const browser = await chromium.launch();
try {
	for (const { file, size, rounded } of outputs) {
		const page = await browser.newPage({
			viewport: { width: size, height: size }
		});
		const img = svg(rounded).replace('<svg ', `<svg width="${size}" height="${size}" `);
		await page.setContent(
			`<style>html,body{margin:0;background:transparent}svg{display:block}</style>${img}`
		);
		await page.screenshot({ path: file, omitBackground: rounded });
		await page.close();
		console.log(`wrote ${file}`);
	}
} finally {
	await browser.close();
}
