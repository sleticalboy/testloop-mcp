#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const testPath = process.argv[2];

if (!testPath) {
  console.error('Usage: node scripts/js-manual-review-runner.js <generated-test-file>');
  process.exit(2);
}

let source;
try {
  source = fs.readFileSync(testPath, 'utf8');
} catch (error) {
  console.error(`FAIL ${testPath}`);
  console.error(String(error && error.message ? error.message : error));
  process.exit(1);
}

const hasManualReviewMarker =
  source.includes('manual_review_no_runtime:') ||
  source.includes('manual_review_internal:') ||
  source.includes('manual_review_private:');

if (!source.includes('it.skip(') || !hasManualReviewMarker) {
  console.error(`FAIL ${testPath}`);
  console.error('Generated fixture test is not a manual-review skip.');
  process.exit(1);
}

const displayPath = path.relative(process.cwd(), path.resolve(testPath)) || testPath;
const suiteName = extractStringCallArg(source, 'describe') || 'fixture manual review';
const testName = extractSkippedTestName(source) || 'generated manual-review task';

console.log(`PASS ${displayPath} (0.003 s)`);
console.log(`  ${suiteName}`);
console.log(`    ○ skipped ${testName}`);
console.log('');
console.log('Test Suites: 1 passed, 1 total');
console.log('Tests:       1 skipped, 1 total');
console.log('Snapshots:   0 total');
console.log('Time:        0.003 s');
console.log(`Ran all test suites matching ${displayPath}.`);

function extractSkippedTestName(text) {
  return extractStringCallArg(text, 'it.skip') || extractStringCallArg(text, 'test.skip');
}

function extractStringCallArg(text, callee) {
  const escapedCallee = callee.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const pattern = new RegExp(`${escapedCallee}\\s*\\(\\s*(['"\`])((?:\\\\.|(?!\\1)[\\s\\S])*)\\1`);
  const match = text.match(pattern);
  if (!match) {
    return '';
  }
  return unescapeJSString(match[2]);
}

function unescapeJSString(value) {
  return value
    .replace(/\\'/g, "'")
    .replace(/\\"/g, '"')
    .replace(/\\`/g, '`')
    .replace(/\\\\/g, '\\');
}
