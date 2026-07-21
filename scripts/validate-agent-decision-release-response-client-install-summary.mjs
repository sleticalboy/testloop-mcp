#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function usage() {
  console.error('Usage: node scripts/validate-agent-decision-release-response-client-install-summary.mjs [--json] [summary-json]');
  console.error('');
  console.error('Defaults:');
  console.error('  summary-json: docs/fixtures/agent-decision-release-response-client-install-summary/passed.json');
}

if (process.argv.includes('-h') || process.argv.includes('--help')) {
  usage();
  process.exit(0);
}

const positionalArgs = [];
let jsonMode = false;
for (const arg of process.argv.slice(2)) {
  if (arg === '--json') {
    jsonMode = true;
    continue;
  }
  if (arg.startsWith('-')) {
    usage();
    process.exit(2);
  }
  positionalArgs.push(arg);
}

if (positionalArgs.length > 1) {
  usage();
  process.exit(2);
}

const summaryPath = path.resolve(
  positionalArgs[0] || 'docs/fixtures/agent-decision-release-response-client-install-summary/passed.json',
);
const requiredFields = [
  'schema_version',
  'status',
  'client_dir',
  'workflow_path',
  'package_dir',
  'release_summary_json',
  'agent_response_json',
  'release_ref',
  'fixture_count',
  'decisions',
  'agent_next_step',
  'npm_exit_code',
  'failures',
];
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
let summary = {};

function emitJSON(status) {
  console.log(JSON.stringify({
    status,
    summary_json: summaryPath,
    release_ref: typeof summary.release_ref === 'string' ? summary.release_ref : '',
    fixture_count: Number.isInteger(summary.fixture_count) ? summary.fixture_count : 0,
    agent_next_step: typeof summary.agent_next_step === 'string' ? summary.agent_next_step : '',
    decisions: Array.isArray(summary.decisions) ? summary.decisions : [],
    failures,
  }, null, 2));
}

function emitFailures() {
  if (jsonMode) {
    emitJSON('failed');
    return;
  }
  console.error('agent decision release response client install summary validation failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
}

function readJSON(filePath) {
  try {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
  } catch (error) {
    throw new Error(`${filePath}: ${error.message}`);
  }
}

function requireNonEmptyString(value, label) {
  if (typeof value !== 'string' || value.length === 0) {
    failures.push(`${label}: expected non-empty string`);
  }
}

try {
  summary = readJSON(summaryPath);
} catch (error) {
  failures.push(error.message);
  emitFailures();
  process.exit(1);
}

for (const field of requiredFields) {
  if (!Object.prototype.hasOwnProperty.call(summary, field)) {
    failures.push(`${summaryPath}: missing required field ${field}`);
  }
}

const extraFields = Object.keys(summary).filter((field) => !requiredFields.includes(field));
for (const field of extraFields) {
  failures.push(`${summaryPath}: unexpected field ${field}`);
}

if (summary.schema_version !== 1) {
  failures.push(`${summaryPath}: schema_version must be 1`);
}
if (summary.status !== 'written') {
  failures.push(`${summaryPath}: status must be written`);
}
for (const field of ['client_dir', 'workflow_path', 'package_dir', 'release_summary_json', 'agent_response_json', 'release_ref']) {
  requireNonEmptyString(summary[field], `${summaryPath}: ${field}`);
}
if (summary.fixture_count !== 8) {
  failures.push(`${summaryPath}: fixture_count must be 8`);
}
if (JSON.stringify(summary.decisions) !== JSON.stringify(expectedDecisions)) {
  failures.push(`${summaryPath}: decisions must be ${expectedDecisions.join(',')}`);
}
if (summary.agent_next_step !== 'ready') {
  failures.push(`${summaryPath}: agent_next_step must be ready`);
}
if (summary.npm_exit_code !== 0) {
  failures.push(`${summaryPath}: npm_exit_code must be 0`);
}
if (!Array.isArray(summary.failures) || summary.failures.length !== 0) {
  failures.push(`${summaryPath}: failures must be an empty array`);
}

if (failures.length > 0) {
  emitFailures();
  process.exit(1);
}

if (jsonMode) {
  emitJSON('passed');
} else {
  console.log(`agent_decision_release_response_client_install_summary_status=passed release_ref=${summary.release_ref}`);
  console.log(`agent_decision_release_response_client_install_summary_fixture_count=${summary.fixture_count}`);
  console.log(`agent_decision_release_response_client_install_summary_agent_next_step=${summary.agent_next_step}`);
}
