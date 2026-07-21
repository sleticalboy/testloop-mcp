#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

function usage() {
  console.error('Usage: node scripts/export-agent-decision-fixtures.mjs <output-dir> [manifest-json]');
  console.error('');
  console.error('Defaults:');
  console.error('  manifest-json: docs/fixtures/agent-decision-fixtures.json');
}

if (process.argv.includes('-h') || process.argv.includes('--help')) {
  usage();
  process.exit(0);
}

if (process.argv.length < 3 || process.argv.length > 4) {
  usage();
  process.exit(2);
}

const outputDir = path.resolve(process.argv[2]);
const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const manifestPath = process.argv[3]
  ? path.resolve(process.argv[3])
  : path.resolve(repoRoot, 'docs/fixtures/agent-decision-fixtures.json');

function readJSON(filePath) {
  try {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
  } catch (error) {
    throw new Error(`${filePath}: ${error.message}`);
  }
}

function copyRequiredFile(relativePath) {
  const source = path.resolve(repoRoot, relativePath);
  const destination = path.resolve(outputDir, relativePath);
  if (!destination.startsWith(outputDir + path.sep)) {
    throw new Error(`${relativePath}: resolves outside output directory`);
  }
  if (!fs.existsSync(source)) {
    throw new Error(`${relativePath}: source file does not exist`);
  }
  fs.mkdirSync(path.dirname(destination), { recursive: true });
  fs.copyFileSync(source, destination);
}

if (fs.existsSync(outputDir) && fs.readdirSync(outputDir).length > 0) {
  console.error(`output directory already exists and is not empty: ${outputDir}`);
  process.exit(2);
}
fs.mkdirSync(outputDir, { recursive: true });

let manifest;
try {
  manifest = readJSON(manifestPath);
  copyRequiredFile('docs/fixtures/agent-decision-fixtures.json');
  copyRequiredFile('docs/fixtures/agent-decision-fixtures.schema.json');
  copyRequiredFile('scripts/validate-agent-decision-fixtures.mjs');
  for (const item of manifest.fixtures || []) {
    if (!item || typeof item.path !== 'string') {
      throw new Error(`${manifestPath}: every fixture must contain a path`);
    }
    copyRequiredFile(item.path);
  }
  fs.writeFileSync(path.join(outputDir, 'package.json'), `${JSON.stringify({
    name: 'testloop-agent-decision-fixtures',
    private: true,
    type: 'module',
    scripts: {
      test: 'node scripts/validate-agent-decision-fixtures.mjs --json docs/fixtures/agent-decision-fixtures.json .',
    },
  }, null, 2)}\n`, 'utf8');
  fs.writeFileSync(path.join(outputDir, 'README.md'), [
    '# testloop-mcp Agent decision fixtures',
    '',
    'Validate all copied fixtures with JSON output:',
    '',
    '```bash',
    'node scripts/validate-agent-decision-fixtures.mjs --json \\',
    '  docs/fixtures/agent-decision-fixtures.json \\',
    '  .',
    '```',
    '',
    'Or run the bundled package script:',
    '',
    '```bash',
    'npm test --silent',
    '```',
    '',
    'The validator returns non-zero on failure and still writes parseable JSON with `status=failed`.',
    '',
  ].join('\n'), 'utf8');
} catch (error) {
  console.error(`agent decision fixture export failed: ${error.message}`);
  process.exit(1);
}

console.log('agent_decision_fixture_export_status=passed');
console.log(`output_dir=${outputDir}`);
console.log(`fixture_count=${manifest.fixtures.length}`);
