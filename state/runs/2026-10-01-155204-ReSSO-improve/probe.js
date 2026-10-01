const w = require('node:worker_threads')
console.log(process.version, 'markAsUncloneable:', typeof w.markAsUncloneable)
