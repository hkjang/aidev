// Standalone reproduction of quota.ts retryAfterSeconds, current vs fixed.
// Run: node /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-15-141846-biz.irumx.app-improve/verify-retry.mjs
const KST_OFFSET_MINUTES = 9 * 60
const MINUTE = 60_000

const kstParts = (at) => {
  const s = new Date(at.getTime() + KST_OFFSET_MINUTES * MINUTE)
  return { y: s.getUTCFullYear(), m: s.getUTCMonth() + 1, d: s.getUTCDate() }
}
const kstDateKey = (at) => {
  const p = kstParts(at)
  return `${p.y}-${String(p.m).padStart(2, '0')}-${String(p.d).padStart(2, '0')}`
}
const minuteStart = (at) => new Date(Math.floor(at.getTime() / MINUTE) * MINUTE)

function windowStart(window, at) {
  if (window === 'minute') return minuteStart(at)
  if (window === 'month') return new Date(`${kstDateKey(at).slice(0, 7)}-01T00:00:00.000Z`)
  return new Date(`${kstDateKey(at)}T00:00:00.000Z`)
}
const windowOpensAt = (window, at) =>
  new Date(windowStart(window, at).getTime() - KST_OFFSET_MINUTES * 60_000)

function retryCurrent(window, at) {
  if (window === 'minute') return Math.max(1, 60 - Math.floor((at.getTime() % 60_000) / 1000))
  if (window === 'month') {
    let probe = new Date(at.getTime())
    const thisMonth = windowStart('month', at).getTime()
    while (windowStart('month', probe).getTime() === thisMonth) probe = new Date(probe.getTime() + 86_400_000)
    return Math.max(1, Math.ceil((windowStart('month', probe).getTime() - at.getTime()) / 1000))
  }
  const next = windowStart('day', new Date(at.getTime() + 86_400_000))
  return Math.max(1, Math.ceil((next.getTime() - at.getTime()) / 1000))
}

function retryFixed(window, at) {
  if (window === 'minute') return Math.max(1, 60 - Math.floor((at.getTime() % 60_000) / 1000))
  if (window === 'month') {
    let probe = new Date(at.getTime())
    const thisMonth = windowStart('month', at).getTime()
    while (windowStart('month', probe).getTime() === thisMonth) probe = new Date(probe.getTime() + 86_400_000)
    return Math.max(1, Math.ceil((windowOpensAt('month', probe).getTime() - at.getTime()) / 1000))
  }
  const next = windowOpensAt('day', new Date(at.getTime() + 86_400_000))
  return Math.max(1, Math.ceil((next.getTime() - at.getTime()) / 1000))
}

const cases = [
  ['day', '2026-03-15T00:00:00.000Z'],
  ['day', '2026-03-14T15:00:00.000Z'],
  ['day', '2026-03-15T14:59:59.000Z'],
  ['month', '2026-03-31T15:00:00.000Z'],
  ['month', '2026-01-31T14:59:59.000Z'],
  ['month', '2026-02-28T15:00:00.000Z'],
  ['minute', '2026-03-15T00:00:00.000Z'],
  ['minute', '2026-03-15T00:00:30.500Z'],
  ['minute', '2026-03-15T00:00:59.000Z'],
]
for (const [w, iso] of cases) {
  const at = new Date(iso)
  console.log(w.padEnd(6), iso, 'current=', String(retryCurrent(w, at)).padStart(8), 'fixed=', String(retryFixed(w, at)).padStart(8))
}

// Invariant check on the fixed version
const probes = [
  '2026-03-15T00:00:00.000Z', '2026-03-14T15:00:00.000Z', '2026-03-15T14:59:59.000Z',
  '2026-12-31T15:00:00.000Z', '2026-02-28T15:00:00.000Z', '2026-01-31T14:59:59.000Z',
  '2026-07-04T06:23:17.000Z',
]
const keyOf = { day: (d) => kstDateKey(d), month: (d) => kstDateKey(d).slice(0, 7), minute: (d) => minuteStart(d).toISOString() }
for (const fn of [['current', retryCurrent], ['fixed', retryFixed]]) {
  for (const w of ['day', 'month', 'minute']) {
    for (const iso of probes) {
      const at = new Date(iso)
      const r = fn[1](w, at)
      const after = keyOf[w](new Date(at.getTime() + r * 1000)) !== keyOf[w](at)
      const tight = keyOf[w](new Date(at.getTime() + (r - 1) * 1000)) === keyOf[w](at)
      if (!after || !tight) console.log(`  INVARIANT FAIL ${fn[0]} ${w} ${iso} r=${r} after=${after} tight=${tight}`)
    }
  }
}
console.log('invariant scan done')
