#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function usage() {
  console.error('Usage: node scripts/validate-release-response-adopter-summary.mjs [--json] [summary-json]');
  console.error('');
  console.error('Defaults:');
  console.error('  summary-json: docs/fixtures/release-response-adopter-summary/passed.json');
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
  positionalArgs[0] || 'docs/fixtures/release-response-adopter-summary/passed.json',
);
const requiredFields = [
  'schema_version',
  'status',
  'repo_dir',
  'readme_path',
  'workflow_path',
  'package_dir',
  'install_summary_json',
  'agent_response_json',
  'consumer_json',
  'release_ref',
  'fixture_count',
  'agent_next_step',
  'should_accept',
  'npm_exit_code',
  'failures',
];
const failures = [];
let summary = {};

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

function emitJSON(status) {
  const summaryFailures = Array.isArray(summary.failures) ? summary.failures : [];
  const outputFailures = status === 'failed'
    ? [...summaryFailures, ...failures]
    : failures;
  console.log(JSON.stringify({
    status,
    summary_json: summaryPath,
    release_ref: typeof summary.release_ref === 'string' ? summary.release_ref : '',
    fixture_count: Number.isInteger(summary.fixture_count) ? summary.fixture_count : 0,
    agent_next_step: typeof summary.agent_next_step === 'string' ? summary.agent_next_step : '',
    should_accept: summary.should_accept === true,
    npm_exit_code: Number.isInteger(summary.npm_exit_code) ? summary.npm_exit_code : null,
    failures: outputFailures,
  }, null, 2));
}

function emitFailures() {
  if (jsonMode) {
    emitJSON('failed');
    return;
  }
  console.error('release response adopter summary validation failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
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
for (const field of [
  'repo_dir',
  'readme_path',
  'workflow_path',
  'package_dir',
  'install_summary_json',
  'agent_response_json',
  'consumer_json',
  'release_ref',
]) {
  requireNonEmptyString(summary[field], `${summaryPath}: ${field}`);
}
if (summary.release_ref !== 'v0.5.20') {
  failures.push(`${summaryPath}: release_ref must be v0.5.20`);
}
if (summary.fixture_count !== 8) {
  failures.push(`${summaryPath}: fixture_count must be 8`);
}
if (summary.agent_next_step !== 'ready') {
  failures.push(`${summaryPath}: agent_next_step must be ready`);
}
if (summary.should_accept !== true) {
  failures.push(`${summaryPath}: should_accept must be true`);
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
  console.log(`release_response_adopter_summary_status=passed release_ref=${summary.release_ref}`);
  console.log(`release_response_adopter_summary_fixture_count=${summary.fixture_count}`);
  console.log(`release_response_adopter_summary_agent_next_step=${summary.agent_next_step}`);
  console.log(`release_response_adopter_summary_should_accept=${summary.should_accept}`);
}
