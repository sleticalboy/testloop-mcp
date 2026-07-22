#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function usage() {
  console.error('Usage: node scripts/validate-release-response-adopter-artifact-verification.mjs [--json] [verification-json]');
  console.error('');
  console.error('Defaults:');
  console.error('  verification-json: docs/fixtures/release-response-adopter-artifact-verification/passed.json');
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

const verificationPath = path.resolve(
  positionalArgs[0] || 'docs/fixtures/release-response-adopter-artifact-verification/passed.json',
);
const requiredFields = [
  'schema_version',
  'status',
  'artifact_dir',
  'summary_json',
  'release_ref',
  'fixture_count',
  'agent_next_step',
  'should_accept',
  'required_files',
  'files',
  'failures',
];
const requiredFilePaths = [
  'testloop-release-response-adopter-summary.json',
  'testloop-release-response-install-summary.json',
  'testloop-release-response-client/testloop-release-smoke-summary.json',
  'testloop-release-response-client/testloop-release-response.json',
  'testloop-release-response-consumer.json',
  'testloop-release-response-summary-consumer.json',
];
const failures = [];
let verification = {};

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
  const verificationFailures = Array.isArray(verification.failures) ? verification.failures : [];
  const outputFailures = status === 'failed'
    ? [...verificationFailures, ...failures]
    : failures;
  console.log(JSON.stringify({
    status,
    verification_json: verificationPath,
    release_ref: typeof verification.release_ref === 'string' ? verification.release_ref : '',
    fixture_count: Number.isInteger(verification.fixture_count) ? verification.fixture_count : 0,
    agent_next_step: typeof verification.agent_next_step === 'string' ? verification.agent_next_step : '',
    should_accept: verification.should_accept === true,
    required_files: Number.isInteger(verification.required_files) ? verification.required_files : 0,
    failures: outputFailures,
  }, null, 2));
}

function emitFailures() {
  if (jsonMode) {
    emitJSON('failed');
    return;
  }
  console.error('release response adopter artifact verification validation failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
}

try {
  verification = readJSON(verificationPath);
} catch (error) {
  failures.push(error.message);
  emitFailures();
  process.exit(1);
}

for (const field of requiredFields) {
  if (!Object.prototype.hasOwnProperty.call(verification, field)) {
    failures.push(`${verificationPath}: missing required field ${field}`);
  }
}

const extraFields = Object.keys(verification).filter((field) => !requiredFields.includes(field));
for (const field of extraFields) {
  failures.push(`${verificationPath}: unexpected field ${field}`);
}

if (verification.schema_version !== 1) {
  failures.push(`${verificationPath}: schema_version must be 1`);
}
if (verification.status !== 'passed') {
  failures.push(`${verificationPath}: status must be passed`);
}
for (const field of [
  'artifact_dir',
  'summary_json',
  'release_ref',
  'agent_next_step',
]) {
  requireNonEmptyString(verification[field], `${verificationPath}: ${field}`);
}
if (verification.release_ref !== 'v0.5.20') {
  failures.push(`${verificationPath}: release_ref must be v0.5.20`);
}
if (verification.fixture_count !== 8) {
  failures.push(`${verificationPath}: fixture_count must be 8`);
}
if (verification.agent_next_step !== 'ready') {
  failures.push(`${verificationPath}: agent_next_step must be ready`);
}
if (verification.should_accept !== true) {
  failures.push(`${verificationPath}: should_accept must be true`);
}
if (verification.required_files !== requiredFilePaths.length) {
  failures.push(`${verificationPath}: required_files must be ${requiredFilePaths.length}`);
}
if (!Array.isArray(verification.failures) || verification.failures.length !== 0) {
  failures.push(`${verificationPath}: failures must be an empty array`);
}
if (!Array.isArray(verification.files)) {
  failures.push(`${verificationPath}: files must be an array`);
} else {
  if (verification.files.length !== requiredFilePaths.length) {
    failures.push(`${verificationPath}: files length must be ${requiredFilePaths.length}`);
  }
  const seenPaths = verification.files.map((entry) => entry && entry.path);
  for (const requiredPath of requiredFilePaths) {
    if (!seenPaths.includes(requiredPath)) {
      failures.push(`${verificationPath}: files missing ${requiredPath}`);
    }
  }
  for (const entry of verification.files) {
    if (!entry || typeof entry !== 'object' || Array.isArray(entry)) {
      failures.push(`${verificationPath}: files entries must be objects`);
      continue;
    }
    const entryExtraFields = Object.keys(entry).filter((field) => !['path', 'exists'].includes(field));
    for (const field of entryExtraFields) {
      failures.push(`${verificationPath}: files entry ${entry.path || '<missing>'} unexpected field ${field}`);
    }
    requireNonEmptyString(entry.path, `${verificationPath}: files entry path`);
    if (entry.exists !== true) {
      failures.push(`${verificationPath}: files entry ${entry.path || '<missing>'} exists must be true`);
    }
  }
}

if (failures.length > 0) {
  emitFailures();
  process.exit(1);
}

if (jsonMode) {
  emitJSON('passed');
} else {
  console.log(`release_response_adopter_artifact_verification_status=passed release_ref=${verification.release_ref}`);
  console.log(`release_response_adopter_artifact_status=${verification.status}`);
  console.log(`release_response_adopter_artifact_verification_fixture_count=${verification.fixture_count}`);
  console.log(`release_response_adopter_artifact_verification_agent_next_step=${verification.agent_next_step}`);
  console.log(`release_response_adopter_artifact_verification_should_accept=${verification.should_accept}`);
  console.log(`release_response_adopter_artifact_verification_required_files=${verification.required_files}`);
}
