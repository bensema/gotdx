package gotdx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestICFQSPostTQLParsesWrappedResponse(t *testing.T) {
	var gotEntry string
	var gotBody map[string]any
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotEntry = r.URL.Query().Get("Entry")
		if r.Header.Get("Content-Type") != "text/plain;charset=UTF-8" {
			t.Fatalf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		// 响应正文前后夹带无关文本，解析器仍应提取出中间的 JSON 结果集。
		_, _ = w.Write([]byte(`noise {"ResultSets":[{"ColName":["code","name"],"Content":[["000001","PINGAN"]]}]} tail`))
	}))

	client := NewICFQS(WithICFQSAddress(server.URL), WithICFQSHTTPClient(server.Client()))
	raw, err := client.ICFQSTopicListRaw(context.Background(), "X11", "4", 2)
	if err != nil {
		t.Fatalf("ICFQSTopicListRaw failed: %v", err)
	}

	if gotEntry != "CWServ.ph_tdxdatacenter_zttz_zy" {
		t.Fatalf("unexpected entry: %s", gotEntry)
	}
	params, ok := gotBody["Params"].([]any)
	if !ok || len(params) != 3 || params[0] != "00601" || params[1] != "X11|4" || params[2].(float64) != 2 {
		t.Fatalf("unexpected params: %#v", gotBody["Params"])
	}
	if gotBody["oauth_zzfw"] != "1" {
		t.Fatalf("missing oauth flag: %#v", gotBody)
	}

	tables := ICFQSFormatTables(raw)
	if len(tables) != 1 || len(tables[0].Rows) != 1 {
		t.Fatalf("unexpected tables: %#v", tables)
	}
	if tables[0].Rows[0]["code"] != "000001" || tables[0].Rows[0]["name"] != "PINGAN" {
		t.Fatalf("unexpected row: %#v", tables[0].Rows[0])
	}
}

func TestICFQSQuotesBatchRawPostsJSONBody(t *testing.T) {
	var gotEntry string
	var gotBody map[string]any
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotEntry = r.URL.Query().Get("Entry")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"ListHead":{"ItemHead":["Code","NOW"]},"ListItem":[{"Item":["000001","12.34"]}]}`))
	}))

	client := NewICFQS(WithICFQSAddress(server.URL), WithICFQSHTTPClient(server.Client()))
	// 校验批量行情请求按 JSON 格式提交市场、代码与请求字段。
	raw, err := client.ICFQSQuotesBatchRaw(
		context.Background(),
		[]ICFQSCode{{Setcode: "0", Code: "000001"}},
		[]string{"NOW"},
	)
	if err != nil {
		t.Fatalf("ICFQSQuotesBatchRaw failed: %v", err)
	}

	if gotEntry != "HQServ.PBCombHQ" {
		t.Fatalf("unexpected entry: %s", gotEntry)
	}
	codes := gotBody["Code"].([]any)
	setcodes := gotBody["Setcode"].([]any)
	wantCols := gotBody["WantCol"].([]any)
	if codes[0] != "000001" || setcodes[0] != "0" || wantCols[0] != "NOW" {
		t.Fatalf("unexpected request body: %#v", gotBody)
	}
	if raw["ListHead"] == nil || raw["ListItem"] == nil {
		t.Fatalf("unexpected raw response: %#v", raw)
	}
}

func TestICFQSFormatTablesUsesColDes(t *testing.T) {
	raw := map[string]any{
		"ResultSets": []any{
			map[string]any{
				"ColDes": []any{
					map[string]any{"Name": "rq"},
					map[string]any{"Name": "code"},
				},
				"Content": []any{
					[]any{"2026-07-04", "000001"},
				},
			},
		},
	}

	// 缺少 `ColName` 时回退到 `ColDes` 元数据中的 `Name`，再按 `Content` 行数据组装表格。
	tables := ICFQSFormatTables(raw)
	if len(tables) != 1 || tables[0].Rows[0]["rq"] != "2026-07-04" || tables[0].Rows[0]["code"] != "000001" {
		t.Fatalf("unexpected tables: %#v", tables)
	}
}
