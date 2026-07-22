#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function usage() {
  console.error('Usage: node scripts/read-testloop-release-response-summary.mjs [--json] [summary-json]');
  console.error('');
  console.error('Defaults:');
  console.error('  summary-json: testloop-release-response-adopter-summary.json');
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
  positionalArgs[0] || 'testloop-release-response-adopter-summary.json',
);

const actionHints = {
  ready: 'accept release response adopter summary',
  'inspect-release-installer': 'check release ref, installer URL and helper refs',
  'inspect-release-client-response': 'inspect base client Agent response',
  'inspect-release-consumer-response': 'inspect consumer smoke Agent response',
  'inspect-agent-decision-fixtures': 'inspect fixture count or decision sequence drift',
  'inspect-release-smoke-summary': 'inspect release response adopter summary and failures',
};

function readJSON(filePath) {
  try {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
  } catch (error) {
    throw new Error(`${filePath}: ${error.message}`);
  }
}

function normalize(payload) {
  const agentNextStep = typeof payload.agent_next_step === 'string' && payload.agent_next_step.length > 0
    ? payload.agent_next_step
    : 'inspect-release-smoke-summary';
  const status = typeof payload.status === 'string' ? payload.status : 'failed';

  return {
    schema_version: 1,
    status,
    agent_next_step: agentNextStep,
    should_accept: status === 'passed' && agentNextStep === 'ready' && payload.should_accept === true,
    action_hint: actionHints[agentNextStep] || 'inspect release response adopter summary',
    summary_json: summaryPath,
    release_ref: typeof payload.release_ref === 'string' ? payload.release_ref : '',
    fixture_count: Number.isInteger(payload.fixture_count) ? payload.fixture_count : 0,
    npm_exit_code: Number.isInteger(payload.npm_exit_code) ? payload.npm_exit_code : null,
    failures: Array.isArray(payload.failures) ? payload.failures : [],
  };
}

let output;
try {
  output = normalize(readJSON(summaryPath));
} catch (error) {
  output = {
    schema_version: 1,
    status: 'failed',
    agent_next_step: 'inspect-release-smoke-summary',
    should_accept: false,
    action_hint: actionHints['inspect-release-smoke-summary'],
    summary_json: summaryPath,
    release_ref: '',
    fixture_count: 0,
    npm_exit_code: null,
    failures: [error.message],
  };
}

if (jsonMode) {
  console.log(JSON.stringify(output, null, 2));
} else {
  console.log(`testloop_release_response_summary_status=${output.status}`);
  console.log(`testloop_release_response_summary_next_step=${output.agent_next_step}`);
  console.log(`testloop_release_response_summary_release_ref=${output.release_ref}`);
  console.log(`testloop_release_response_summary_fixture_count=${output.fixture_count}`);
  console.log(`testloop_release_response_summary_should_accept=${output.should_accept}`);
  console.log(`testloop_release_response_summary_action_hint=${output.action_hint}`);
  if (output.failures.length > 0) {
    console.log(`testloop_release_response_summary_failures=${output.failures.join('; ')}`);
  }
}

if (!output.should_accept) {
  process.exitCode = 1;
}
