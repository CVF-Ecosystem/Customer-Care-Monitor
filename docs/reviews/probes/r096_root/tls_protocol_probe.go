package main
import("crypto/tls";"crypto/x509";"encoding/json";"fmt";"io";"net/http";"net/http/httptest";"os";"time")
func probe(fix bool,url string,cert *x509.Certificate) map[string]any {
 t:=http.DefaultTransport.(*http.Transport).Clone();t.DisableKeepAlives=true;t.ForceAttemptHTTP2=false;t.TLSNextProto=map[string]func(string,*tls.Conn)http.RoundTripper{}
 advertised:=append([]string(nil),t.TLSClientConfig.NextProtos...)
 pool:=x509.NewCertPool();pool.AddCert(cert);t.TLSClientConfig.RootCAs=pool
 if fix {t.TLSClientConfig.NextProtos=[]string{"http/1.1"}}
 c:=&http.Client{Transport:t,Timeout:3*time.Second};defer t.CloseIdleConnections()
 resp,err:=c.Get(url);r:=map[string]any{"fix":fix,"clonedAdvertisedProtocols":advertised,"effectiveAdvertisedProtocols":t.TLSClientConfig.NextProtos,"status":0}
 if err!=nil {r["error"]=err.Error();return r};defer resp.Body.Close();body,_:=io.ReadAll(resp.Body);r["status"]=resp.StatusCode;r["responseProtocol"]=resp.Proto;r["body"]=string(body);return r
}
func main(){s:=httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){fmt.Fprint(w,"loopback transport fixture") }));s.EnableHTTP2=true;s.StartTLS();defer s.Close();a:=probe(false,s.URL,s.Certificate());b:=probe(true,s.URL,s.Certificate());json.NewEncoder(os.Stdout).Encode(map[string]any{"scope":"local TLS configuration only; no provider/DB/governance proof","before":a,"after":b});if a["status"]!=0 || b["status"]!=200 {os.Exit(2)}}
