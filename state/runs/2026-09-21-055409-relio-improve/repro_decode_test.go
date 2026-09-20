package server
import (
 "net/http/httptest"
 "strings"
 "testing"
)
func TestScoutDecodeSize(t *testing.T) {
 for _, tc := range []struct{name, body string; want int}{
  {"oversized first value", `{"name":"`+strings.Repeat("x",2<<20)+`"}`,413},
  {"oversized trailing whitespace", `{}`+strings.Repeat(" ",2<<20),400},
 } {
  t.Run(tc.name,func(t *testing.T){
   r:=httptest.NewRequest("POST","/api/v1/customers",strings.NewReader(tc.body))
   w:=httptest.NewRecorder()
   (&Server{}).createCustomer(w,r)
   t.Logf("status=%d response=%s",w.Code,w.Body.String())
   if w.Code!=tc.want {t.Fatalf("want baseline %d",tc.want)}
  })
 }
}
