package server_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/oapi-codegen/runtime/types"
)

func TestBatchValuationFlow(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	_, _, contract := e.batchContract(admin, "valuation")
	var b api.Batch
	if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: "VAL-1", OrderArea: floatPtr(7.1234)}, &b); code != 201 {
		t.Fatalf("batch: %d", code)
	}
	path := fmt.Sprintf("/batches/%d", b.Id)
	var items []api.BatchItem
	if code := e.do("PUT", path+"/items", admin, []api.BatchItemInput{{PieceNo: "A", Qty: 2, PieceArea: floatPtr(1.5)}, {PieceNo: "B", Qty: 1, PieceArea: floatPtr(2)}}, &items); code != 200 {
		t.Fatalf("items: %d", code)
	}
	var page api.BatchValuationPage
	if code := e.do("GET", "/batch-valuations", admin, nil, &page); code != 200 || page.Total != 1 || len(page.Items) != 1 || page.Items[0].BatchId != b.Id || page.Items[0].Amount != 0 || page.Items[0].Audited || page.Items[0].Basis != "order_area" {
		t.Fatalf("initial list: %d %+v", code, page)
	}
	var v api.BatchValuation
	in := api.BatchValuationInput{Basis: "order_area", UnitPrice: 150}
	if code := e.do("PUT", path+"/valuation", admin, in, &v); code != 200 || math.Abs(v.Amount-math.Round(b.OrderArea*150*100)/100) > 0.001 || v.BasisArea != b.OrderArea {
		t.Fatalf("order: %d %+v", code, v)
	}
	in.Basis = "item_area"
	if code := e.do("PUT", path+"/valuation", admin, in, &v); code != 200 || v.BasisArea != 5 {
		t.Fatalf("item: %d %+v", code, v)
	}
	var gen api.LabelGenerateResult
	if code := e.do("POST", path+"/labels/generate", admin, nil, &gen); code != 200 || gen.Total != 3 {
		t.Fatalf("generate: %d %+v", code, gen)
	}
	var labels api.LabelPage
	if code := e.do("GET", fmt.Sprintf("/labels?batchId=%d", b.Id), admin, nil, &labels); code != 200 || len(labels.Items) != 3 {
		t.Fatalf("labels: %d %+v", code, labels)
	}
	codes := []string{labels.Items[0].BarCode, labels.Items[1].BarCode}
	for _, bar := range codes {
		for _, job := range []string{"10", "15", "20", "25", "30", "35", "40", "45", "50", "55"} {
			var scan api.ScanResult
			if code := e.do("POST", "/scans", admin, scanReq(bar, job), &scan); code != 201 {
				t.Fatalf("scan %s: %d", job, code)
			}
		}
	}
	var note api.DeliveryNoteDetail
	if code := e.do("POST", "/delivery-notes", admin, api.DeliveryNoteCreate{BatchId: b.Id, BarCodes: codes, DeliveredAt: types.Date{Time: time.Now()}}, &note); code != 201 {
		t.Fatalf("note: %d", code)
	}
	in.Basis = "delivered_area"
	if code := e.do("PUT", path+"/valuation", admin, in, &v); code != 200 || v.DeliveredQty != 2 || v.DeliveredArea <= 0 || v.BasisArea != note.Area || v.BasisArea != v.DeliveredArea {
		t.Fatalf("delivered: %d %+v note=%+v", code, v, note)
	}
	if code := e.do("POST", path+"/valuation/audit", admin, nil, &v); code != 200 || !v.Audited || v.AuditedBy == nil || v.AuditedAt == nil {
		t.Fatalf("audit: %d %+v", code, v)
	}
	var er api.Error
	if code := e.do("PUT", path+"/valuation", admin, in, &er); code != 409 || er.Code != "already_audited" {
		t.Fatalf("locked: %d %+v", code, er)
	}
	if code := e.do("POST", path+"/valuation/audit", admin, nil, &er); code != 409 || er.Code != "already_audited" {
		t.Fatalf("audit twice: %d %+v", code, er)
	}
	if code := e.do("GET", "/batch-valuations?audited=true", admin, nil, &page); code != 200 || page.Total != 1 || page.TotalAmount != v.Amount {
		t.Fatalf("audited list: %d %+v", code, page)
	}
	e.outsourceUser("valuation-wa", e.outA)
	wa := e.login("valuation-wa", "pw-valuation-wa")
	if code := e.do("GET", "/batch-valuations", wa, nil, &page); code != 200 || page.Total != 0 || len(page.Items) != 0 {
		t.Fatalf("scope: %d %+v", code, page)
	}
	var other api.Batch
	if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: "VAL-2"}, &other); code != 201 {
		t.Fatalf("other batch: %d", code)
	}
	if code := e.do("POST", fmt.Sprintf("/batches/%d/valuation/audit", other.Id), admin, nil, &er); code != 404 || er.Code != "not_found" {
		t.Fatalf("unvalued audit: %d %+v", code, er)
	}
}
