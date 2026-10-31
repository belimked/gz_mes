package server_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/belimked/gz_mes/backend/internal/api"
)

func TestLabelAndScanFlow(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	_, _, contract := e.batchContract(admin, "labels")
	create := func(lot string) api.Batch {
		t.Helper()
		var b api.Batch
		if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: lot}, &b); code != 201 {
			t.Fatalf("create batch: %d", code)
		}
		return b
	}
	b := create("LABEL-1")
	path := fmt.Sprintf("/batches/%d", b.Id)
	var items []api.BatchItem
	input := []api.BatchItemInput{{PieceNo: "A", Qty: 2, PieceArea: floatPtr(1.5)}, {PieceNo: "B", Qty: 2, PieceArea: floatPtr(2)}, {PieceNo: "C", Qty: 1, PieceArea: floatPtr(3)}}
	if code := e.do("PUT", path+"/items", admin, input, &items); code != 200 {
		t.Fatalf("items: %d", code)
	}
	var result api.LabelGenerateResult
	if code := e.do("POST", path+"/labels/generate", admin, nil, &result); code != 200 || result.Created != 5 || result.Total != 5 {
		t.Fatalf("generate: %d %+v", code, result)
	}
	if code := e.do("POST", path+"/labels/generate", admin, nil, &result); code != 200 || result.Created != 0 || result.Total != 5 {
		t.Fatalf("regenerate: %d %+v", code, result)
	}
	var er api.Error
	if code := e.do("PUT", path+"/items", admin, input, &er); code != 409 || er.Code != "has_labels" {
		t.Fatalf("replace labelled: %d %+v", code, er)
	}
	var page api.LabelPage
	if code := e.do("GET", fmt.Sprintf("/labels?batchId=%d", b.Id), admin, nil, &page); code != 200 || page.Total != 5 || len(page.Items) != 5 {
		t.Fatalf("labels: %d %+v", code, page)
	}
	for _, l := range page.Items {
		if len(l.BarCode) != 13 {
			t.Fatalf("barcode %q", l.BarCode)
		}
		for _, r := range l.BarCode {
			if r < '0' || r > '9' {
				t.Fatalf("barcode %q", l.BarCode)
			}
		}
	}
	barcode := page.Items[0].BarCode
	scan := func(code string) api.ScanResult {
		t.Helper()
		var r api.ScanResult
		if status := e.do("POST", "/scans", admin, api.ScanRequest{BarCode: barcode, JobCode: code}, &r); status != 201 {
			t.Fatalf("scan %s: %d", code, status)
		}
		return r
	}
	if code := e.do("POST", "/scans", admin, api.ScanRequest{BarCode: barcode, JobCode: "20"}, &er); code != 409 || er.Code != "out_of_order" || !strings.Contains(er.Message, "折弯") {
		t.Fatalf("out of order: %d %+v", code, er)
	}
	first := scan("10")
	if first.Progress.Status != "in_production" || first.Progress.Jobs[0].ScannedQty != 1 {
		t.Fatalf("first: %+v", first.Progress)
	}
	if code := e.do("POST", "/scans", admin, api.ScanRequest{BarCode: barcode, JobCode: "10"}, &er); code != 409 || er.Code != "already_scanned" {
		t.Fatalf("duplicate: %d %+v", code, er)
	}
	e.outsourceUser("label-wa", e.outA)
	wa := e.login("label-wa", "pw-label-wa")
	if code := e.do("POST", "/scans", wa, api.ScanRequest{BarCode: barcode, JobCode: "15"}, &er); code != 404 || er.Code != "not_found" {
		t.Fatalf("scope: %d %+v", code, er)
	}
	codes := []string{"10", "15", "20", "25", "30", "35", "40", "45", "50", "55", "60"}
	for i, l := range page.Items {
		barcode = l.BarCode
		for j, code := range codes {
			if i == 0 && j == 0 {
				continue
			}
			scan(code)
		}
	}
	var progress api.BatchProgress
	if code := e.do("GET", path+"/progress", admin, nil, &progress); code != 200 || progress.Status != "done" || progress.LabelQty != 5 || len(progress.Jobs) != 11 || progress.Jobs[10].ScannedQty != 5 {
		t.Fatalf("progress: %d %+v", code, progress)
	}
	var scans api.ScanPage
	if code := e.do("GET", fmt.Sprintf("/scans?batchId=%d", b.Id), admin, nil, &scans); code != 200 || scans.Total != 55 {
		t.Fatalf("scans: %d %+v", code, scans)
	}
	if s0 := scans.Items[0]; s0.JobName != "发货对账" || s0.ProjectName == "" || s0.ScannedBy != "管理员" || s0.PieceArea <= 0 {
		t.Fatalf("scan row (newest first, enriched): %+v", s0)
	}
	if code := e.do("GET", fmt.Sprintf("/scans?batchId=%d&keyword=%s&staff=%s", b.Id, page.Items[1].BarCode, url.QueryEscape("管理")), admin, nil, &scans); code != 200 || scans.Total != 11 {
		t.Fatalf("scans keyword/staff: %d %+v", code, scans)
	}
	if code := e.do("GET", "/scans?staff=nobody-here", admin, nil, &scans); code != 200 || scans.Total != 0 {
		t.Fatalf("scans staff miss: %d %+v", code, scans)
	}
	barcode = page.Items[0].BarCode
	if code := e.do("POST", "/labels/"+barcode+"/cancel", admin, nil, &er); code != 409 || er.Code != "has_scans" {
		t.Fatalf("cancel scanned: %d %+v", code, er)
	}
	var detail api.LabelDetail
	if code := e.do("GET", "/labels/"+barcode, admin, nil, &detail); code != 200 || len(detail.Scans) != 11 {
		t.Fatalf("detail: %d %+v", code, detail)
	}
	empty := create("LABEL-EMPTY")
	if code := e.do("POST", fmt.Sprintf("/batches/%d/labels/generate", empty.Id), admin, nil, &er); code != 400 || er.Code != "no_items" {
		t.Fatalf("empty: %d %+v", code, er)
	}
	another := create("LABEL-2")
	anotherPath := fmt.Sprintf("/batches/%d", another.Id)
	if code := e.do("PUT", anotherPath+"/items", admin, []api.BatchItemInput{{PieceNo: "ONE", Qty: 1}}, &items); code != 200 {
		t.Fatalf("new items: %d", code)
	}
	if code := e.do("POST", anotherPath+"/labels/generate", admin, nil, &result); code != 200 || result.Total != 1 {
		t.Fatalf("new labels: %d %+v", code, result)
	}
	if code := e.do("GET", fmt.Sprintf("/labels?batchId=%d", another.Id), admin, nil, &page); code != 200 || len(page.Items) != 1 {
		t.Fatalf("new list: %d %+v", code, page)
	}
	barcode = page.Items[0].BarCode
	var cancelled api.Label
	if code := e.do("POST", "/labels/"+barcode+"/cancel", admin, nil, &cancelled); code != 200 || !cancelled.Cancelled {
		t.Fatalf("cancel: %d %+v", code, cancelled)
	}
	if code := e.do("POST", "/scans", admin, api.ScanRequest{BarCode: barcode, JobCode: "10"}, &er); code != 409 || er.Code != "label_cancelled" {
		t.Fatalf("cancelled scan: %d %+v", code, er)
	}
}
