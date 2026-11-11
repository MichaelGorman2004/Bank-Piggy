import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import * as icons from 'simple-icons';

const require = createRequire(import.meta.url);
const catalog = require('../apps/web/src/features/providers/catalog.json');
const outputDirectory = 'apps/web/public/assets/providers';
const archiveColors = { amazon: 'FF9900', microsoft: '5E5E5E', adobe: 'FF0000', slack: '4A154B' };
const checkOnly = process.argv.includes('--check');

/** Builds an SVG from a pinned icon or a checked-in source asset. */
async function providerSvg(provider) {
  const icon = icons[provider.icon];
  const archived = archiveColors[provider.icon];
  if (!icon && !archived) throw new Error(`Missing provider icon: ${provider.icon}`);
  const source =
    icon?.svg ?? (await readFile(`scripts/provider-sources/${provider.icon}.svg`, 'utf8'));
  if (/<script|<foreignObject|\son[a-z]+=|(?:href|src)=/i.test(source)) {
    throw new Error(`Provider icon contains an active or external reference: ${provider.id}`);
  }
  const color = icon?.hex ?? archived;
  return source.trim().replace('<svg ', `<svg fill="#${color}" `) + '\n';
}

/** Generates only allowlisted local images, or checks the committed copies without writing. */
async function main() {
  const identifiers = new Set();
  if (!checkOnly) await mkdir(outputDirectory, { recursive: true });
  for (const provider of catalog) {
    if (!/^[a-z0-9-]+$/.test(provider.id) || identifiers.has(provider.id)) {
      throw new Error(`Invalid or duplicate provider identifier: ${provider.id}`);
    }
    identifiers.add(provider.id);
    const expected = await providerSvg(provider);
    const path = `${outputDirectory}/${provider.id}.svg`;
    if (checkOnly) {
      const actual = await readFile(path, 'utf8');
      if (actual !== expected) throw new Error(`Provider icon needs regeneration: ${path}`);
    } else {
      await writeFile(path, expected);
    }
  }
  console.log(`${identifiers.size} local provider icons ${checkOnly ? 'verified' : 'generated'}.`);
}

await main().catch((error) => {
  console.error(error.message);
  process.exitCode = 1;
});
