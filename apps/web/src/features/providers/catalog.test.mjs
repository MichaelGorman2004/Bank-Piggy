import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const catalog = JSON.parse(await readFile(new URL('./catalog.json', import.meta.url), 'utf8'));

test('the provider catalog has 100 unique options backed by safe local images', async () => {
  assert.equal(catalog.length, 100);
  assert.equal(new Set(catalog.map((provider) => provider.id)).size, catalog.length);
  for (const provider of catalog) {
    assert.match(provider.id, /^[a-z0-9-]+$/);
    assert.ok(provider.name.trim());
    const path = new URL(`../../../public/assets/providers/${provider.id}.svg`, import.meta.url);
    const svg = await readFile(path, 'utf8');
    assert.match(svg, /^<svg /);
    assert.doesNotMatch(svg, /<script|<foreignObject|\son[a-z]+=|(?:href|src)=/i);
  }
  assert.ok(catalog.some((provider) => provider.id === 'netflix'));
  assert.ok(catalog.some((provider) => provider.id === 'amazon-prime'));
});
