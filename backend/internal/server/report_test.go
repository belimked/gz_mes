package server_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
)

func TestReports(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	_, _, contract := e.batchContract(admin, "rep")
	var b1, b2 api.Batch
	if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: "REP-1", ClientOrderNo: strPtr("PO-77")}, &b1); code != 201 {
		t.Fatalf("batch1: %d", code)
	}
	if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: "REP-2"}, &b2); code != 201 {
		t.Fatalf("batch2: %d", code)
	}
	items := []api.BatchItemInput{{PieceNo: "A", Qty: 2, PieceArea: floatPtr(1.5), SprayingArea: floatPtr(3), WeldLine: floatPtr(0.5)}, {PieceNo: "B", Qty: 1, PieceArea: floatPtr(2), SprayingArea: floatPtr(4), WeldLine: floatPtr(1)}}
	for _, b := range []api.Batch{b1, b2} {
		var out []api.BatchItem
		if code := e.do("PUT", fmt.Sprintf("/batches/%d/items", b.Id), admin, items, &out); code != 200 {
			t.Fatalf("items: %d", code)
		}
	}
	var gen api.LabelGenerateResult
	if code := e.do("POST", fmt.Sprintf("/batches/%d/labels/generate", b1.Id), admin, nil, &gen); code != 200 || gen.Total != 3 {
		t.Fatalf("generate: %d %+v", code, gen)
	}
	var labels api.LabelPage
	e.do("GET", fmt.Sprintf("/labels?batchId=%d", b1.Id), admin, nil, &labels)
	// All 3 labels through 开料, 2 of them through 折弯.
	for i, l := range labels.Items {
		for _, job := range []string{"10", "15"} {
			if job == "15" && i == 2 {
				continue
			}
			var r api.ScanResult
			if code := e.do("POST", "/scans", admin, scanReq(l.BarCode, job), &r); code != 201 {
				t.Fatalf("scan %s: %d", job, code)
			}
		}
	}
	var jobs []api.Job
	e.do("GET", "/jobs", admin, nil, &jobs)
	jobID := map[string]int64{}
	for _, j := range jobs {
		jobID[j.Code] = j.Id
	}
	cell := func(r api.BatchJobRow, code string) api.JobCell {
		for _, c := range r.Cells {
			if c.JobId == jobID[code] {
				return c
			}
		}
		t.Fatalf("no cell %s", code)
		return api.JobCell{}
	}
	today := time.Now().Format("2006-01-02")

	// 工程日报: only batches scanned today.
	var rep api.BatchJobReport
	if code := e.do("GET", "/reports/batch-jobs?mode=daily", admin, nil, &rep); code != 200 || len(rep.Jobs) != 11 || rep.Total != 1 || len(rep.Items) != 1 || rep.Summary.Batches != 1 || rep.Summary.LabelQty != 3 || rep.Summary.ItemQty != 3 || rep.Summary.ItemArea != 5 {
		t.Fatalf("daily: %d %+v", code, rep)
	}
	r := rep.Items[0]
	if r.BatchId != b1.Id || r.LabelQty != 3 || r.CurrentJobName != "折弯" || r.LastScanAt == nil || len(r.Cells) != 11 || r.ClientOrderNo != "PO-77" {
		t.Fatalf("daily row: %+v", r)
	}
	if c := cell(r, "10"); c.Qty != 3 || c.Area != 5 || c.Pct != 100 {
		t.Fatalf("cell 10: %+v", c)
	}
	if c := cell(r, "15"); c.Qty != 2 || c.Pct != 66 || c.Area < 3 || c.Area > 3.5 {
		t.Fatalf("cell 15: %+v", c)
	}
	if c := cell(r, "20"); c.Qty != 0 || c.Pct != 0 {
		t.Fatalf("cell 20: %+v", c)
	}
	if len(rep.Summary.Cells) != 11 || rep.Summary.Cells[0].Qty != 3 || rep.Summary.Cells[1].Qty != 2 || rep.Summary.Cells[1].Pct != 66 || rep.Summary.Cells[2].Qty != 0 {
		t.Fatalf("daily summary cells: %+v", rep.Summary.Cells)
	}
	if code := e.do("GET", "/reports/batch-jobs?mode=daily&from=2020-01-01&to=2020-01-02", admin, nil, &rep); code != 200 || rep.Total != 0 || len(rep.Items) != 0 || rep.Summary.Cells[0].Qty != 0 {
		t.Fatalf("daily past: %d %+v", code, rep)
	}
	if code := e.do("GET", fmt.Sprintf("/reports/batch-jobs?mode=daily&jobId=%d", jobID["20"]), admin, nil, &rep); code != 200 || rep.Total != 0 {
		t.Fatalf("daily job filter: %d %+v", code, rep)
	}
	if code := e.do("GET", fmt.Sprintf("/reports/batch-jobs?mode=daily&jobId=%d&keyword=PO-77", jobID["15"]), admin, nil, &rep); code != 200 || rep.Total != 1 {
		t.Fatalf("daily job+kw filter: %d %+v", code, rep)
	}
	var er api.Error
	if code := e.do("GET", "/reports/batch-jobs?mode=daily&from=2026-02-01&to=2026-01-01", admin, nil, &er); code != 400 {
		t.Fatalf("bad range: %d", code)
	}
	if code := e.do("GET", "/reports/batch-jobs?mode=weekly", admin, nil, &er); code != 400 {
		t.Fatalf("bad mode: %d", code)
	}

	// 工程进度: every batch in scope, unscanned ones included.
	if code := e.do("GET", "/reports/batch-jobs?mode=progress", admin, nil, &rep); code != 200 || rep.Total != 2 || rep.Summary.LabelQty != 3 || rep.Summary.ItemQty != 6 {
		t.Fatalf("progress: %d %+v", code, rep)
	}
	var unscanned api.BatchJobReport // fresh value: nil lastScanAt is omitted from JSON
	if code := e.do("GET", "/reports/batch-jobs?mode=progress&scanned=false", admin, nil, &unscanned); code != 200 || unscanned.Total != 1 || unscanned.Items[0].BatchId != b2.Id || unscanned.Items[0].CurrentJobName != "" || unscanned.Items[0].LastScanAt != nil {
		t.Fatalf("progress unscanned: %d %+v", code, unscanned)
	}
	if code := e.do("GET", "/reports/batch-jobs?mode=progress&scanned=true&status=in_production", admin, nil, &rep); code != 200 || rep.Total != 1 || rep.Items[0].BatchId != b1.Id {
		t.Fatalf("progress scanned: %d %+v", code, rep)
	}
	if code := e.do("GET", "/reports/batch-jobs?mode=progress&from="+today+"&to="+today, admin, nil, &rep); code != 200 || rep.Total != 2 {
		t.Fatalf("progress created window: %d %+v", code, rep)
	}
	if code := e.do("GET", "/reports/batch-jobs?mode=progress&to=2020-01-01", admin, nil, &rep); code != 200 || rep.Total != 0 {
		t.Fatalf("progress created past: %d %+v", code, rep)
	}

	// 员工产量: admin scanned 5 pieces (3×开料 + 2×折弯).
	var so api.StaffOutputReport
	if code := e.do("GET", "/reports/staff-output", admin, nil, &so); code != 200 || so.Total != 2 || so.Summary.Staff != 1 || so.Summary.Qty != 5 || so.Summary.Area != 8.5 || so.Summary.SprayArea != 17 || so.Summary.WeldLine != 3.5 {
		t.Fatalf("staff summary: %d %+v", code, so)
	}
	if row := so.Items[0]; row.JobName != "开料" || row.Qty != 3 || row.Area != 5 || row.StaffNo != "" || row.DisplayName == "" || row.BatchId != nil || row.FirstScanAt.After(row.LastScanAt) {
		t.Fatalf("staff row: %+v", row)
	}
	if code := e.do("GET", "/reports/staff-output?detail=true", admin, nil, &so); code != 200 || so.Total != 2 || so.Items[0].BatchId == nil || *so.Items[0].BatchId != b1.Id || so.Items[0].LotNo == nil || *so.Items[0].LotNo != "REP-1" {
		t.Fatalf("staff detail: %d %+v", code, so)
	}
	if code := e.do("GET", fmt.Sprintf("/reports/staff-output?jobId=%d", jobID["15"]), admin, nil, &so); code != 200 || so.Total != 1 || so.Summary.Qty != 2 {
		t.Fatalf("staff job: %d %+v", code, so)
	}
	if code := e.do("GET", "/reports/staff-output?staff=admin&batchKeyword=REP-1", admin, nil, &so); code != 200 || so.Total != 2 {
		t.Fatalf("staff kw: %d %+v", code, so)
	}
	if code := e.do("GET", "/reports/staff-output?staff=nobody", admin, nil, &so); code != 200 || so.Total != 0 || so.Summary.Qty != 0 {
		t.Fatalf("staff none: %d %+v", code, so)
	}
	if code := e.do("GET", "/reports/staff-output?from=2020-01-01&to=2020-01-02", admin, nil, &so); code != 200 || so.Total != 0 {
		t.Fatalf("staff past: %d %+v", code, so)
	}
	if code := e.do("GET", "/reports/staff-output?from=2026-02-01&to=2026-01-01", admin, nil, &er); code != 400 {
		t.Fatalf("staff bad range: %d", code)
	}

	// Scope: an outsource account sees nothing of the plant.
	e.outsourceUser("rep-wa", e.outA)
	wa := e.login("rep-wa", "pw-rep-wa")
	if code := e.do("GET", "/reports/batch-jobs?mode=progress", wa, nil, &rep); code != 200 || rep.Total != 0 || len(rep.Jobs) != 11 {
		t.Fatalf("scope batch-jobs: %d %+v", code, rep)
	}
	if code := e.do("GET", "/reports/staff-output", wa, nil, &so); code != 200 || so.Total != 0 {
		t.Fatalf("scope staff: %d %+v", code, so)
	}
}
