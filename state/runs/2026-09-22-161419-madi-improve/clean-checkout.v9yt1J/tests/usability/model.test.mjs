import assert from "node:assert/strict";
import { protocolVersion, validateRecord, summarize } from "./model.mjs";
const base = {
  protocol: protocolVersion,
  measurement_type: "automated_probe",
  session_id: "TEST-SYNTHETIC",
  participant_id: "SYNTHETIC-001",
  cohort: "new",
  build: "fixture-A",
  task_id: "share",
  attempt: 1,
  outcome: "completed",
  elapsed_ms: 1000,
  assistance_count: 0,
  undo_count: 0,
  sharing_misunderstanding: "not_assessed",
  viewport: "desktop",
  observed_at: "2026-09-09T00:00:00.000Z",
  consent: false,
  observer_confirmed: false,
};
assert.equal(summarize([]).human_observation_records, 0);
assert.equal(summarize([]).groups.length, 0);
const result = summarize([
  base,
  {
    ...base,
    participant_id: "SYNTHETIC-002",
    outcome: "abandoned",
    elapsed_ms: 3000,
    sharing_misunderstanding: "yes",
    assistance_count: 2,
    undo_count: 1,
  },
  { ...base, build: "fixture-B", cohort: "existing", elapsed_ms: 500 },
]);
assert.equal(result.groups.length, 2);
const g = result.groups[0];
assert.equal(g.completion_rate, 0.5);
assert.equal(g.median_all_elapsed_ms, 2000);
assert.equal(g.median_completed_elapsed_ms, 1000);
assert.equal(g.sharing_assessed, 1);
assert.equal(g.sharing_misunderstanding_rate, 1);
assert.equal(g.assistance_total, 2);
assert.equal(g.undo_total, 1);
for (const record of [
  { ...base, measurement_type: "human_observation" },
  { ...base, consent: true },
  { ...base, observer_confirmed: true },
  { ...base, document_id: "not-allowed" },
  { ...base, participant_id: "name@example.com" },
  { ...base, elapsed_ms: Infinity },
  { ...base, elapsed_ms: -1 },
  { ...base, undo_count: 1.5 },
  { ...base, sharing_misunderstanding: "unknown" },
  { ...base, observed_at: "not-a-time" },
])
  assert.throws(() => validateRecord(record));
assert.throws(() => summarize([base, base]), /중복/);
// Synthetic objects exercise human validation only; never saved or reported as observation evidence.
const human = {
  ...base,
  measurement_type: "human_observation",
  consent: true,
  observer_confirmed: true,
};
const separated = summarize([base, human]);
assert.equal(separated.groups.length, 2);
assert.equal(separated.human_observation_records, 1);
assert.equal(separated.automated_probe_records, 1);
console.log(
  "PASS synthetic validator fixtures only: human consent, automated separation, build/cohort/task segmentation, sample and abandoned denominators, unknown-not-zero, duration medians, duplicate/PII/invalid rejection; actual human observations0",
);
