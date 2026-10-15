const KST = 9 * 60, MINUTE = 60000
const kstParts = (at) => { const s = new Date(at.getTime() + KST * MINUTE); return { y: s.getUTCFullYear(), m: s.getUTCMonth() + 1, d: s.getUTCDate() } }
const kstDateKey = (at) => { const p = kstParts(at); return p.y + '-' + String(p.m).padStart(2, '0') + '-' + String(p.d).padStart(2, '0') }
const minuteStart = (at) => new Date(Math.floor(at.getTime() / MINUTE) * MINUTE)
const windowStart = (w, at) => {
  if (w === 'minute') return minuteStart(at)
  if (w === 'month') return new Date(kstDateKey(at).slice(0, 7) + '-01T00:00:00.000Z')
  return new Date(kstDateKey(at) + 'T00:00:00.000Z')
}
function cur(w, at) {
  if (w === 'minute') return Math.max(1, 60 - Math.floor((at.getTime() % 60000) / 1000))
  if (w === 'month') {
    let p = new Date(at.getTime()); const t = windowStart('month', at).getTime()
    while (windowStart('month', p).getTime() === t) p = new Date(p.getTime() + 86400000)
    return Math.max(1, Math.ceil((windowStart('month', p).getTime() - at.getTime()) / 1000))
  }
  const n = windowStart('day', new Date(at.getTime() + 86400000))
  return Math.max(1, Math.ceil((n.getTime() - at.getTime()) / 1000))
}
const opens = (w, at) => new Date(windowStart(w, at).getTime() - KST * MINUTE)
function fix(w, at) {
  if (w === 'minute') return cur('minute', at)
  if (w === 'month') {
    let p = new Date(at.getTime()); const t = windowStart('month', at).getTime()
    while (windowStart('month', p).getTime() === t) p = new Date(p.getTime() + 86400000)
    return Math.max(1, Math.ceil((opens('month', p).getTime() - at.getTime()) / 1000))
  }
  const n = opens('day', new Date(at.getTime() + 86400000))
  return Math.max(1, Math.ceil((n.getTime() - at.getTime()) / 1000))
}
const cases = [
  ['day', '2026-03-15T00:00:00.000Z'], ['day', '2026-03-14T15:00:00.000Z'], ['day', '2026-03-15T14:59:59.000Z'],
  ['month', '2026-03-31T15:00:00.000Z'], ['month', '2026-01-31T14:59:59.000Z'], ['month', '2026-02-28T15:00:00.000Z'],
  ['minute', '2026-03-15T00:00:00.000Z'], ['minute', '2026-03-15T00:00:59.200Z'],
]
const key = (w, x) => w === 'minute' ? minuteStart(x).toISOString() : (w === 'month' ? kstDateKey(x).slice(0, 7) : kstDateKey(x))
for (const [w, iso] of cases) { const at = new Date(iso); console.log(w, iso, 'cur=', cur(w, at), 'fix=', fix(w, at), 'diff=', cur(w, at) - fix(w, at)) }
console.log('--- invariant with fix ---')
for (const [w, iso] of cases) {
  const at = new Date(iso), r = fix(w, at), base = key(w, at)
  console.log(w, iso, 'rolled=', key(w, new Date(at.getTime() + r * 1000)) !== base, 'tight=', key(w, new Date(at.getTime() + (r - 1) * 1000)) === base)
}
console.log('--- invariant today (current code) ---')
for (const [w, iso] of cases) {
  const at = new Date(iso), r = cur(w, at), base = key(w, at)
  console.log(w, iso, 'rolled=', key(w, new Date(at.getTime() + r * 1000)) !== base, 'tight=', key(w, new Date(at.getTime() + (r - 1) * 1000)) === base)
}
