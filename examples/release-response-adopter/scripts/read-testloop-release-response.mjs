#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function usage() {
  console.error('Usage: node scripts/read-testloop-release-response.mjs [--json] [response-json]');
  console.error('');
  console.error('Defaults:');
  console.error('  response-json: testloop-release-response-client/testloop-release-response.json');
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
  positionalArgs[0] || 'testloop-release-response-client/testloop-release-response.json',
);

const actionHints = {
  ready: 'accept release response contract result',
  'inspect-release-installer': 'check release ref, installer URL and helper refs',
  'inspect-release-client-response': 'inspect base client Agent response',
  'inspect-release-consumer-response': 'inspect consumer smoke Agent response',
  'inspect-agent-decision-fixtures': 'inspect fixture count or decision sequence drift',
  'inspect-release-smoke-summary': 'inspect missing, invalid or incompatible release smoke summary',
};

function readJSON(filePath) {
  try {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
  } catch (error) {
    throw new Error(`${filePath}: ${error.message}`);
  }
}

function normalize(payload) {
  const evidence = payload && typeof payload.evidence === 'object' && payload.evidence !== null
    ? payload.evidence
    : {};
  const agentNextStep = typeof payload.agent_next_step === 'string' && payload.agent_next_step.length > 0
    ? payload.agent_next_step
    : 'inspect-release-smoke-summary';

  return {
    schema_version: 1,
    status: typeof payload.status === 'string' ? payload.status : 'failed',
    agent_next_step: agentNextStep,
    should_accept: payload.status === 'passed' && agentNextStep === 'ready',
    action_hint: actionHints[agentNextStep] || 'inspect release response JSON',
    response_json: responsePath,
    evidence: {
      release_ref: typeof evidence.release_ref === 'string' ? evidence.release_ref : '',
      fixture_count: Number.isInteger(evidence.fixture_count) ? evidence.fixture_count : 0,
      decisions: Array.isArray(evidence.decisions) ? evidence.decisions : [],
      agent_next_steps: evidence.agent_next_steps && typeof evidence.agent_next_steps === 'object'
        ? evidence.agent_next_steps
        : {},
    },
    failures: Array.isArray(payload.failures) ? payload.failures : [],
  };
}

let output;
try {
  output = normalize(readJSON(responsePath));
} catch (error) {
  output = {
    schema_version: 1,
    status: 'failed',
    agent_next_step: 'inspect-release-smoke-summary',
    should_accept: false,
    action_hint: actionHints['inspect-release-smoke-summary'],
    response_json: responsePath,
    evidence: {
      release_ref: '',
      fixture_count: 0,
      decisions: [],
      agent_next_steps: {},
    },
    failures: [error.message],
  };
}

if (jsonMode) {
  console.log(JSON.stringify(output, null, 2));
} else {
  console.log(`testloop_release_response_status=${output.status}`);
  console.log(`testloop_release_response_next_step=${output.agent_next_step}`);
  console.log(`testloop_release_response_release_ref=${output.evidence.release_ref}`);
  console.log(`testloop_release_response_fixture_count=${output.evidence.fixture_count}`);
  console.log(`testloop_release_response_should_accept=${output.should_accept}`);
  console.log(`testloop_release_response_action_hint=${output.action_hint}`);
  if (output.failures.length > 0) {
    console.log(`testloop_release_response_failures=${output.failures.join('; ')}`);
  }
}

if (output.failures.length > 0 && output.status === 'failed' && output.evidence.release_ref === '') {
  process.exitCode = 1;
}
