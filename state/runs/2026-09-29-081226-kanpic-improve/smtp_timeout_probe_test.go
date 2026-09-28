package mail
import("context";"net";"testing";"time";"bufio";"fmt")
func TestScoutSMTPTimeout(t *testing.T) {
for _, phase := range []string{"greeting", "ehlo"} { for _, action := range []string{"Deliver", "Verify"} { t.Run(phase+"/"+action,func(t *testing.T){
l,err:=net.Listen("tcp","127.0.0.1:0");if err!=nil{t.Fatal(err)};defer l.Close()
accepted:=make(chan net.Conn,1)
go func(){c,e:=l.Accept();if e!=nil{return};accepted<-c;if phase=="ehlo"{fmt.Fprint(c,"220 local ESMTP\r\n");bufio.NewReader(c).ReadString('\n')}}()
cfg:=Config{Host:"127.0.0.1",Port:l.Addr().(*net.TCPAddr).Port,FromAddress:"a@example.test",Security:"none",Timeout:100*time.Millisecond}
ctx,cancel:=context.WithTimeout(context.Background(),150*time.Millisecond);defer cancel()
done:=make(chan error,1);go func(){if action=="Verify"{done<-Verify(ctx,cfg)}else{done<-Deliver(ctx,cfg,Message{To:"b@example.test"})}}()
var c net.Conn;select{case c=<-accepted:case <-time.After(time.Second):t.Fatal("accept timeout")};defer c.Close()
select {case err:=<-done:t.Logf("returned: %v",err);case <-time.After(500*time.Millisecond):t.Errorf("still blocked after 500ms: Config.Timeout=100ms context=150ms");c.Close();select{case <-done:case <-time.After(time.Second):t.Error("cleanup timeout")}}
})}}
}
