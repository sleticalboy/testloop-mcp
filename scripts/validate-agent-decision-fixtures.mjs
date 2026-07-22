#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function usage() {
  console.error('Usage: node scripts/validate-agent-decision-fixtures.mjs [--json] [manifest-json] [repo-root]');
  console.error('');
  console.error('Defaults:');
  console.error('  manifest-json: docs/fixtures/agent-decision-fixtures.json');
  console.error('  repo-root:      current working directory');
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

if (positionalArgs.length > 2) {
  usage();
  process.exit(2);
}

const manifestPath = path.resolve(positionalArgs[0] || 'docs/fixtures/agent-decision-fixtures.json');
const repoRoot = path.resolve(positionalArgs[1] || process.cwd());
const failures = [];
const fixtureResults = [];
const decisions = [];
const validFixtureKinds = new Set(['validate_coverage_task', 'real_project_agent_loop']);
const validFixtureSources = new Set(['synthetic', 'real_project']);
const validStatuses = new Set(['passed', 'failed']);
const validActions = new Set([
  'ready',
  'manual_review_internal',
  'manual_review_environment',
  'manual_review_external_service',
  'apply_fix_suggestions',
  'needs_better_input',
]);
const validExpectedDecisions = new Set([
  'accept',
  'manual-review',
  'apply-repair',
  'needs-better-input',
]);

function emitJSON(status, manifest) {
  const fixtures = Array.isArray(manifest?.fixtures) ? manifest.fixtures : [];
  console.log(JSON.stringify({
    status,
    fixture_count: fixtures.length,
    decisions,
    fixtures: fixtureResults,
    failures,
  }, null, 2));
}

function emitFailures() {
  if (jsonMode) {
    emitJSON('failed', manifest);
    return;
  }
  console.error('agent decision fixture validation failed:');
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

function requireOneOf(value, allowed, label) {
  if (typeof value !== 'string' || !allowed.has(value)) {
    failures.push(`${label}: expected one of ${Array.from(allowed).join(', ')}`);
  }
}

function requireNonEmptyString(value, label) {
  if (typeof value !== 'string' || value.length === 0) {
    failures.push(`${label}: expected non-empty string`);
  }
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
  if (metadata.next_action_kind !== 'coverage_target_miss') {
    failures.push(`${fixturePath}: needs_better_input requires metadata.next_action_kind=coverage_target_miss`);
  }
  if (typeof metadata.next_action_reason !== 'string' || metadata.next_action_reason.length === 0) {
    failures.push(`${fixturePath}: needs_better_input requires metadata.next_action_reason`);
  }
  if (typeof metadata.needs_better_input_reason !== 'string' || metadata.needs_better_input_reason.length === 0) {
    failures.push(`${fixturePath}: needs_better_input requires metadata.needs_better_input_reason`);
  }
  if (metadata.next_action_reason !== metadata.needs_better_input_reason) {
    failures.push(`${fixturePath}: next_action_reason must match needs_better_input_reason`);
  }
  if (metadata.coverage_miss_reason && metadata.next_action_reason !== metadata.coverage_miss_reason) {
    failures.push(`${fixturePath}: next_action_reason must match legacy coverage_miss_reason when present`);
  }
}

function validateManualReviewPayload(payload, fixturePath) {
  const metadata = requireObject(payload.metadata, `${fixturePath}: metadata`);
  if (metadata.next_action_kind !== 'manual_review') {
    failures.push(`${fixturePath}: manual_review_* requires metadata.next_action_kind=manual_review`);
  }
  if (typeof metadata.next_action_reason !== 'string' || metadata.next_action_reason.length === 0) {
    failures.push(`${fixturePath}: manual_review_* requires metadata.next_action_reason`);
  }
  if (typeof metadata.manual_review_kind !== 'string' || metadata.manual_review_kind.length === 0) {
    failures.push(`${fixturePath}: manual_review_* requires metadata.manual_review_kind`);
  }
  if (typeof metadata.manual_review_reason !== 'string' || metadata.manual_review_reason.length === 0) {
    failures.push(`${fixturePath}: manual_review_* requires metadata.manual_review_reason`);
  }
  if (metadata.next_action_reason !== metadata.manual_review_reason) {
    failures.push(`${fixturePath}: next_action_reason must match manual_review_reason`);
  }
}

function actionReason(payload) {
  const metadata = payload && typeof payload === 'object' && payload.metadata && typeof payload.metadata === 'object'
    ? payload.metadata
    : {};
  for (const key of [
    'next_action_reason',
    'manual_review_reason',
    'needs_better_input_reason',
    'coverage_miss_reason',
    'external_service_reason',
    'environment_reason',
    'internal_reason',
  ]) {
    const value = metadata[key];
    if (typeof value === 'string' && value.trim().length > 0) {
      return value.trim();
    }
  }
  return '';
}

let manifest;
try {
  manifest = readJSON(manifestPath);
} catch (error) {
  failures.push(error.message);
  emitFailures();
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
for (const [index, item] of (manifest.fixtures || []).entries()) {
  const label = `${manifestPath}: fixtures[${index}]`;
  const entry = requireObject(item, label);
  const fixtureRelPath = entry.path;
  const result = {
    path: fixtureRelPath,
    kind: entry.kind,
    source: entry.source,
    manifest_status: entry.status,
    manifest_action: entry.action,
    expected_decision: entry.expected_decision,
  };
  requireOneOf(entry.kind, validFixtureKinds, `${label}: kind`);
  requireOneOf(entry.source, validFixtureSources, `${label}: source`);
  requireOneOf(entry.status, validStatuses, `${label}: status`);
  requireOneOf(entry.action, validActions, `${label}: action`);
  requireOneOf(entry.expected_decision, validExpectedDecisions, `${label}: expected_decision`);
  requireNonEmptyString(entry.client_expectation, `${label}: client_expectation`);
  if (typeof fixtureRelPath !== 'string' || fixtureRelPath.length === 0) {
    failures.push(`${label}: path must be a non-empty string`);
    result.error = 'path must be a non-empty string';
    fixtureResults.push(result);
    continue;
  }
  if (seenPaths.has(fixtureRelPath)) {
    failures.push(`${label}: duplicate fixture path ${fixtureRelPath}`);
  }
  seenPaths.add(fixtureRelPath);

  const fixturePath = path.resolve(repoRoot, fixtureRelPath);
  if (!fs.existsSync(fixturePath)) {
    failures.push(`${fixturePath}: fixture file does not exist`);
    result.error = 'fixture file does not exist';
    fixtureResults.push(result);
    continue;
  }

  let payload;
  try {
    payload = readJSON(fixturePath);
  } catch (error) {
    failures.push(error.message);
    result.error = error.message;
    fixtureResults.push(result);
    continue;
  }
  const status = payload.status;
  const action = payload.action;
  result.status = status;
  result.action = action;
  if (status !== entry.status || action !== entry.action) {
    failures.push(`${fixtureRelPath}: status/action=${status}/${action}, manifest=${entry.status}/${entry.action}`);
  }

  const decision = decisionFor(status, action);
  result.decision = decision;
  const reason = actionReason(payload);
  if (reason) {
    result.reason = reason;
  }
  decisions.push(decision);
  if (decision !== entry.expected_decision) {
    failures.push(`${fixtureRelPath}: decision=${decision}, expected=${entry.expected_decision}`);
  }
  fixtureResults.push(result);

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
  if (entry.kind === 'validate_coverage_task' && typeof action === 'string' && action.startsWith('manual_review_')) {
    validateManualReviewPayload(payload, fixtureRelPath);
  }
}

if (failures.length > 0) {
  emitFailures();
  process.exit(1);
}

if (jsonMode) {
  emitJSON('passed', manifest);
} else {
  console.log(`agent_decision_fixture_status=passed fixture_count=${manifest.fixtures.length}`);
  console.log(`agent_decision_fixture_decisions=${decisions.join(',')}`);
}
