#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function usage() {
  console.error('Usage: node scripts/validate-agent-decision-client-ci-summary.mjs [--json] [summary-json]');
  console.error('');
  console.error('Defaults:');
  console.error('  summary-json: docs/fixtures/agent-decision-client-ci-summary/passed.json');
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
  positionalArgs[0] || 'docs/fixtures/agent-decision-client-ci-summary/passed.json',
);
const requiredFields = [
  'schema_version',
  'status',
  'client_dir',
  'fixture_dir',
  'result_json',
  'result_schema',
  'fixture_count',
  'decisions',
  'failures',
  'validator_exit_code',
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
    fixture_count: Number.isInteger(summary.fixture_count) ? summary.fixture_count : 0,
    decisions: Array.isArray(summary.decisions) ? summary.decisions : [],
    failures,
  }, null, 2));
}

function emitFailures() {
  if (jsonMode) {
    emitJSON('failed');
    return;
  }
  console.error('agent decision client CI summary validation failed:');
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
if (summary.status !== 'passed') {
  failures.push(`${summaryPath}: status must be passed`);
}
if (summary.fixture_count !== 8) {
  failures.push(`${summaryPath}: fixture_count must be 8`);
}
for (const field of ['client_dir', 'fixture_dir', 'result_json', 'result_schema']) {
  requireNonEmptyString(summary[field], `${summaryPath}: ${field}`);
}
if (JSON.stringify(summary.decisions) !== JSON.stringify(expectedDecisions)) {
  failures.push(`${summaryPath}: decisions must be ${expectedDecisions.join(',')}`);
}
if (!Array.isArray(summary.failures) || summary.failures.length !== 0) {
  failures.push(`${summaryPath}: failures must be an empty array`);
}
if (summary.validator_exit_code !== 0) {
  failures.push(`${summaryPath}: validator_exit_code must be 0`);
}

const expectedResultJSON = path.join(summary.client_dir || '', 'agent-decision-fixtures-result.json');
if (path.normalize(summary.result_json || '') !== path.normalize(expectedResultJSON)) {
  failures.push(`${summaryPath}: result_json must be ${expectedResultJSON}`);
}
const expectedResultSchema = path.join(
  summary.fixture_dir || '',
  'docs/fixtures/agent-decision-fixtures-result.schema.json',
);
if (path.normalize(summary.result_schema || '') !== path.normalize(expectedResultSchema)) {
  failures.push(`${summaryPath}: result_schema must be ${expectedResultSchema}`);
}

if (failures.length > 0) {
  emitFailures();
  process.exit(1);
}

if (jsonMode) {
  emitJSON('passed');
} else {
  console.log(`agent_decision_client_ci_summary_status=passed fixture_count=${summary.fixture_count}`);
  console.log(`agent_decision_client_ci_summary_decisions=${summary.decisions.join(',')}`);
}
