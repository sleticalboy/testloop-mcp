#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function usage() {
  console.error('Usage: node scripts/read-testloop-agent-decision-response.mjs [--json] [response-json]');
  console.error('');
  console.error('Defaults:');
  console.error('  response-json: testloop-agent-decision-client-response.json');
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
  positionalArgs[0] || 'testloop-agent-decision-client-response.json',
);

const actionHints = {
  ready: 'accept Agent decision client response',
  'inspect-client-validator': 'inspect exported fixture validator and exit code',
  'inspect-agent-decision-fixtures': 'inspect fixture count or decision sequence drift',
  'inspect-agent-decision-client-summary': 'inspect missing, invalid or incompatible client CI summary',
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
    : 'inspect-agent-decision-client-summary';
  const status = typeof payload.status === 'string' ? payload.status : 'failed';

  return {
    schema_version: 1,
    status,
    agent_next_step: agentNextStep,
    should_accept: status === 'passed' && agentNextStep === 'ready',
    action_hint: actionHints[agentNextStep] || 'inspect Agent decision client response',
    response_json: responsePath,
    evidence: {
      fixture_count: Number.isInteger(evidence.fixture_count) ? evidence.fixture_count : 0,
      decisions: Array.isArray(evidence.decisions) ? evidence.decisions : [],
      validator_exit_code: Number.isInteger(evidence.validator_exit_code) ? evidence.validator_exit_code : null,
      result_json: typeof evidence.result_json === 'string' ? evidence.result_json : '',
      result_schema: typeof evidence.result_schema === 'string' ? evidence.result_schema : '',
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
    agent_next_step: 'inspect-agent-decision-client-summary',
    should_accept: false,
    action_hint: actionHints['inspect-agent-decision-client-summary'],
    response_json: responsePath,
    evidence: {
      fixture_count: 0,
      decisions: [],
      validator_exit_code: null,
      result_json: '',
      result_schema: '',
    },
    failures: [error.message],
  };
}

if (jsonMode) {
  console.log(JSON.stringify(output, null, 2));
} else {
  console.log(`testloop_agent_decision_response_status=${output.status}`);
  console.log(`testloop_agent_decision_response_next_step=${output.agent_next_step}`);
  console.log(`testloop_agent_decision_response_fixture_count=${output.evidence.fixture_count}`);
  console.log(`testloop_agent_decision_response_validator_exit_code=${output.evidence.validator_exit_code}`);
  console.log(`testloop_agent_decision_response_should_accept=${output.should_accept}`);
  console.log(`testloop_agent_decision_response_action_hint=${output.action_hint}`);
  if (output.failures.length > 0) {
    console.log(`testloop_agent_decision_response_failures=${output.failures.join('; ')}`);
  }
}

if (!output.should_accept) {
  process.exitCode = 1;
}
