#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function usage() {
  console.error('Usage: node scripts/validate-agent-decision-client-consumer-response.mjs [--json] [response-json]');
  console.error('');
  console.error('Defaults:');
  console.error('  response-json: docs/fixtures/agent-decision-client-consumer-response/passed.json');
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

const responsePath = path.resolve(
  positionalArgs[0] || 'docs/fixtures/agent-decision-client-consumer-response/passed.json',
);
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
const requiredFields = [
  'schema_version',
  'status',
  'agent_next_step',
  'summary_json',
  'evidence',
  'failures',
];
const requiredEvidenceFields = [
  'helper_ref',
  'fixture_count',
  'decisions',
  'result_json',
  'client_summary_json',
  'client_summary_validator_json',
  'client_response_json',
  'client_response_validator_json',
  'workflow_path',
  'install_summary_validator_exit_code',
  'client_summary_validator_exit_code',
  'client_response_validator_exit_code',
  'fixture_validator_exit_code',
  'npm_validator_exit_code',
];
const failures = [];
let response = {};

function emitJSON(status) {
  const evidence = response && typeof response.evidence === 'object' ? response.evidence : {};
  console.log(JSON.stringify({
    status,
    response_json: responsePath,
    agent_next_step: typeof response.agent_next_step === 'string' ? response.agent_next_step : '',
    fixture_count: Number.isInteger(evidence.fixture_count) ? evidence.fixture_count : 0,
    decisions: Array.isArray(evidence.decisions) ? evidence.decisions : [],
    failures,
  }, null, 2));
}

function emitFailures() {
  if (jsonMode) {
    emitJSON('failed');
    return;
  }
  console.error('agent decision client consumer response validation failed:');
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
  response = readJSON(responsePath);
} catch (error) {
  failures.push(error.message);
  emitFailures();
  process.exit(1);
}

for (const field of requiredFields) {
  if (!Object.prototype.hasOwnProperty.call(response, field)) {
    failures.push(`${responsePath}: missing required field ${field}`);
  }
}

const extraFields = Object.keys(response).filter((field) => !requiredFields.includes(field));
for (const field of extraFields) {
  failures.push(`${responsePath}: unexpected field ${field}`);
}

const evidence = response.evidence && typeof response.evidence === 'object' ? response.evidence : {};
if (response.schema_version !== 1) {
  failures.push(`${responsePath}: schema_version must be 1`);
}
if (response.status !== 'passed') {
  failures.push(`${responsePath}: status must be passed`);
}
if (response.agent_next_step !== 'ready') {
  failures.push(`${responsePath}: agent_next_step must be ready`);
}
requireNonEmptyString(response.summary_json, `${responsePath}: summary_json`);
if (!Array.isArray(response.failures) || response.failures.length !== 0) {
  failures.push(`${responsePath}: failures must be an empty array`);
}

for (const field of requiredEvidenceFields) {
  if (!Object.prototype.hasOwnProperty.call(evidence, field)) {
    failures.push(`${responsePath}: evidence missing required field ${field}`);
  }
}
const extraEvidenceFields = Object.keys(evidence).filter((field) => !requiredEvidenceFields.includes(field));
for (const field of extraEvidenceFields) {
  failures.push(`${responsePath}: evidence unexpected field ${field}`);
}

if (evidence.helper_ref !== 'v0.5.21') {
  failures.push(`${responsePath}: evidence.helper_ref must be v0.5.21`);
}
if (evidence.fixture_count !== expectedDecisions.length) {
  failures.push(`${responsePath}: evidence.fixture_count must be ${expectedDecisions.length}`);
}
if (JSON.stringify(evidence.decisions) !== JSON.stringify(expectedDecisions)) {
  failures.push(`${responsePath}: evidence.decisions must be ${expectedDecisions.join(',')}`);
}
for (const field of ['result_json', 'client_summary_json', 'client_summary_validator_json', 'client_response_json', 'client_response_validator_json', 'workflow_path']) {
  requireNonEmptyString(evidence[field], `${responsePath}: evidence.${field}`);
}
for (const field of [
  'install_summary_validator_exit_code',
  'client_summary_validator_exit_code',
  'client_response_validator_exit_code',
  'fixture_validator_exit_code',
  'npm_validator_exit_code',
]) {
  if (evidence[field] !== 0) {
    failures.push(`${responsePath}: evidence.${field} must be 0`);
  }
}

if (failures.length > 0) {
  emitFailures();
  process.exit(1);
}

if (jsonMode) {
  emitJSON('passed');
} else {
  console.log(`agent_decision_client_consumer_response_status=passed agent_next_step=${response.agent_next_step}`);
  console.log(`agent_decision_client_consumer_response_fixture_count=${evidence.fixture_count}`);
  console.log(`agent_decision_client_consumer_response_decisions=${evidence.decisions.join(',')}`);
}
