#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function usage() {
  console.error('Usage: node scripts/render-agent-decision-client-consumer-response.mjs [--json] [consumer-smoke-summary-json]');
  console.error('');
  console.error('Defaults:');
  console.error('  consumer-smoke-summary-json: docs/fixtures/agent-decision-client-consumer-smoke-summary/passed.json');
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
  positionalArgs[0] || 'docs/fixtures/agent-decision-client-consumer-smoke-summary/passed.json',
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

function readJSON(filePath) {
  try {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
  } catch (error) {
    throw new Error(`${filePath}: ${error.message}`);
  }
}

function nonEmptyString(value) {
  return typeof value === 'string' && value.length > 0;
}

function decide(summary) {
  const failures = [];
  const evidence = {
    helper_ref: nonEmptyString(summary.helper_ref) ? summary.helper_ref : '',
    fixture_count: Number.isInteger(summary.fixture_count) ? summary.fixture_count : 0,
    decisions: Array.isArray(summary.decisions) ? summary.decisions : [],
    result_json: nonEmptyString(summary.result_json) ? summary.result_json : '',
    client_summary_json: nonEmptyString(summary.client_summary_json) ? summary.client_summary_json : '',
    client_summary_validator_json: nonEmptyString(summary.client_summary_validator_json)
      ? summary.client_summary_validator_json
      : '',
    client_response_json: nonEmptyString(summary.client_response_json) ? summary.client_response_json : '',
    client_response_validator_json: nonEmptyString(summary.client_response_validator_json)
      ? summary.client_response_validator_json
      : '',
    workflow_path: nonEmptyString(summary.workflow_path) ? summary.workflow_path : '',
    install_summary_validator_exit_code: Number.isInteger(summary.install_summary_validator_exit_code)
      ? summary.install_summary_validator_exit_code
      : -1,
    client_summary_validator_exit_code: Number.isInteger(summary.client_summary_validator_exit_code)
      ? summary.client_summary_validator_exit_code
      : -1,
    client_response_validator_exit_code: Number.isInteger(summary.client_response_validator_exit_code)
      ? summary.client_response_validator_exit_code
      : -1,
    fixture_validator_exit_code: Number.isInteger(summary.fixture_validator_exit_code)
      ? summary.fixture_validator_exit_code
      : -1,
    npm_validator_exit_code: Number.isInteger(summary.npm_validator_exit_code) ? summary.npm_validator_exit_code : -1,
  };

  if (summary.schema_version !== 1) {
    failures.push('schema_version must be 1');
  }
  if (!nonEmptyString(summary.helper_ref)) {
    failures.push('helper_ref is required');
  }
  if (!nonEmptyString(summary.workflow_path)) {
    failures.push('workflow_path is required');
  }
  if (!nonEmptyString(summary.client_summary_json)) {
    failures.push('client_summary_json is required');
  }
  if (!nonEmptyString(summary.client_summary_validator_json)) {
    failures.push('client_summary_validator_json is required');
  }
  if (!nonEmptyString(summary.client_response_json)) {
    failures.push('client_response_json is required');
  }
  if (!nonEmptyString(summary.client_response_validator_json)) {
    failures.push('client_response_validator_json is required');
  }
  if (!nonEmptyString(summary.result_json)) {
    failures.push('result_json is required');
  }
  if (summary.status !== 'passed') {
    failures.push(`summary status is ${summary.status || 'missing'}`);
  }
  if (!Array.isArray(summary.failures)) {
    failures.push('failures must be an array');
  } else if (summary.failures.length > 0) {
    failures.push(...summary.failures.map((failure) => `consumer smoke failure: ${failure}`));
  }
  for (const field of [
    'install_summary_validator_exit_code',
    'client_summary_validator_exit_code',
    'client_response_validator_exit_code',
    'fixture_validator_exit_code',
    'npm_validator_exit_code',
  ]) {
    if (summary[field] !== 0) {
      failures.push(`${field}=${summary[field]}`);
    }
  }
  if (summary.fixture_count !== expectedDecisions.length) {
    failures.push(`fixture_count=${summary.fixture_count}, want ${expectedDecisions.length}`);
  }
  if (JSON.stringify(summary.decisions) !== JSON.stringify(expectedDecisions)) {
    failures.push(`decisions must be ${expectedDecisions.join(',')}`);
  }

  let agentNextStep = 'ready';
  if (failures.some((failure) => failure.includes('fixture_count') || failure.includes('decisions'))) {
    agentNextStep = 'inspect-agent-decision-fixtures';
  } else if (failures.some((failure) => failure.includes('validator_exit_code'))) {
    agentNextStep = 'inspect-consumer-smoke-validator';
  } else if (failures.length > 0) {
    agentNextStep = 'inspect-consumer-smoke-summary';
  }

  return {
    schema_version: 1,
    status: failures.length === 0 ? 'passed' : 'failed',
    agent_next_step: agentNextStep,
    summary_json: summaryPath,
    evidence,
    failures,
  };
}

let response;
try {
  response = decide(readJSON(summaryPath));
} catch (error) {
  response = {
    schema_version: 1,
    status: 'failed',
    agent_next_step: 'inspect-consumer-smoke-summary',
    summary_json: summaryPath,
    evidence: {
      helper_ref: '',
      fixture_count: 0,
      decisions: [],
      result_json: '',
      client_summary_json: '',
      client_summary_validator_json: '',
      client_response_json: '',
      client_response_validator_json: '',
      workflow_path: '',
      install_summary_validator_exit_code: -1,
      client_summary_validator_exit_code: -1,
      client_response_validator_exit_code: -1,
      fixture_validator_exit_code: -1,
      npm_validator_exit_code: -1,
    },
    failures: [error.message],
  };
}

if (jsonMode) {
  console.log(JSON.stringify(response, null, 2));
} else {
  console.log(`agent_decision_client_consumer_response_status=${response.status}`);
  console.log(`agent_next_step=${response.agent_next_step}`);
  console.log(`helper_ref=${response.evidence.helper_ref}`);
  console.log(`fixture_count=${response.evidence.fixture_count}`);
  console.log(`decisions=${response.evidence.decisions.join(',')}`);
  console.log(`result_json=${response.evidence.result_json}`);
  console.log(`client_summary_validator_json=${response.evidence.client_summary_validator_json}`);
  console.log(`client_response_validator_json=${response.evidence.client_response_validator_json}`);
  if (response.failures.length > 0) {
    console.log(`failures=${response.failures.join('; ')}`);
  }
}

if (response.status !== 'passed') {
  process.exitCode = 1;
}
