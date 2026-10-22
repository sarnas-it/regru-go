package regru

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// newTestClient поднимает сервер, который отдаёт body и передаёт разобранную форму в check.
func newTestClient(t *testing.T, body string, check func(path string, form url.Values)) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.RawQuery != "" {
			t.Errorf("query string must be empty, got %q", r.URL.RawQuery)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if check != nil {
			check(r.URL.Path, r.PostForm)
		}
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return New("user", "secret", WithBaseURL(srv.URL))
}

func TestListDomains(t *testing.T) {
	c := newTestClient(t, `{"result":"success","answer":{"services":[
		{"creation_date":"2009-04-18","dname":"test.ru","expiration_date":"2011-04-18","service_id":"111","servtype":"domain","state":"A","subtype":"test","uplink_service_id":"0"}]}}`,
		func(path string, f url.Values) {
			if path != "/service/get_list" {
				t.Errorf("path = %s", path)
			}
			if f.Get("username") != "user" || f.Get("password") != "secret" || f.Get("servtype") != "domain" {
				t.Errorf("form = %v", f)
			}
		})
	got, err := c.ListDomains(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Domain != "test.ru" || got[0].ID != "111" || got[0].State != StateActive {
		t.Fatalf("got %+v", got)
	}
}

func TestServiceInfoItemErrorAndChunking(t *testing.T) {
	var calls, firstSize int
	c := newTestClient(t, `{"result":"success","answer":{"services":[
		{"service_id":12345,"servtype":"domain","state":"A","result":"success"},
		{"error_code":"DOMAIN_NOT_FOUND","error_text":"not found","result":"error"}]}}`,
		func(_ string, f url.Values) {
			calls++
			var in struct {
				Services []map[string]string `json:"services"`
			}
			if f.Get("input_format") != "json" {
				t.Errorf("input_format = %q", f.Get("input_format"))
			}
			if err := json.Unmarshal([]byte(f.Get("input_data")), &in); err != nil {
				t.Fatal(err)
			}
			if calls == 1 {
				firstSize = len(in.Services)
			}
			var m map[string]any
			json.Unmarshal([]byte(f.Get("input_data")), &m)
			if _, ok := m["password"]; ok {
				t.Error("credentials must not be inside input_data")
			}
		})

	got, err := c.ServiceInfo(context.Background(), "a.ru", "b.ru")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "12345" || got[0].Err != nil {
		t.Fatalf("got %+v", got)
	}
	if !errors.Is(got[1].Err, ErrDomainNotFound) {
		t.Fatalf("item err = %v", got[1].Err)
	}

	calls = 0
	many := make([]string, 1500)
	for i := range many {
		many[i] = "d.ru"
	}
	if _, err := c.ServiceInfo(context.Background(), many...); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || firstSize != 1000 {
		t.Fatalf("calls = %d, first chunk = %d, want 2 and 1000", calls, firstSize)
	}
}

func TestAPIError(t *testing.T) {
	c := newTestClient(t, `{"result":"error","error_code":"ACCESS_DENIED_FROM_IP","error_text":"Access to API from this IP denied"}`, nil)
	_, err := c.Balance(context.Background(), "")
	var ae *APIError
	if !errors.As(err, &ae) || ae.Code != "ACCESS_DENIED_FROM_IP" {
		t.Fatalf("err = %v", err)
	}
	if !errors.Is(err, ErrAccessDeniedFromIP) || errors.Is(err, ErrDomainNotFound) {
		t.Fatalf("errors.Is mismatch for %v", err)
	}
}

func TestDomainPrices(t *testing.T) {
	c := newTestClient(t, `{"result":"success","answer":{"currency":"RUR","price_group":"Retail","prices":{
		"__idn.com":{"reg_max_period":"10","reg_min_period":"1","reg_price":"960"},
		"ru":{"reg_max_period":"1","reg_min_period":"1","reg_price":"590"}}}}`,
		func(_ string, f url.Values) {
			if f.Get("show_renew_data") != "1" || f.Get("currency") != "USD" {
				t.Errorf("form = %v", f)
			}
		})
	p, err := c.DomainPrices(context.Background(), PricesOptions{Currency: "USD", WithRenew: true})
	if err != nil {
		t.Fatal(err)
	}
	if f, _ := p.Zones["ru"].RegPrice.Float64(); f != 590 {
		t.Fatalf("ru price = %v", p.Zones["ru"].RegPrice)
	}
	if _, ok := p.Zones[IDNPrefix+"com"]; !ok {
		t.Fatal("idn zone missing")
	}
}

func TestLotListDecodesMixedTypes(t *testing.T) {
	c := newTestClient(t, `{"result":"success","answer":{"lots_cnt":"1","lots":[
		{"bids_cnt":"0","category_ids":"[ 1, 2, 3 ]","dname":"test-shop-api-1.ru","is_online":"1","keywords":"test, shop, api",
		 "lot_hits":"null","price_type":"fixed","site":"0","start_price":"3000.00","tm":1}]}}`,
		func(_ string, f url.Values) {
			if f.Get("show_my_lots") != "1" || f.Get("itemsonpage") != "100" || f.Get("pg") != "0" {
				t.Errorf("form = %v", f)
			}
		})
	lots, err := c.LotList(context.Background(), LotListParams{MyLots: true, PerPage: 100})
	if err != nil {
		t.Fatal(err)
	}
	l := lots[0]
	if l.StartPrice != "3000.00" || !l.Online || l.WithSite || !l.WithTrademark || l.PriceType != PriceFixed {
		t.Fatalf("lot = %+v", l)
	}
	if len(l.CategoryIDs) != 3 || len(l.Keywords) != 3 || l.Keywords[2] != "api" {
		t.Fatalf("lists: %v %v", l.CategoryIDs, l.Keywords)
	}
}

func TestLotListRejectsBadPageSize(t *testing.T) {
	c := New("u", "p")
	if _, err := c.LotList(context.Background(), LotListParams{PerPage: 30}); err == nil {
		t.Fatal("want error for PerPage=30")
	}
}

func TestEachLotPaginates(t *testing.T) {
	pages := []string{
		`{"result":"success","answer":{"lots":[` + lotN(25) + `]}}`,
		`{"result":"success","answer":{"lots":[{"dname":"last.ru"}]}}`,
	}
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		seen = append(seen, r.PostForm.Get("pg"))
		idx := len(seen) - 1
		w.Write([]byte(pages[idx]))
	}))
	defer srv.Close()
	c := New("u", "p", WithBaseURL(srv.URL))

	n := 0
	if err := c.EachLot(context.Background(), LotListParams{}, func(Lot) error { n++; return nil }); err != nil {
		t.Fatal(err)
	}
	if n != 26 || len(seen) != 2 || seen[0] != "0" || seen[1] != "1" {
		t.Fatalf("n=%d pages=%v", n, seen)
	}
}

func lotN(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			s += ","
		}
		s += `{"dname":"x.ru"}`
	}
	return s
}

func TestUnexpectedResponse(t *testing.T) {
	c := newTestClient(t, `<html>502</html>`, nil)
	if _, err := c.Nop(context.Background()); err == nil {
		t.Fatal("want error")
	}
}

func TestZoneRecords(t *testing.T) {
	c := newTestClient(t, `{"result":"success","answer":{"domains":[
		{"dname":"a.ru","result":"success","service_id":12345,"rrs":[
			{"content":"1.2.3.4","prio":"0","rectype":"A","state":"A","subname":"www"}]},
		{"dname":"b.ru","result":"DOMAIN_IS_NOT_USE_REGRU_NSS"},
		{"dname":"c.ru","result":"error","error_code":"DOMAIN_NOT_FOUND","error_text":"x"}]}}`,
		func(path string, f url.Values) {
			if path != "/zone/get_resource_records" {
				t.Errorf("path = %s", path)
			}
		})
	zs, err := c.ZoneRecords(context.Background(), "a.ru", "b.ru", "c.ru")
	if err != nil {
		t.Fatal(err)
	}
	if zs[0].Err != nil || len(zs[0].Records) != 1 || zs[0].Records[0].Sub != "www" || zs[0].Records[0].Content != "1.2.3.4" {
		t.Fatalf("zone a: %+v", zs[0])
	}
	if !errors.Is(zs[1].Err, ErrNotRegruNS) {
		t.Fatalf("zone b err = %v, want ErrNotRegruNS", zs[1].Err)
	}
	if !errors.Is(zs[2].Err, ErrDomainNotFound) {
		t.Fatalf("zone c err = %v", zs[2].Err)
	}
}

func TestAddARecord(t *testing.T) {
	c := newTestClient(t, `{"result":"success","answer":{"domains":[{"dname":"a.ru","result":"success","service_id":"1"}]}}`,
		func(path string, f url.Values) {
			if path != "/zone/add_alias" || f.Get("input_format") != "json" {
				t.Errorf("path %s form %v", path, f)
			}
			var in struct {
				Domains   []map[string]string `json:"domains"`
				Subdomain string              `json:"subdomain"`
				IP        string              `json:"ipaddr"`
			}
			json.Unmarshal([]byte(f.Get("input_data")), &in)
			if in.Domains[0]["dname"] != "a.ru" || in.Subdomain != "@" || in.IP != "9.9.9.9" {
				t.Errorf("input = %+v", in)
			}
		})
	if err := c.AddARecord(context.Background(), "a.ru", "@", "9.9.9.9"); err != nil {
		t.Fatal(err)
	}

	c = newTestClient(t, `{"result":"success","answer":{"domains":[{"dname":"a.ru","result":"DOMAIN_IS_NOT_USE_REGRU_NSS"}]}}`, nil)
	if err := c.AddARecord(context.Background(), "a.ru", "@", "9.9.9.9"); !errors.Is(err, ErrNotRegruNS) {
		t.Fatalf("err = %v", err)
	}
}

func TestRemoveARecord(t *testing.T) {
	c := newTestClient(t, `{"result":"success","answer":{"domains":[{"dname":"a.ru","result":"success","service_id":"1"}]}}`,
		func(path string, f url.Values) {
			if path != "/zone/remove_record" {
				t.Errorf("path = %s", path)
			}
			var in struct {
				Domains []map[string]string `json:"domains"`
				Sub     string              `json:"subdomain"`
				Type    string              `json:"record_type"`
				Content string              `json:"content"`
			}
			json.Unmarshal([]byte(f.Get("input_data")), &in)
			if in.Domains[0]["dname"] != "a.ru" || in.Sub != "www" || in.Type != "A" || in.Content != "1.2.3.4" {
				t.Errorf("input = %+v", in)
			}
		})
	if err := c.RemoveARecord(context.Background(), "a.ru", "www", "1.2.3.4"); err != nil {
		t.Fatal(err)
	}
	// без содержимого запрос вообще не уходит: иначе API удалил бы все A-записи имени
	if err := c.RemoveARecord(context.Background(), "a.ru", "www", ""); err == nil {
		t.Fatal("empty content must be rejected")
	}
}
