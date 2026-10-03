package channels
import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "net/url"
 "strings"
 "testing"
 "time"
)
type reviewerReadErrorBody struct { b []byte; off int }
func (r *reviewerReadErrorBody) Read(dst []byte) (int,error) { if r.off>=len(r.b) { return 0,io.EOF }; n:=copy(dst,r.b[r.off:]);r.off+=n; if r.off==len(r.b){return n,io.ErrUnexpectedEOF}; return n,nil }
func (r *reviewerReadErrorBody) Close() error { return nil }
type reviewerBodyErrorTransport struct { inner http.RoundTripper }
func (r reviewerBodyErrorTransport) RoundTrip(q *http.Request)(*http.Response,error) { s,e:=r.inner.RoundTrip(q); if e!=nil{return s,e}; b,e:=io.ReadAll(s.Body); s.Body.Close(); if e!=nil{return nil,e}; s.Body=&reviewerReadErrorBody{b:b}; return s,nil }
func TestReviewerReadErrorCannotPass(t *testing.T) {
 tp,inv:=pfFixture(); opts:=pfOpts(reviewerBodyErrorTransport{tp})
 r,e:=RunPancakeProof(context.Background(),opts,inv)
 if e!=nil {t.Fatal(e)}
 if r.Disposition==ProofPass {t.Fatalf("READ_ERROR_FALSE_PASS: overall=%s attempts=%d; every body returned io.ErrUnexpectedEOF",r.Disposition,r.Budget.AttemptsUsed)}
}
func TestReviewerDecodedSensitiveValuesRejected(t *testing.T) {
 for _,canary:=range []string{"CANARY-<RAW>&VALUE", "CANARY-QUOTE-\"-VALUE", "CANARY-SLASH-\\-VALUE", "CANARY-LINE-\n-VALUE"} {
  t.Run(fmt.Sprintf("encoding-%d",len(canary)),func(t *testing.T){
   tp,inv:=pfFixture(); r:=pfRun(t,tp,inv,nil); r.Reasons=append(r.Reasons,canary)
   b,e:=MarshalProofReceipt(r,[]string{canary})
   if errors.Is(e,ErrPancakeProofUnsafe) {return}; if e!=nil{t.Fatal(e)}
   var decoded ProofReceipt; if e=json.Unmarshal(b,&decoded);e!=nil{t.Fatal(e)}
   for _,s:=range decoded.Reasons{if s==canary{t.Fatal("SANITATION_BYPASS: decoded receipt retains the supplied sensitive value after JSON escaping")}}
  })
 }
}
type reviewerCountingTransport struct { n int }
func (r *reviewerCountingTransport) RoundTrip(q *http.Request)(*http.Response,error) {r.n++; return &http.Response{StatusCode:200,Status:"200",Header:http.Header{},Body:io.NopCloser(strings.NewReader(`{"conversations":[],"messages":[]}`)),Request:q},nil}
func TestReviewerApprovedTraversalIDsDenied(t *testing.T) {
 for _,id:=range []string{"c-aaa1/../x","..","a\\..\\x"} {
  t.Run(id,func(t *testing.T){
   rt:=&reviewerCountingTransport{};opts:=pfOpts(rt)
   tr:=&proofTransport{inner:rt,opts:&opts,approved:map[string]bool{id:true},now:opts.Now,start:opts.Now()};tr.beginRun(1);tr.setConv(id)
   raw:="https://pages.fm/api/public_api/v1/pages/"+url.PathEscape(opts.PageID)+"/conversations/"+url.PathEscape(id)+"/messages?page_access_token="+url.QueryEscape(opts.Token)
   _,e:=tr.RoundTrip(pfReq(t,http.MethodGet,raw))
   if e==nil || rt.n!=0{t.Fatalf("APPROVED_TRAVERSAL_REACHED_TRANSPORT: err=%v calls=%d",e,rt.n)}
  })
 }
 t.Run("page-dot-segment",func(t *testing.T){
  rt:=&reviewerCountingTransport{};opts:=pfOpts(rt);opts.PageID=".."
  tr:=&proofTransport{inner:rt,opts:&opts,approved:map[string]bool{},now:opts.Now,start:opts.Now()};tr.beginRun(1)
  raw:="https://pages.fm/api/public_api/v2/pages/../conversations?type=INBOX&order_by=updated_at&since="+fmt.Sprint(opts.Since.Unix())+"&until=1790000000&page_access_token="+url.QueryEscape(opts.Token)
  _,e:=tr.RoundTrip(pfReq(t,http.MethodGet,raw));if e==nil||rt.n!=0{t.Fatalf("PAGE_TRAVERSAL_REACHED_TRANSPORT: err=%v calls=%d",e,rt.n)}
 })
}
func TestReviewerConversationRawEligibilityMatchesAdapter(t *testing.T) {
 tp,inv:=pfFixture()
 tp.Routes["conv:"]=[]ProofResponse{pfOK(pfList("conversations",pfConvRow("c-aaa1","INBOX","2026-09-24T10:00:00.000000"),pfConvRow("c-bbb2","INBOX","2026-09-01T00:00:00.000000"),pfConvRow("c-old3","INBOX","2026-09-01T00:00:00.000000")))}
 tp.Routes["conv:c-old3"]=[]ProofResponse{pfOK(pfList("conversations",pfConvRow("c-aaa1","INBOX","2026-09-24T10:00:00.000000"),pfConvRow("c-cmt4","COMMENT","2026-09-24T10:00:00.000000"),pfConvRow("c-bbb2","INBOX","2026-09-20T00:00:00.000000"),pfConvRow("c-ddd5","INBOX","2026-09-25T01:00:00+07:00")))}
 r:=pfRun(t,tp,inv,nil);if r.Disposition!=ProofPass{t.Fatalf("positive adapter/inventory control failed: %s",r.Disposition)}
 for _,run:=range r.Runs{if run.ConversationRows.RawEligible!=run.ConversationRows.Mapped{t.Fatalf("RAW_ELIGIBILITY_INCORRECT: PASS receipt eligible=%d mapped=%d; old occurrence incorrectly suppresses later eligible conversation",run.ConversationRows.RawEligible,run.ConversationRows.Mapped)}}
}
func TestReviewerUntilValidationRunsAfterObservation(t *testing.T) {
 tp,inv:=pfFixture();opts:=pfOpts(tp);tr:=&proofTransport{inner:tp,opts:&opts,approved:map[string]bool{},now:opts.Now,start:opts.Now()}
 for _,c:=range inv.Conversations{tr.approved[c.ID]=true}
 first:=true
 opts.Now=func() time.Time {if first {first=false; tr.untils=append(tr.untils,"1")};return time.Now()}
 tr.now=time.Now;tr.start=time.Now()
 r:=runProofOnce(context.Background(),1,newProofAdapter(&opts,tr),tr,&opts,inv,5)
 if r.UntilStable{t.Fatal("probe setup failed: observation must be unstable")}
 if r.Disposition==ProofPass{t.Fatal("UNTIL_UNSTABLE_FALSE_PASS: UntilStable=false was populated in defer after the disposition check")}
}

func TestReviewerUnsupportedReceiptValuesRejected(t *testing.T) {
 for _,tc:=range []struct{name string;edit func(*ProofReceipt)}{
  {"unknown-schema",func(r *ProofReceipt){r.SchemaVersion="unknown-schema"}},
  {"unknown-disposition",func(r *ProofReceipt){r.Disposition="UNKNOWN"}},
  {"invalid-source-sha",func(r *ProofReceipt){r.SourceSHA="not-a-source-sha"}},
 }{
  t.Run(tc.name,func(t *testing.T){tp,inv:=pfFixture();r:=pfRun(t,tp,inv,nil);tc.edit(r)
   if _,e:=MarshalProofReceipt(r,nil);!errors.Is(e,ErrPancakeProofUnsafe){t.Fatalf("UNSUPPORTED_RECEIPT_ACCEPTED: %s error=%v",tc.name,e)}
  })
 }
}
