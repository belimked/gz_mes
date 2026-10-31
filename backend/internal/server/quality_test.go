package server_test

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
)

func TestQualityFlow(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	_, _, contract := e.batchContract(admin, "qc")
	var b api.Batch
	if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: "QC-1"}, &b); code != 201 {
		t.Fatalf("create batch: %d", code)
	}
	path := fmt.Sprintf("/batches/%d", b.Id)
	var items []api.BatchItem
	if code := e.do("PUT", path+"/items", admin, []api.BatchItemInput{{PieceNo: "Q", Qty: 3, PieceArea: floatPtr(2)}}, &items); code != 200 {
		t.Fatalf("items: %d", code)
	}
	var gen api.LabelGenerateResult
	if code := e.do("POST", path+"/labels/generate", admin, nil, &gen); code != 200 || gen.Created != 3 {
		t.Fatalf("generate: %d %+v", code, gen)
	}
	var page api.LabelPage
	if code := e.do("GET", fmt.Sprintf("/labels?batchId=%d", b.Id), admin, nil, &page); code != 200 || len(page.Items) != 3 {
		t.Fatalf("labels: %d", code)
	}
	var jobs []api.Job
	if code := e.do("GET", "/jobs", admin, nil, &jobs); code != 200 {
		t.Fatalf("jobs: %d", code)
	}
	qcCount := 0
	var job20 int64
	for _, j := range jobs {
		if j.IsQc {
			qcCount++
		}
		if j.Code == "20" {
			job20 = j.Id
		}
	}
	if qcCount != 2 {
		t.Fatalf("qc jobs: %d", qcCount)
	}
	// Move every label to 钣金下线 (20).
	for _, l := range page.Items {
		for _, job := range []string{"10", "15", "20"} {
			var r api.ScanResult
			if code := e.do("POST", "/scans", admin, scanReq(l.BarCode, job), &r); code != 201 {
				t.Fatalf("scan %s: %d", job, code)
			}
		}
	}
	var er api.Error
	// A QC job without result is rejected.
	if code := e.do("POST", "/scans", admin, api.ScanRequest{BarCode: page.Items[0].BarCode, JobCode: "25"}, &er); code != 400 || er.Code != "qc_result_required" {
		t.Fatalf("qc without result: %d %+v", code, er)
	}
	// Fail label 0 twice-ish: first fail then pass (复检); fail label 1 and leave it pending; pass label 2.
	fail := api.Fail
	pass := api.Pass
	cat, reason := "外观", "划伤"
	var r api.ScanResult
	if code := e.do("POST", "/scans", admin, api.ScanRequest{BarCode: page.Items[0].BarCode, JobCode: "25", Result: &fail, DefectJobId: &job20, DefectCategory: &cat, DefectReason: &reason}, &r); code != 201 {
		t.Fatalf("fail scan: %d", code)
	}
	if r.Scan != nil || r.Qc == nil || r.Qc.Result != api.Fail || r.Qc.DefectJobName == nil || *r.Qc.DefectJobName != "钣金下线" || r.Qc.Reinspected {
		t.Fatalf("fail result: scan=%v qc=%+v", r.Scan, r.Qc)
	}
	if r.Label.LastJobName == nil || *r.Label.LastJobName != "钣金下线" {
		t.Fatalf("failed piece should stay at 20: %+v", r.Label.LastJobName)
	}
	// The failed piece cannot skip to 30.
	if code := e.do("POST", "/scans", admin, scanReq(page.Items[0].BarCode, "30"), &er); code != 409 || er.Code != "out_of_order" {
		t.Fatalf("skip after fail: %d %+v", code, er)
	}
	// Bad defect job reference.
	bad := int64(999999)
	if code := e.do("POST", "/scans", admin, api.ScanRequest{BarCode: page.Items[0].BarCode, JobCode: "25", Result: &fail, DefectJobId: &bad}, &er); code != 400 {
		t.Fatalf("bad defect job: %d %+v", code, er)
	}
	time.Sleep(5 * time.Millisecond)
	if code := e.do("POST", "/scans", admin, api.ScanRequest{BarCode: page.Items[0].BarCode, JobCode: "25", Result: &pass}, &r); code != 201 || r.Scan == nil || r.Qc == nil || r.Qc.Result != api.Pass {
		t.Fatalf("re-pass: %d %+v", code, r)
	}
	if code := e.do("POST", "/scans", admin, api.ScanRequest{BarCode: page.Items[1].BarCode, JobCode: "25", Result: &fail, DefectJobId: &job20, DefectCategory: &cat, DefectReason: &reason}, &r); code != 201 {
		t.Fatalf("fail 2: %d", code)
	}
	if code := e.do("POST", "/scans", admin, scanReq(page.Items[2].BarCode, "25"), &r); code != 201 {
		t.Fatalf("pass 3: %d", code)
	}
	// Passing twice is already_scanned.
	if code := e.do("POST", "/scans", admin, scanReq(page.Items[2].BarCode, "25"), &er); code != 409 || er.Code != "already_scanned" {
		t.Fatalf("double pass: %d %+v", code, er)
	}

	// 质检扫描记录: 4 inspections — fail(0) reinspected, pass(0), fail(1) pending, pass(2).
	var qc api.QcRecordPage
	if code := e.do("GET", fmt.Sprintf("/qc-scans?batchId=%d", b.Id), admin, nil, &qc); code != 200 || qc.Total != 4 {
		t.Fatalf("qc list: %d %+v", code, qc)
	}
	s := qc.Summary
	if s.Inspected != 4 || s.Passed != 2 || s.Failed != 2 || s.Reinspected != 1 || s.Pending != 1 || s.FailedArea != 4 {
		t.Fatalf("qc summary: %+v", s)
	}
	if qc.Items[0].BarCode != page.Items[2].BarCode {
		t.Fatalf("newest first: %+v", qc.Items[0])
	}
	if code := e.do("GET", fmt.Sprintf("/qc-scans?batchId=%d&reinspected=true", b.Id), admin, nil, &qc); code != 200 || qc.Total != 1 || !qc.Items[0].Reinspected || qc.Items[0].ReinspectedAt == nil {
		t.Fatalf("reinspected filter: %d %+v", code, qc)
	}
	if code := e.do("GET", fmt.Sprintf("/qc-scans?batchId=%d&reinspected=false", b.Id), admin, nil, &qc); code != 200 || qc.Total != 1 || qc.Items[0].BarCode != page.Items[1].BarCode {
		t.Fatalf("pending filter: %d %+v", code, qc)
	}
	if code := e.do("GET", "/qc-scans?result=fail&keyword="+url.QueryEscape("划伤"), admin, nil, &qc); code != 200 || qc.Total != 2 {
		t.Fatalf("keyword filter: %d %+v", code, qc)
	}

	// 质量报表 (today): one group row (same date/job/batch/defect), qty 2, 1 reinspected.
	var qr api.QualityReport
	if code := e.do("GET", "/reports/quality", admin, nil, &qr); code != 200 || qr.Total != 1 || len(qr.Items) != 1 {
		t.Fatalf("quality report: %d %+v", code, qr)
	}
	if row := qr.Items[0]; row.Qty != 2 || row.Area != 4 || row.ReinspectedQty != 1 || row.DefectCategory != "外观" || row.PieceNos != "Q" {
		t.Fatalf("quality row: %+v", row)
	}
	if qr.Summary.Inspected != 4 || qr.Summary.PassRate != 50 {
		t.Fatalf("quality summary: %+v", qr.Summary)
	}
	if code := e.do("GET", "/reports/quality?from=2020-01-01&to=2019-01-01", admin, nil, &er); code != 400 {
		t.Fatalf("bad range: %d", code)
	}

	// 不良品日报: one batch, cells for 2 QC jobs.
	var dd api.DefectDailyReport
	if code := e.do("GET", "/reports/defect-daily", admin, nil, &dd); code != 200 || dd.Total != 1 || len(dd.Jobs) != 2 || len(dd.Items) != 1 {
		t.Fatalf("defect daily: %d %+v", code, dd)
	}
	row := dd.Items[0]
	if row.FailedQty != 2 || row.PendingQty != 1 || len(row.Cells) != 2 || row.Cells[0].FailedQty != 2 || row.Cells[0].ReinspectedQty != 1 || row.Cells[0].PendingQty != 1 || row.Cells[1].FailedQty != 0 {
		t.Fatalf("defect row: %+v cells=%+v", row, row.Cells)
	}
	if dd.Summary.FailedQty != 2 || dd.Summary.PendingArea != 2 {
		t.Fatalf("defect summary: %+v", dd.Summary)
	}
	if code := e.do("GET", "/reports/defect-daily?keyword=nothing-matches", admin, nil, &dd); code != 200 || dd.Total != 0 || len(dd.Items) != 0 {
		t.Fatalf("defect daily empty: %d %+v", code, dd)
	}

	// Scope: outsource user sees nothing.
	e.outsourceUser("qc-wa", e.outA)
	wa := e.login("qc-wa", "pw-qc-wa")
	if code := e.do("GET", "/qc-scans", wa, nil, &qc); code != 200 || qc.Total != 0 {
		t.Fatalf("scope: %d %+v", code, qc)
	}
}
