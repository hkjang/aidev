package app
import("context";"fmt";"net/http/httptest";"testing";"time")
func TestScoutAttachmentCleanupDuringFirstUpload(t *testing.T) {
 s:=newTestServer(t)
 author:=s.createUser("scout_attachment", "USER", nil)
 id,_:=s.draft(author,"2026-08-24","cleanup probe")
 ctx,cancel:=context.WithTimeout(context.Background(),15*time.Second); defer cancel()
 tx,err:=s.app.db.Begin(ctx); if err!=nil {t.Fatal(err)}; defer tx.Rollback(context.Background())
 if _,err=tx.Exec(ctx,"LOCK TABLE report_attachments IN SHARE MODE");err!=nil {t.Fatal(err)}
 path:=fmt.Sprintf("/api/v1/reports/%d/attachments",id)
 payload:=samplePNG(t,4,4)
 done:=make(chan *httptest.ResponseRecorder,1)
 go func(){done<-s.uploadMany(path,"files",map[string][]byte{"probe.png":payload},author)}()
 for {
  var waiting bool
  err=s.app.db.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE 'INSERT INTO report_attachments%')`).Scan(&waiting)
  if err!=nil {t.Fatal(err)}
  if waiting {break}; select{case <-ctx.Done():t.Fatal(ctx.Err());case <-time.After(10*time.Millisecond):}
 }
 t.Logf("before cleanup: rows=%d files=%v",s.attachmentCount(id),s.attachmentFiles(t,id))
 s.app.cleanupAttachmentFiles(ctx)
 if err=tx.Commit(ctx);err!=nil {t.Fatal(err)}
 var response *httptest.ResponseRecorder
 select{case response=<-done:case <-ctx.Done():t.Fatal(ctx.Err())}
 t.Logf("upload status=%d body=%s",response.Code,response.Body.String())
 if response.Code!=201 {t.Fatalf("upload=%d",response.Code)}
 var aid int64
 if err=s.app.db.QueryRow(ctx,"SELECT id FROM report_attachments WHERE report_id=$1",id).Scan(&aid);err!=nil {t.Fatal(err)}
 got:=s.request("GET",fmt.Sprintf("%s/%d",path,aid),nil,author)
 if got.Code!=200 {t.Fatalf("successful upload then image GET=%d body=%s",got.Code,got.Body.String())}
}
