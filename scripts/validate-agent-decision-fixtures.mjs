#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function usage() {
  console.error('Usage: node scripts/validate-agent-decision-fixtures.mjs [manifest-json] [repo-root]');
  console.error('');
  console.error('Defaults:');
  console.error('  manifest-json: docs/fixtures/agent-decision-fixtures.json');
  console.error('  repo-root:      current working directory');
}

if (process.argv.includes('-h') || process.argv.includes('--help')) {
  usage();
  process.exit(0);
}

if (process.argv.length > 4) {
  usage();
  process.exit(2);
}

const manifestPath = path.resolve(process.argv[2] || 'docs/fixtures/agent-decision-fixtures.json');
const repoRoot = path.resolve(process.argv[3] || process.cwd());
const failures = [];

function readJSON(filePath) {
  try {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
  } catch (error) {
    throw new Error(`${filePath}: ${error.message}`);
  }
}

function hasKey(value, key) {
  if (Array.isArray(value)) {
    return value.some((item) => hasKey(item, key));
  }
  if (value && typeof value === 'object') {
    return Object.prototype.hasOwnProperty.call(value, key) ||
      Object.values(value).some((item) => hasKey(item, key));
  }
  return false;
}

function decisionFor(status, action) {
  if (status === 'passed' && action === 'ready') {
    return 'accept';
  }
  if (typeof action === 'string' && action.startsWith('manual_review_')) {
    return 'manual-review';
  }
  if (action === 'apply_fix_suggestions') {
    return 'apply-repair';
  }
  if (action === 'needs_better_input') {
    return 'needs-better-input';
  }
  if (status === 'generation_error') {
    return 'inspect-generation';
  }
  if (status === 'run_error') {
    return 'inspect-runner';
  }
  if (status === 'failed') {
    return 'repair-generated-test';
  }
  return 'inspect';
}

function requireObject(value, label) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    failures.push(`${label}: expected object`);
    return {};
  }
  return value;
}

function validateRepairPayload(payload, fixturePath) {
  const runResult = requireObject(payload.run_result, `${fixturePath}: run_result`);
  const suggestions = runResult.fix_suggestions;
  if (!Array.isArray(suggestions) || suggestions.length === 0) {
    failures.push(`${fixturePath}: apply_fix_suggestions requires run_result.fix_suggestions[]`);
    return;
  }
  if (!suggestions.some((suggestion) => suggestion && typeof suggestion === 'object' && suggestion.repair_task)) {
    failures.push(`${fixturePath}: apply_fix_suggestions requires at least one repair_task`);
  }
}

function validateNeedsBetterInput(payload, fixturePath) {
  const metadata = requireObject(payload.metadata, `${fixturePath}: metadata`);
  if (typeof metadata.coverage_miss_reason !== 'string' || metadata.coverage_miss_reason.length === 0) {
    failures.push(`${fixturePath}: needs_better_input requires metadata.coverage_miss_reason`);
  }
}

let manifest;
try {
  manifest = readJSON(manifestPath);
} catch (error) {
  console.error('agent decision fixture validation failed:');
  console.error(`- ${error.message}`);
  process.exit(1);
}
if (manifest.$schema !== './agent-decision-fixtures.schema.json') {
  failures.push(`${manifestPath}: $schema must be ./agent-decision-fixtures.schema.json`);
}
if (manifest.schema_version !== 1) {
  failures.push(`${manifestPath}: schema_version must be 1`);
}
if (!Array.isArray(manifest.fixtures) || manifest.fixtures.length === 0) {
  failures.push(`${manifestPath}: fixtures must be a non-empty array`);
}

const seenPaths = new Set();
const decisions = [];
for (const [index, item] of (manifest.fixtures || []).entries()) {
  const label = `${manifestPath}: fixtures[${index}]`;
  const entry = requireObject(item, label);
  const fixtureRelPath = entry.path;
  if (typeof fixtureRelPath !== 'string' || fixtureRelPath.length === 0) {
    failures.push(`${label}: path must be a non-empty string`);
    continue;
  }
  if (seenPaths.has(fixtureRelPath)) {
    failures.push(`${label}: duplicate fixture path ${fixtureRelPath}`);
  }
  seenPaths.add(fixtureRelPath);

  const fixturePath = path.resolve(repoRoot, fixtureRelPath);
  if (!fs.existsSync(fixturePath)) {
    failures.push(`${fixturePath}: fixture file does not exist`);
    continue;
  }

  let payload;
  try {
    payload = readJSON(fixturePath);
  } catch (error) {
    failures.push(error.message);
    continue;
  }
  const status = payload.status;
  const action = payload.action;
  if (status !== entry.status || action !== entry.action) {
    failures.push(`${fixtureRelPath}: status/action=${status}/${action}, manifest=${entry.status}/${entry.action}`);
  }

  const decision = decisionFor(status, action);
  decisions.push(decision);
  if (decision !== entry.expected_decision) {
    failures.push(`${fixtureRelPath}: decision=${decision}, expected=${entry.expected_decision}`);
  }

  if (entry.kind === 'real_project_agent_loop' && hasKey(payload, 'raw_output')) {
    failures.push(`${fixtureRelPath}: real project fixture must not contain raw_output`);
  }
  if (action === 'apply_fix_suggestions') {
    validateRepairPayload(payload, fixtureRelPath);
  }
  if (action === 'needs_better_input') {
    validateNeedsBetterInput(payload, fixtureRelPath);
  }
  if (typeof action === 'string' && action.startsWith('manual_review_') && decision !== 'manual-review') {
    failures.push(`${fixtureRelPath}: manual_review_* must map to manual-review`);
  }
}

if (failures.length > 0) {
  console.error('agent decision fixture validation failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exit(1);
}

console.log(`agent_decision_fixture_status=passed fixture_count=${manifest.fixtures.length}`);
console.log(`agent_decision_fixture_decisions=${decisions.join(',')}`);
