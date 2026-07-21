#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

function usage() {
  console.error('Usage: node scripts/export-agent-decision-release-response-client.mjs <output-dir> [release-smoke-summary-json]');
  console.error('');
  console.error('Defaults:');
  console.error('  release-smoke-summary-json: docs/fixtures/agent-decision-client-release-smoke-summary/passed.json');
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
const summaryPath = process.argv[3]
  ? path.resolve(process.argv[3])
  : path.resolve(repoRoot, 'docs/fixtures/agent-decision-client-release-smoke-summary/passed.json');
const responseFixtureDir = 'docs/fixtures/agent-decision-client-release-response';
const responseFixtureFiles = [
  'passed.json',
  'installer-drift.json',
  'client-response-drift.json',
  'consumer-response-drift.json',
  'fixture-drift.json',
];

function copyRequiredFile(relativePath, destinationRelativePath = relativePath) {
  const source = path.resolve(repoRoot, relativePath);
  const destination = path.resolve(outputDir, destinationRelativePath);
  if (!destination.startsWith(outputDir + path.sep)) {
    throw new Error(`${destinationRelativePath}: resolves outside output directory`);
  }
  if (!fs.existsSync(source)) {
    throw new Error(`${relativePath}: source file does not exist`);
  }
  fs.mkdirSync(path.dirname(destination), { recursive: true });
  fs.copyFileSync(source, destination);
}

function copyInputFile(sourcePath, destinationRelativePath) {
  const destination = path.resolve(outputDir, destinationRelativePath);
  if (!destination.startsWith(outputDir + path.sep)) {
    throw new Error(`${destinationRelativePath}: resolves outside output directory`);
  }
  if (!fs.existsSync(sourcePath)) {
    throw new Error(`${sourcePath}: source file does not exist`);
  }
  fs.mkdirSync(path.dirname(destination), { recursive: true });
  fs.copyFileSync(sourcePath, destination);
}

try {
  if (fs.existsSync(outputDir) && fs.readdirSync(outputDir).length > 0) {
    console.error(`output directory already exists and is not empty: ${outputDir}`);
    process.exit(2);
  }
  fs.mkdirSync(outputDir, { recursive: true });

  copyInputFile(summaryPath, 'testloop-release-smoke-summary.json');
  copyRequiredFile(
    'scripts/render-agent-decision-client-release-response.mjs',
    'scripts/render-agent-decision-client-release-response.mjs',
  );
  copyRequiredFile(
    'docs/fixtures/agent-decision-client-release-response.schema.json',
    'docs/fixtures/agent-decision-client-release-response.schema.json',
  );
  for (const fixtureFile of responseFixtureFiles) {
    copyRequiredFile(
      `${responseFixtureDir}/${fixtureFile}`,
      `${responseFixtureDir}/${fixtureFile}`,
    );
  }

  fs.writeFileSync(path.join(outputDir, 'scripts/assert-release-response.mjs'), `import fs from 'node:fs';
import process from 'node:process';

const responsePath = process.argv[2];
if (!responsePath) {
  console.error('Usage: node scripts/assert-release-response.mjs <response-json>');
  process.exit(2);
}

const payload = JSON.parse(fs.readFileSync(responsePath, 'utf8'));
const expectedDecisions = [
  'accept',
  'accept',
  'accept',
  'manual-review',
  'manual-review',
  'manual-review',
  'apply-repair',
  'needs-better-input',
];
const failures = [];

if (payload.schema_version !== 1) {
  failures.push('schema_version must be 1');
}
if (payload.status !== 'passed') {
  failures.push(\`status=\${payload.status || 'missing'}, want passed\`);
}
if (payload.agent_next_step !== 'ready') {
  failures.push(\`agent_next_step=\${payload.agent_next_step || 'missing'}, want ready\`);
}
if (!payload.evidence || typeof payload.evidence.release_ref !== 'string' || payload.evidence.release_ref.length === 0) {
  failures.push('evidence.release_ref is required');
}
if (!payload.evidence || payload.evidence.fixture_count !== expectedDecisions.length) {
  failures.push(\`evidence.fixture_count=\${payload.evidence?.fixture_count}, want \${expectedDecisions.length}\`);
}
if (JSON.stringify(payload.evidence?.decisions || []) !== JSON.stringify(expectedDecisions)) {
  failures.push('evidence.decisions drifted');
}
if (payload.evidence?.agent_next_steps?.client !== 'ready') {
  failures.push('evidence.agent_next_steps.client must be ready');
}
if (payload.evidence?.agent_next_steps?.consumer !== 'ready') {
  failures.push('evidence.agent_next_steps.consumer must be ready');
}
if (!Array.isArray(payload.failures) || payload.failures.length > 0) {
  failures.push('payload.failures must be an empty array');
}

if (failures.length > 0) {
  console.error(failures.join('\\n'));
  process.exit(1);
}
`, 'utf8');

  fs.writeFileSync(path.join(outputDir, 'package.json'), `${JSON.stringify({
    name: 'testloop-agent-decision-release-response-client',
    private: true,
    type: 'module',
    scripts: {
      test: 'node scripts/render-agent-decision-client-release-response.mjs --json testloop-release-smoke-summary.json > testloop-release-response.json && node scripts/assert-release-response.mjs testloop-release-response.json',
    },
  }, null, 2)}\n`, 'utf8');

  fs.writeFileSync(path.join(outputDir, 'README.md'), [
    '# testloop-mcp Agent decision release response client',
    '',
    'Validate the copied release smoke summary and render an Agent response:',
    '',
    '```bash',
    'npm test --silent',
    '```',
    '',
    'Replace `testloop-release-smoke-summary.json` with the output from:',
    '',
    '```bash',
    'scripts/showcase-agent-decision-client-release-smoke.sh --json',
    '```',
    '',
    'The renderer writes `testloop-release-response.json`. Upload that file as a CI artifact when the check fails.',
    '',
    'Bundled response fixtures live under `docs/fixtures/agent-decision-client-release-response/`.',
    '',
  ].join('\n'), 'utf8');
} catch (error) {
  console.error(`agent decision release response client export failed: ${error.message}`);
  process.exit(1);
}

console.log('agent_decision_release_response_client_export_status=passed');
console.log(`output_dir=${outputDir}`);
console.log(`summary_json=${path.resolve(outputDir, 'testloop-release-smoke-summary.json')}`);
console.log(`response_fixture_count=${responseFixtureFiles.length}`);
