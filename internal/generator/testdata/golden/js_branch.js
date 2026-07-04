function formatText(mode, prefix, text) {
  if (mode === 'short') {
    return prefix;
  }
  return prefix + text;
}

module.exports = { formatText };
