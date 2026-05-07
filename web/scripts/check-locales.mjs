#!/usr/bin/env node
/**
 * i18n key-parity check.
 *
 * Loads all supported locale files, computes their leaf-key sets (dotted paths
 * such as `message.success` or `financialPlanning.tabs.plannedTransactions`),
 * and compares each locale against `en.json` as the base.
 *
 * Exits 0 on full parity. Exits 1 on any divergence, printing a human-readable
 * report listing the offending locale(s) along with missing and extra keys.
 *
 * Intended to be invoked via `npm run lint:i18n`.
 */

import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const __dirname = dirname(fileURLToPath(import.meta.url));
const LOCALES_DIR = resolve(__dirname, '..', 'src', 'locales');

const BASE_LOCALE = 'en';
const LOCALES = ['en', 'uk', 'ru', 'bg', 'zh'];

/**
 * Recursively collect all leaf-key paths from a nested object.
 * A "leaf" is any non-plain-object value (strings, numbers, arrays, null, etc.).
 *
 * @param {unknown} node
 * @param {string} prefix
 * @param {Set<string>} acc
 */
function collectLeafKeys(node, prefix, acc) {
  if (node === null || typeof node !== 'object' || Array.isArray(node)) {
    acc.add(prefix);
    return;
  }
  for (const key of Object.keys(node)) {
    const path = prefix === '' ? key : `${prefix}.${key}`;
    collectLeafKeys(node[key], path, acc);
  }
}

async function loadLocale(code) {
  const filePath = resolve(LOCALES_DIR, `${code}.json`);
  const raw = await readFile(filePath, 'utf8');
  return JSON.parse(raw);
}

function diffSets(baseSet, otherSet) {
  const missing = [];
  const extra = [];
  for (const key of baseSet) {
    if (!otherSet.has(key)) missing.push(key);
  }
  for (const key of otherSet) {
    if (!baseSet.has(key)) extra.push(key);
  }
  missing.sort();
  extra.sort();
  return { missing, extra };
}

async function main() {
  const keySets = new Map();
  for (const code of LOCALES) {
    const data = await loadLocale(code);
    const set = new Set();
    collectLeafKeys(data, '', set);
    keySets.set(code, set);
  }

  const baseSet = keySets.get(BASE_LOCALE);
  if (!baseSet) {
    console.error(`Base locale "${BASE_LOCALE}" not loaded.`);
    process.exit(2);
  }

  let hasDivergence = false;
  const reports = [];

  for (const code of LOCALES) {
    if (code === BASE_LOCALE) continue;
    const otherSet = keySets.get(code);
    const { missing, extra } = diffSets(baseSet, otherSet);
    if (missing.length > 0 || extra.length > 0) {
      hasDivergence = true;
      reports.push({ code, missing, extra });
    }
  }

  if (!hasDivergence) {
    console.log(
      `OK: all locales (${LOCALES.join(', ')}) share the same ${baseSet.size} leaf keys.`,
    );
    process.exit(0);
  }

  console.error('i18n key-parity check FAILED.');
  console.error(`Base locale: ${BASE_LOCALE} (${baseSet.size} keys)`);
  console.error('');
  for (const { code, missing, extra } of reports) {
    console.error(`Locale "${code}":`);
    if (missing.length > 0) {
      console.error(`  Missing (present in ${BASE_LOCALE}, absent in ${code}):`);
      for (const key of missing) console.error(`    - ${key}`);
    }
    if (extra.length > 0) {
      console.error(`  Extra (present in ${code}, absent in ${BASE_LOCALE}):`);
      for (const key of extra) console.error(`    + ${key}`);
    }
    console.error('');
  }
  process.exit(1);
}

main().catch((err) => {
  console.error('i18n key-parity check crashed:', err);
  process.exit(2);
});
