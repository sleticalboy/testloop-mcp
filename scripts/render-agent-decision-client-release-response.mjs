#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function usage() {
  console.error('Usage: node scripts/render-agent-decision-client-release-response.mjs [--json] [release-smoke-summary-json]');
  console.error('');
  console.error('Defaults:');
  console.error('  release-smoke-summary-json: docs/fixtures/agent-decision-client-release-smoke-summary/passed.json');
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
  positionalArgs[0] || 'docs/fixtures/agent-decision-client-release-smoke-summary/passed.json',
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
  const helperRefs = summary.helper_refs && typeof summary.helper_refs === 'object'
    ? summary.helper_refs
    : {};
  const agentNextSteps = summary.agent_next_steps && typeof summary.agent_next_steps === 'object'
    ? summary.agent_next_steps
    : {};
  const evidence = {
    release_ref: nonEmptyString(summary.release_ref) ? summary.release_ref : '',
    installer_url: nonEmptyString(summary.installer_url) ? summary.installer_url : '',
    helper_refs: {
      install: nonEmptyString(helperRefs.install) ? helperRefs.install : '',
      consumer: nonEmptyString(helperRefs.consumer) ? helperRefs.consumer : '',
    },
    fixture_count: Number.isInteger(summary.fixture_count) ? summary.fixture_count : 0,
    decisions: Array.isArray(summary.decisions) ? summary.decisions : [],
    agent_next_steps: {
      client: nonEmptyString(agentNextSteps.client) ? agentNextSteps.client : '',
      consumer: nonEmptyString(agentNextSteps.consumer) ? agentNextSteps.consumer : '',
    },
  };

  if (summary.schema_version !== 1) {
    failures.push('schema_version must be 1');
  }
  if (!nonEmptyString(summary.release_ref)) {
    failures.push('release_ref is required');
  }
  if (!nonEmptyString(summary.installer_url)) {
    failures.push('installer_url is required');
  }
  if (
    typeof summary.installer_url === 'string'
    && /^https?:\/\//.test(summary.installer_url)
    && !summary.installer_url.includes(summary.release_ref || '')
  ) {
    failures.push('installer_url must include release_ref');
  }
  if (summary.status !== 'passed') {
    failures.push(`summary status is ${summary.status || 'missing'}`);
  }
  if (!Array.isArray(summary.failures)) {
    failures.push('failures must be an array');
  } else if (summary.failures.length > 0) {
    failures.push(...summary.failures.map((failure) => `release smoke failure: ${failure}`));
  }
  if (helperRefs.install !== summary.release_ref) {
    failures.push(`helper_refs.install=${helperRefs.install || 'missing'}, want ${summary.release_ref || 'release_ref'}`);
  }
  if (helperRefs.consumer !== summary.release_ref) {
    failures.push(`helper_refs.consumer=${helperRefs.consumer || 'missing'}, want ${summary.release_ref || 'release_ref'}`);
  }
  if (summary.fixture_count !== expectedDecisions.length) {
    failures.push(`fixture_count=${summary.fixture_count}, want ${expectedDecisions.length}`);
  }
  if (JSON.stringify(summary.decisions) !== JSON.stringify(expectedDecisions)) {
    failures.push(`decisions must be ${expectedDecisions.join(',')}`);
  }
  if (agentNextSteps.client !== 'ready') {
    failures.push(`agent_next_steps.client=${agentNextSteps.client || 'missing'}, want ready`);
  }
  if (agentNextSteps.consumer !== 'ready') {
    failures.push(`agent_next_steps.consumer=${agentNextSteps.consumer || 'missing'}, want ready`);
  }

  let agentNextStep = 'ready';
  if (failures.some((failure) => failure.includes('installer_url') || failure.includes('helper_refs'))) {
    agentNextStep = 'inspect-release-installer';
  } else if (failures.some((failure) => failure.includes('agent_next_steps.client'))) {
    agentNextStep = 'inspect-release-client-response';
  } else if (failures.some((failure) => failure.includes('agent_next_steps.consumer'))) {
    agentNextStep = 'inspect-release-consumer-response';
  } else if (failures.some((failure) => failure.includes('fixture_count') || failure.includes('decisions'))) {
    agentNextStep = 'inspect-agent-decision-fixtures';
  } else if (failures.length > 0) {
    agentNextStep = 'inspect-release-smoke-summary';
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
    agent_next_step: 'inspect-release-smoke-summary',
    summary_json: summaryPath,
    evidence: {
      release_ref: '',
      installer_url: '',
      helper_refs: {
        install: '',
        consumer: '',
      },
      fixture_count: 0,
      decisions: [],
      agent_next_steps: {
        client: '',
        consumer: '',
      },
    },
    failures: [error.message],
  };
}

if (jsonMode) {
  console.log(JSON.stringify(response, null, 2));
} else {
  console.log(`agent_decision_client_release_response_status=${response.status}`);
  console.log(`agent_next_step=${response.agent_next_step}`);
  console.log(`release_ref=${response.evidence.release_ref}`);
  console.log(`installer_url=${response.evidence.installer_url}`);
  console.log(`helper_refs=${response.evidence.helper_refs.install},${response.evidence.helper_refs.consumer}`);
  console.log(`fixture_count=${response.evidence.fixture_count}`);
  console.log(`decisions=${response.evidence.decisions.join(',')}`);
  console.log(`agent_next_steps=${response.evidence.agent_next_steps.client},${response.evidence.agent_next_steps.consumer}`);
  if (response.failures.length > 0) {
    console.log(`failures=${response.failures.join('; ')}`);
  }
}

if (response.status !== 'passed') {
  process.exitCode = 1;
}
