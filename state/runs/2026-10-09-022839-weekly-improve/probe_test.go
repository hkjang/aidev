package app
import("fmt"; "net/http"; "testing")
func TestScoutUploadOrderQueryOverflow(t *testing.T) {
 s := newTestServer(t)
 author := s.createUser("scout_overflow", "USER", nil)
 id, _ := s.draft(author, "2026-08-24", "순서 조회 오류")
 path := fmt.Sprintf("/api/v1/reports/%d/attachments",id)
 ids := s.uploadCaptures(t,path,author,"a.png","b.png")
 s.patchCapture(t,path,ids["a.png"],map[string]any{"sortOrder":0},author)
 s.patchCapture(t,path,ids["b.png"],map[string]any{"sortOrder":2147483647},author)
 response := s.uploadMany(path,"files",map[string][]byte{"c.png":samplePNG(t,13,13)},author)
 t.Logf("upload response: %d %s",response.Code,response.Body.String())
 items := s.listCaptures(t,path,author)
 t.Logf("listed order: %v",orderByFilename(items,placementAfter))
 if response.Code != http.StatusInternalServerError { t.Errorf("failed next-order query must refuse upload; got HTTP %d",response.Code) }
 if len(items)!=2 { t.Errorf("failed query stored a new attachment: got %d rows, want 2",len(items)) }
}
