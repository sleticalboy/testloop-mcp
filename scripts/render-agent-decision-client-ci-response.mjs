#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function usage() {
  console.error('Usage: node scripts/render-agent-decision-client-ci-response.mjs [--json] <client-ci-summary-json>');
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

const summaryPath = path.resolve(positionalArgs[0]);
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

function decide(summary) {
  const failures = [];
  const evidence = {
    client_dir: typeof summary.client_dir === 'string' ? summary.client_dir : '',
    fixture_dir: typeof summary.fixture_dir === 'string' ? summary.fixture_dir : '',
    result_json: typeof summary.result_json === 'string' ? summary.result_json : '',
    fixture_count: Number.isInteger(summary.fixture_count) ? summary.fixture_count : 0,
    decisions: Array.isArray(summary.decisions) ? summary.decisions : [],
    validator_exit_code: Number.isInteger(summary.validator_exit_code) ? summary.validator_exit_code : -1,
  };

  if (summary.status !== 'passed') {
    failures.push(`summary status is ${summary.status || 'missing'}`);
  }
  if (summary.validator_exit_code !== 0) {
    failures.push(`validator_exit_code=${summary.validator_exit_code}`);
  }
  if (!Array.isArray(summary.failures)) {
    failures.push('failures must be an array');
  } else if (summary.failures.length > 0) {
    failures.push(...summary.failures.map((failure) => `client fixture failure: ${failure}`));
  }
  if (summary.fixture_count !== expectedDecisions.length) {
    failures.push(`fixture_count=${summary.fixture_count}, want ${expectedDecisions.length}`);
  }
  if (JSON.stringify(summary.decisions) !== JSON.stringify(expectedDecisions)) {
    failures.push(`decisions must be ${expectedDecisions.join(',')}`);
  }
  for (const field of ['client_dir', 'fixture_dir', 'result_json']) {
    if (typeof summary[field] !== 'string' || summary[field].length === 0) {
      failures.push(`${field} is required`);
    }
  }

  let agentNextStep = 'ready';
  if (failures.some((failure) => failure.includes('validator_exit_code'))) {
    agentNextStep = 'inspect-client-validator';
  } else if (failures.some((failure) => failure.includes('fixture_count') || failure.includes('decisions') || failure.includes('client fixture failure'))) {
    agentNextStep = 'inspect-agent-decision-fixtures';
  } else if (failures.length > 0) {
    agentNextStep = 'inspect-agent-decision-client-summary';
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
    agent_next_step: 'inspect-agent-decision-client-summary',
    summary_json: summaryPath,
    evidence: {
      client_dir: '',
      fixture_dir: '',
      result_json: '',
      fixture_count: 0,
      decisions: [],
      validator_exit_code: -1,
    },
    failures: [error.message],
  };
}

if (jsonMode) {
  console.log(JSON.stringify(response, null, 2));
} else {
  console.log(`agent_decision_client_response_status=${response.status}`);
  console.log(`agent_next_step=${response.agent_next_step}`);
  console.log(`fixture_count=${response.evidence.fixture_count}`);
  console.log(`decisions=${response.evidence.decisions.join(',')}`);
  console.log(`validator_exit_code=${response.evidence.validator_exit_code}`);
  console.log(`result_json=${response.evidence.result_json}`);
  if (response.failures.length > 0) {
    console.log(`failures=${response.failures.join('; ')}`);
  }
}

if (response.status !== 'passed') {
  process.exitCode = 1;
}
