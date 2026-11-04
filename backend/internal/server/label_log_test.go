package server_test

import (
	"fmt"
	"testing"

	"github.com/belimked/gz_mes/backend/internal/api"
)

func TestLabelLogs(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	_, _, c := e.batchContract(admin, "log")
	var b api.Batch
	if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: c.Id, LotNo: "LOG-001"}, &b); code != 201 {
		t.Fatal(code)
	}
	path := fmt.Sprintf("/batches/%d", b.Id)
	if code := e.do("PUT", path+"/items", admin, []api.BatchItemInput{{PieceNo: "P01", Qty: 2, PieceArea: floatPtr(1.25)}}, nil); code != 200 {
		t.Fatal(code)
	}
	var gen api.LabelGenerateResult
	e.do("POST", path+"/labels/generate", admin, nil, &gen)
	if gen.Created != 2 {
		t.Fatal(gen)
	}
	e.do("POST", path+"/labels/generate", admin, nil, &gen)
	if gen.Created != 0 {
		t.Fatal(gen)
	}
	var page api.LabelLogPage
	e.do("GET", "/label-logs?opType=generate&keyword=LOG-001", admin, nil, &page)
	if page.Total != 2 || page.Totals.GenerateQty != 2 || page.Totals.Area != 2.5 {
		t.Fatal(page)
	}
	bar := page.Items[0].BarCode
	var er api.Error
	if code := e.do("POST", "/labels/"+bar+"/reprint", admin, map[string]any{"reason": "  "}, &er); code != 400 {
		t.Fatal(code)
	}
	if code := e.do("POST", "/labels/"+bar+"/reprint", admin, map[string]any{"reason": "重印"}, nil); code != 200 {
		t.Fatal(code)
	}
	var audit api.AuditLogPage
	e.do("GET", "/audit-logs?keyword="+bar, admin, nil, &audit)
	if audit.Total == 0 || audit.Items[0].Route != "/labels/{barCode}/reprint" {
		t.Fatalf("route pattern: %+v", audit.Items)
	}
	if code := e.do("POST", "/labels/"+bar+"/cancel", admin, map[string]any{"reason": "损坏"}, nil); code != 200 {
		t.Fatal(code)
	}
	if code := e.do("POST", "/labels/"+bar+"/reprint", admin, map[string]any{"reason": "重印"}, &er); code != 409 || er.Code != "label_cancelled" {
		t.Fatal(code, er)
	}
	var detail api.LabelDetail
	e.do("GET", "/labels/"+bar, admin, nil, &detail)
	if len(detail.Logs) != 3 || detail.Logs[0].OpType != "generate" || detail.Logs[1].OpType != "reprint" || detail.Logs[2].Reason != "损坏" {
		t.Fatal(detail.Logs)
	}
	e.do("GET", "/label-logs?keyword=P01", admin, nil, &page)
	if page.Total != 4 || page.Totals.CancelQty != 1 || page.Totals.ReprintQty != 1 {
		t.Fatal(page)
	}
	e.outsourceUser("other", e.outA)
	other := e.login("other", "pw-other")
	e.do("GET", "/label-logs", other, nil, &page)
	if page.Total != 0 {
		t.Fatal(page)
	}
}
