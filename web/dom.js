export function createDomCache(doc = document) {
  const cache = new Map();

  return function el(id) {
    const cached = cache.get(id);
    if (cached && cached.isConnected) {
      return cached;
    }
    const node = doc.getElementById(id);
    if (node) {
      cache.set(id, node);
    } else {
      cache.delete(id);
    }
    return node;
  };
}
