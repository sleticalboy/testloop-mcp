async function fetchData(url) {
  if (url === undefined) {
    throw new Error('missing url');
  }
  return { url };
}

class Widget {
  load(mode, count) {
    if (mode === 'short') {
      return count;
    }
    return count + 1;
  }

  async save(payload) {
    if (payload === undefined) {
      throw new Error('missing payload');
    }
    return { ok: true };
  }
}

module.exports = { fetchData, Widget };
