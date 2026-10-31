package server_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/oapi-codegen/runtime/types"
)

func TestDashboard(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	var d api.Dashboard
	if code := e.do("GET", "/dashboard", admin, nil, &d); code != 200 || len(d.BatchStatus) != 4 || len(d.JobLoad) != 11 || len(d.ScanTrend) != 14 {
		t.Fatalf("empty dashboard: %d %+v", code, d)
	}
	for _, v := range d.BatchStatus {
		if v.Count != 0 {
			t.Fatalf("nonempty status: %+v", v)
		}
	}
	if code := e.do("GET", "/dashboard?days=5", admin, nil, nil); code != 400 {
		t.Fatalf("invalid days: %d", code)
	}
	if code := e.do("GET", "/dashboard?days=7", admin, nil, &d); code != 200 || len(d.ScanTrend) != 7 {
		t.Fatalf("seven days: %d %+v", code, d.ScanTrend)
	}
	_, _, contract := e.batchContract(admin, "dashboard")
	urgent := api.Urgency("urgent")
	yesterday := types.Date{Time: time.Now().AddDate(0, 0, -1)}
	var b api.Batch
	if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: "DASH-1", Urgency: &urgent}, &b); code != 201 {
		t.Fatalf("create active: %d", code)
	}
	path := fmt.Sprintf("/batches/%d", b.Id)
	var items []api.BatchItem
	if code := e.do("PUT", path+"/items", admin, []api.BatchItemInput{{PieceNo: "A", Qty: 2, PieceArea: floatPtr(1.5)}}, &items); code != 200 {
		t.Fatalf("items: %d", code)
	}
	var gen api.LabelGenerateResult
	if code := e.do("POST", path+"/labels/generate", admin, nil, &gen); code != 200 || gen.Total != 2 {
		t.Fatalf("labels: %d %+v", code, gen)
	}
	var labels api.LabelPage
	if code := e.do("GET", fmt.Sprintf("/labels?batchId=%d", b.Id), admin, nil, &labels); code != 200 || len(labels.Items) != 2 {
		t.Fatalf("list labels: %d %+v", code, labels)
	}
	for _, l := range labels.Items {
		for _, job := range []string{"10", "15"} {
			var scan api.ScanResult
			if code := e.do("POST", "/scans", admin, api.ScanRequest{BarCode: l.BarCode, JobCode: job}, &scan); code != 201 {
				t.Fatalf("scan %s: %d", job, code)
			}
		}
	}
	if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: "DASH-2", RequiredDate: &yesterday}, &b); code != 201 {
		t.Fatalf("create overdue: %d", code)
	}
	if code := e.do("GET", "/dashboard", admin, nil, &d); code != 200 {
		t.Fatalf("dashboard: %d", code)
	}
	if d.Kpi.InProduction != 1 || d.Kpi.PendingSchedule != 1 || d.Kpi.UrgentOpen != 1 || d.Kpi.OverdueOpen != 1 || d.Kpi.TodayScanQty != 4 || d.Kpi.LabelQty != 2 || d.Kpi.TodayScanArea != 6 {
		t.Errorf("kpi: %+v", d.Kpi)
	}
	if len(d.ActiveBatches) != 1 || d.ActiveBatches[0].CurrentSeq != 2 || d.ActiveBatches[0].Progress <= 0 {
		t.Errorf("active: %+v", d.ActiveBatches)
	}
	if len(d.CustomerArea) == 0 || len(d.RecentScans) != 4 {
		t.Errorf("customer / scans: %+v %+v", d.CustomerArea, d.RecentScans)
	}
	for _, j := range d.JobLoad {
		if j.JobCode == "15" && (j.Qty != 2 || j.Area != 3) {
			t.Errorf("15: %+v", j)
		}
		if j.JobCode == "10" && j.Qty != 0 {
			t.Errorf("10: %+v", j)
		}
	}
	e.outsourceUser("dashboard-wa", e.outA)
	if code := e.do("GET", "/dashboard", e.login("dashboard-wa", "pw-dashboard-wa"), nil, &d); code != 200 || d.Kpi.InProduction != 0 || len(d.RecentScans) != 0 {
		t.Errorf("scope: %d %+v", code, d)
	}
}
