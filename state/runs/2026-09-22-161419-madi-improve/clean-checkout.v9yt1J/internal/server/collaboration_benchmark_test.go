package server

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/reearth/ygo/crdt"
)

// Opt-in native-stage measurement, not an end-to-end latency/load SLA. Each
// variant processes the same updates for eight 600-block documents and proves
// the final content and clocks match. Wire encodings may have different Item
// segmentation after replay. PostgreSQL, HTTP and browser time are excluded.
func TestCollaborationNativeCacheMeasurement(t *testing.T) {
	if os.Getenv("MADI_COLLABORATION_BENCHMARK") != "1" {
		t.Skip("opt-in representative native CRDT cache measurement")
	}
	const rooms, blocks, edits = 8, 600, 10
	states := make([][]byte, rooms)
	updates := make([][][]byte, rooms)
	var encodedBytes int
	for i := 0; i < rooms; i++ {
		d := crdt.New()
		root := d.GetXmlFragment("content")
		d.Transact(func(tx *crdt.Transaction) {
			for j := 0; j < blocks; j++ {
				p := crdt.NewYXmlElement("paragraph")
				p.SetAttribute(tx, "id", fmt.Sprintf("room-%d-block-%d", i, j))
				text := crdt.NewYXmlText()
				text.Insert(tx, 0, fmt.Sprintf("문단 %d: 공동 편집 운영 문서의 저장·복원·권한을 확인합니다. %s", j, strings.Repeat("내용 ", 24)), nil)
				p.InsertText(tx, 0, text)
				root.InsertElement(tx, j, p)
			}
		})
		states[i] = d.EncodeStateAsUpdate()
		encodedBytes += len(states[i])
		for j := 0; j < edits; j++ {
			vector := d.StateVector()
			collaborationTestInsert(d, " 변경")
			updates[i] = append(updates[i], crdt.EncodeStateAsUpdateV1(d, vector))
		}
		d.Destroy()
	}
	debug.FreeOSMemory()
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)
	beforeRSS := collaborationRSS()
	run := func(cached bool) (time.Duration, int64, uint64, [][]byte) {
		var initial runtime.MemStats
		runtime.ReadMemStats(&initial)
		docs := make([]*crdt.Doc, rooms)
		result := make([][]byte, rooms)
		copy(result, states)
		start := time.Now()
		for n := 0; n < edits; n++ {
			for i := 0; i < rooms; i++ {
				d := docs[i]
				if d == nil {
					d = crdt.New()
					d.GetXmlFragment("content")
					if err := crdt.ApplyUpdateV1(d, result[i], nil); err != nil {
						t.Fatal(err)
					}
					if _, _, err := collaborationMarkdown(d.GetXmlFragment("content")); err != nil {
						t.Fatal(err)
					}
				}
				if err := crdt.ApplyUpdateV1(d, updates[i][n], nil); err != nil {
					t.Fatal(err)
				}
				result[i] = d.EncodeStateAsUpdate()
				if _, _, err := collaborationMarkdown(d.GetXmlFragment("content")); err != nil {
					t.Fatal(err)
				}
				if cached {
					docs[i] = d
				} else {
					d.Destroy()
				}
			}
		}
		elapsed := time.Since(start)
		var after runtime.MemStats
		runtime.ReadMemStats(&after)
		rss := collaborationRSS()
		for _, d := range docs {
			if d != nil {
				d.Destroy()
			}
		}
		return elapsed, rss, after.TotalAlloc - initial.TotalAlloc, result
	}
	cold, coldRSS, coldAlloc, coldResult := run(false)
	debug.FreeOSMemory()
	warm, warmRSS, warmAlloc, warmResult := run(true)
	for i := range coldResult {
		a, b := collaborationTestClone(t, coldResult[i]), collaborationTestClone(t, warmResult[i])
		am, ax, ae := collaborationMarkdown(a.GetXmlFragment("content"))
		bm, bx, be := collaborationMarkdown(b.GetXmlFragment("content"))
		if ae != nil || be != nil || am != bm || string(jsonValue(ax)) != string(jsonValue(bx)) || !reflect.DeepEqual(a.StateVector(), b.StateVector()) {
			t.Fatal("native cache changed canonical content/metadata/clocks", i, ae, be)
		}
	}
	t.Logf(`COLLABORATION_NATIVE_MEASUREMENT {"documents":%d,"blocks_per_document":%d,"updates":%d,"initial_encoded_bytes":%d,"baseline_heap_bytes":%d,"baseline_rss_bytes":%d,"rebuild_ms":%.3f,"cached_ms":%.3f,"rebuild_total_alloc_bytes":%d,"cached_total_alloc_bytes":%d,"rebuild_observed_rss_bytes":%d,"cached_observed_rss_bytes":%d,"includes_database":false,"rss_is_upper_bound":false}`, rooms, blocks, rooms*edits, encodedBytes, baseline.HeapAlloc, beforeRSS, float64(cold.Microseconds())/1000, float64(warm.Microseconds())/1000, coldAlloc, warmAlloc, coldRSS, warmRSS)
}

func collaborationRSS() int64 {
	data, e := os.ReadFile("/proc/self/status")
	if e != nil {
		return -1
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "VmRSS:" {
			n, _ := strconv.ParseInt(fields[1], 10, 64)
			return n * 1024
		}
	}
	return -1
}
