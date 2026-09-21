package client

import("context";"net/http";"net/http/httptest";"testing")
func TestNetworkInfo(t *testing.T){srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){if r.URL.Path!="/network/info"{t.Fatalf("path=%s",r.URL.Path)};_,_=w.Write([]byte(`{"network":"testnet","network_id":"ind-testnet-1","chain_id":777101}`))}));defer srv.Close();info,err:=New(srv.URL).NetworkInfo(context.Background());if err!=nil{t.Fatal(err)};if info["network_id"]!="ind-testnet-1"{t.Fatalf("%v",info)}}
