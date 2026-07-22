#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function usage() {
  console.error('Usage: node scripts/verify-release-response-adopter-artifact.mjs [--json] <artifact-dir>');
  console.error('');
  console.error('Verify a downloaded testloop release response adopter artifact directory.');
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

if (positionalArgs.length !== 1) {
  usage();
  process.exit(2);
}

const artifactDir = path.resolve(positionalArgs[0]);
const requiredFiles = [
  'testloop-release-response-adopter-summary.json',
  'testloop-release-response-install-summary.json',
  'testloop-release-response-client/testloop-release-smoke-summary.json',
  'testloop-release-response-client/testloop-release-response.json',
  'testloop-release-response-consumer.json',
  'testloop-release-response-summary-consumer.json',
];
const failures = [];
const fileResults = [];

function artifactPath(relativePath) {
  return path.join(artifactDir, relativePath);
}

function normalizeForSuffix(value) {
  return String(value || '').replace(/\\/g, '/');
}

function hasSuffix(value, suffix) {
  return normalizeForSuffix(value).endsWith(suffix);
}

function recordRequiredFiles() {
  if (!fs.existsSync(artifactDir)) {
    failures.push(`artifact dir does not exist: ${artifactDir}`);
    return;
  }
  if (!fs.statSync(artifactDir).isDirectory()) {
    failures.push(`artifact path is not a directory: ${artifactDir}`);
    return;
  }

  for (const relativePath of requiredFiles) {
    const fullPath = artifactPath(relativePath);
    const exists = fs.existsSync(fullPath);
    fileResults.push({path: relativePath, exists});
    if (!exists) {
      failures.push(`missing required file ${relativePath}`);
    }
  }
}

function readJSON(relativePath, label) {
  const fullPath = artifactPath(relativePath);
  if (!fs.existsSync(fullPath)) {
    return {};
  }
  try {
    return JSON.parse(fs.readFileSync(fullPath, 'utf8'));
  } catch (error) {
    failures.push(`${label}: invalid JSON: ${error.message}`);
    return {};
  }
}

function expectEqual(actual, expected, label) {
  if (actual !== expected) {
    failures.push(`${label}=${JSON.stringify(actual)}, want ${JSON.stringify(expected)}`);
  }
}

function expectEmptyArray(value, label) {
  if (!Array.isArray(value) || value.length !== 0) {
    failures.push(`${label} must be an empty array`);
  }
}

function expectPathSuffix(value, suffix, label) {
  if (typeof value !== 'string' || !hasSuffix(value, suffix)) {
    failures.push(`${label}=${JSON.stringify(value)}, want suffix ${suffix}`);
  }
}

recordRequiredFiles();

const adopterSummary = readJSON('testloop-release-response-adopter-summary.json', 'adopter summary');
const installSummary = readJSON('testloop-release-response-install-summary.json', 'install summary');
const releaseSummary = readJSON(
  'testloop-release-response-client/testloop-release-smoke-summary.json',
  'release smoke summary',
);
const agentResponse = readJSON(
  'testloop-release-response-client/testloop-release-response.json',
  'release response',
);
const consumer = readJSON('testloop-release-response-consumer.json', 'consumer response');
const summaryConsumer = readJSON(
  'testloop-release-response-summary-consumer.json',
  'summary consumer response',
);

if (Object.keys(adopterSummary).length > 0) {
  expectEqual(adopterSummary.schema_version, 1, 'adopter summary schema_version');
  expectEqual(adopterSummary.status, 'passed', 'adopter summary status');
  expectEqual(adopterSummary.release_ref, 'v0.5.20', 'adopter summary release_ref');
  expectEqual(adopterSummary.fixture_count, 8, 'adopter summary fixture_count');
  expectEqual(adopterSummary.agent_next_step, 'ready', 'adopter summary agent_next_step');
  expectEqual(adopterSummary.should_accept, true, 'adopter summary should_accept');
  expectEqual(adopterSummary.npm_exit_code, 0, 'adopter summary npm_exit_code');
  expectEmptyArray(adopterSummary.failures, 'adopter summary failures');
  expectPathSuffix(
    adopterSummary.install_summary_json,
    'testloop-release-response-install-summary.json',
    'adopter summary install_summary_json',
  );
  expectPathSuffix(
    adopterSummary.agent_response_json,
    'testloop-release-response-client/testloop-release-response.json',
    'adopter summary agent_response_json',
  );
  expectPathSuffix(
    adopterSummary.consumer_json,
    'testloop-release-response-consumer.json',
    'adopter summary consumer_json',
  );
  expectPathSuffix(
    adopterSummary.summary_consumer_json,
    'testloop-release-response-summary-consumer.json',
    'adopter summary summary_consumer_json',
  );
}

if (Object.keys(installSummary).length > 0) {
  expectEqual(installSummary.status, 'written', 'install summary status');
  expectEqual(installSummary.release_ref, 'v0.5.20', 'install summary release_ref');
  expectEqual(installSummary.fixture_count, 8, 'install summary fixture_count');
  expectEqual(installSummary.agent_next_step, 'ready', 'install summary agent_next_step');
  expectEqual(installSummary.npm_exit_code, 0, 'install summary npm_exit_code');
  expectEmptyArray(installSummary.failures, 'install summary failures');
}

if (Object.keys(releaseSummary).length > 0) {
  expectEqual(releaseSummary.status, 'passed', 'release smoke summary status');
  expectEqual(releaseSummary.release_ref, 'v0.5.20', 'release smoke summary release_ref');
  expectEqual(releaseSummary.fixture_count, 8, 'release smoke summary fixture_count');
  expectEqual(releaseSummary.agent_next_steps?.client, 'ready', 'release smoke summary client next step');
  expectEqual(releaseSummary.agent_next_steps?.consumer, 'ready', 'release smoke summary consumer next step');
  expectEmptyArray(releaseSummary.failures, 'release smoke summary failures');
}

if (Object.keys(agentResponse).length > 0) {
  expectEqual(agentResponse.status, 'passed', 'release response status');
  expectEqual(agentResponse.agent_next_step, 'ready', 'release response agent_next_step');
  expectEqual(agentResponse.evidence?.release_ref, 'v0.5.20', 'release response release_ref');
  expectEqual(agentResponse.evidence?.fixture_count, 8, 'release response fixture_count');
  expectEmptyArray(agentResponse.failures, 'release response failures');
}

for (const [label, payload] of [
  ['consumer response', consumer],
  ['summary consumer response', summaryConsumer],
]) {
  if (Object.keys(payload).length === 0) {
    continue;
  }
  expectEqual(payload.status, 'passed', `${label} status`);
  expectEqual(payload.agent_next_step, 'ready', `${label} agent_next_step`);
  expectEqual(payload.should_accept, true, `${label} should_accept`);
  expectEmptyArray(payload.failures, `${label} failures`);
}

const status = failures.length === 0 ? 'passed' : 'failed';
const agentNextStep = status === 'passed'
  ? (typeof adopterSummary.agent_next_step === 'string' ? adopterSummary.agent_next_step : '')
  : 'inspect-release-response-adopter-artifact';
const output = {
  status,
  artifact_dir: artifactDir,
  summary_json: artifactPath('testloop-release-response-adopter-summary.json'),
  release_ref: typeof adopterSummary.release_ref === 'string' ? adopterSummary.release_ref : '',
  fixture_count: Number.isInteger(adopterSummary.fixture_count) ? adopterSummary.fixture_count : 0,
  agent_next_step: agentNextStep,
  should_accept: status === 'passed' && adopterSummary.should_accept === true,
  required_files: requiredFiles.length,
  files: fileResults,
  failures,
};

if (jsonMode) {
  console.log(JSON.stringify(output, null, 2));
} else {
  console.log(`release_response_adopter_artifact_status=${output.status}`);
  console.log(`artifact_dir=${output.artifact_dir}`);
  console.log(`summary_json=${output.summary_json}`);
  console.log(`release_ref=${output.release_ref}`);
  console.log(`fixture_count=${output.fixture_count}`);
  console.log(`agent_next_step=${output.agent_next_step}`);
  console.log(`should_accept=${output.should_accept}`);
  console.log(`required_files=${output.required_files}`);
  if (failures.length > 0) {
    console.log(`failures=${failures.join('; ')}`);
  }
}

if (status !== 'passed') {
  process.exitCode = 1;
}
