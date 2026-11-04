package server_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/oapi-codegen/runtime/types"
)

func TestInventoryAndProjectSummary(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	_, _, contract := e.batchContract(admin, "inventory")
	codes := make([]string, 0)
	for n := 1; n <= 2; n++ {
		var b api.Batch
		if status := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: fmt.Sprintf("INV-%d", n)}, &b); status != 201 {
			t.Fatal(status)
		}
		path := fmt.Sprintf("/batches/%d", b.Id)
		if status := e.do("PUT", path+"/items", admin, []api.BatchItemInput{{PieceNo: "P", Qty: 2, PieceArea: floatPtr(1.5)}}, nil); status != 200 {
			t.Fatal(status)
		}
		e.do("POST", path+"/labels/generate", admin, nil, nil)
		var labels api.LabelPage
		e.do("GET", fmt.Sprintf("/labels?batchId=%d", b.Id), admin, nil, &labels)
		for _, l := range labels.Items {
			codes = append(codes, l.BarCode)
		}
	}
	// First label remains semi, second reaches finished, third is shipped, fourth is cancelled.
	for i, bar := range codes {
		if i == 3 {
			if status := e.do("POST", "/labels/"+bar+"/cancel", admin, map[string]any{"reason": "damaged"}, nil); status != 200 {
				t.Fatal(status)
			}
			continue
		}
		steps := []string{"10", "15", "20"}
		if i > 0 {
			steps = append(steps, "25", "30", "35", "40")
		}
		if i == 2 {
			steps = append(steps, "45", "50", "55")
		}
		for _, job := range steps {
			var result api.ScanResult
			if status := e.do("POST", "/scans", admin, scanReq(bar, job), &result); status != 201 {
				t.Fatalf("%s: %d", job, status)
			}
		}
	}
	var labels api.LabelPage
	e.do("GET", "/labels?keyword=INV-2", admin, nil, &labels)
	var b api.BatchPage
	e.do("GET", "/batches?keyword=INV-2", admin, nil, &b)
	if len(b.Items) != 1 {
		t.Fatal(b)
	}
	var note api.DeliveryNoteDetail
	if status := e.do("POST", "/delivery-notes", admin, api.DeliveryNoteCreate{BatchId: b.Items[0].Id, BarCodes: []string{codes[2]}, DeliveredAt: types.Date{Time: time.Now()}}, &note); status != 201 {
		t.Fatal(status)
	}
	today := time.Now().UTC().Format("2006-01-02")
	for _, tc := range []struct {
		kind           string
		stock, in, out int
	}{{"semi", 1, 3, 2}, {"finished", 1, 2, 1}} {
		var report api.InventoryReport
		path := "/inventory?kind=" + tc.kind + "&from=" + today + "&to=" + today
		if status := e.do("GET", path, admin, nil, &report); status != 200 {
			t.Fatal(status)
		}
		if report.Totals.StockQty != tc.stock || report.Totals.InQty != tc.in || report.Totals.OutQty != tc.out || report.Totals.StockArea != 1.5 {
			t.Fatalf("%s %+v", tc.kind, report)
		}
		e.do("GET", "/inventory?kind="+tc.kind+"&from=2000-01-01&to=2000-01-02", admin, nil, &report)
		if report.Total != 1 || report.Totals.InQty != 0 || report.Totals.OutQty != 0 {
			t.Fatal(report)
		}
		e.do("GET", "/inventory?kind="+tc.kind+"&hasStock=false", admin, nil, &report)
		if report.Total != 2 || report.Totals.InQty != 0 {
			t.Fatal(report)
		}
	}
	var projects api.ProjectSummaryReport
	if status := e.do("GET", "/reports/project-summary?keyword=inventory", admin, nil, &projects); status != 200 {
		t.Fatal(status)
	}
	if projects.Total != 1 || projects.Items[0].BatchCount != 2 || projects.Items[0].PlanQty != 4 || projects.Items[0].LabelQty != 3 || projects.Items[0].Status != "in_production" {
		t.Fatal(projects)
	}
	for _, cell := range projects.Items[0].Cells {
		for _, job := range projects.Jobs {
			if job.Id == cell.JobId && job.Code == "20" && cell.Qty != 3 {
				t.Fatal(cell)
			}
		}
	}
	e.do("GET", "/reports/project-summary?keyword=inventory&status=done", admin, nil, &projects)
	if projects.Total != 0 {
		t.Fatal(projects)
	}
	e.do("GET", "/reports/project-summary?keyword=inventory&status=not_done&from=2000-01-01&to=2000-01-02", admin, nil, &projects)
	if projects.Total != 0 {
		t.Fatal(projects)
	}
	e.do("GET", "/reports/project-summary?keyword=inventory&status=not_done&size=1", admin, nil, &projects)
	if projects.Total != 1 || projects.Summary.LabelQty != 3 {
		t.Fatal(projects)
	}
	// Cancelling a scanned label removes it from both stock and period movements.
	// Historical data may contain scans on a subsequently cancelled label.
	if _, err := pool.Exec(context.Background(), "UPDATE label SET cancelled=true WHERE bar_code=$1", codes[0]); err != nil {
		t.Fatal(err)
	}
	var after api.InventoryReport
	e.do("GET", "/inventory?kind=semi&from="+today+"&to="+today, admin, nil, &after)
	if after.Totals.StockQty != 0 || after.Totals.InQty != 2 || after.Totals.OutQty != 2 {
		t.Fatal(after)
	}
	e.outsourceUser("inventory-other", e.outA)
	other := e.login("inventory-other", "pw-inventory-other")
	var hidden api.InventoryReport
	e.do("GET", "/inventory?kind=semi", other, nil, &hidden)
	if hidden.Total != 0 {
		t.Fatal(hidden)
	}
	e.do("GET", "/reports/project-summary?keyword=inventory", other, nil, &projects)
	if projects.Total != 0 {
		t.Fatal(projects)
	}
}
