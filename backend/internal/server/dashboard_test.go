package server_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/oapi-codegen/runtime/types"
)

func TestDashboard(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	if _, err := pool.Exec(context.Background(), "UPDATE app_user SET staff_no='0001' WHERE username='admin'"); err != nil {
		t.Fatal(err)
	}
	var d api.Dashboard
	if code := e.do("GET", "/dashboard", admin, nil, &d); code != 200 || len(d.BatchStatus) != 4 || len(d.JobLoad) != 11 || len(d.ScanTrend) != 14 {
		t.Fatalf("empty dashboard: %d %+v", code, d)
	}
	if len(d.RecentActivity) != 0 {
		t.Fatalf("empty recent activity: %+v", d.RecentActivity)
	}
	if d.Alerts == nil || len(d.Alerts) != 0 {
		t.Fatalf("empty alerts must be []: %+v", d.Alerts)
	}
	for _, v := range d.BatchStatus {
		if v.Count != 0 {
			t.Fatalf("nonempty status: %+v", v)
		}
	}
	for _, v := range d.ScanTrend {
		if v.ActiveBatches != 0 || v.DoneBatches != 0 || v.DeliveredArea != 0 {
			t.Fatalf("nonempty scan trend: %+v", v)
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
	if _, err := pool.Exec(context.Background(), "UPDATE batch SET status='scheduled' WHERE id=$1", b.Id); err != nil {
		t.Fatal(err)
	}
	if code := e.do("GET", "/dashboard", admin, nil, &d); code != 200 {
		t.Fatalf("dashboard: %d", code)
	}
	foundOverdue := false
	for _, alert := range d.Alerts {
		if alert.Rule == api.Overdue && alert.BatchId != nil && *alert.BatchId == b.Id && alert.Days != nil && *alert.Days == 1 {
			foundOverdue = true
		}
	}
	if !foundOverdue {
		t.Errorf("missing overdue alert: %+v", d.Alerts)
	}
	if d.Kpi.InProduction != 1 || d.Kpi.PendingSchedule != 0 || d.Kpi.UrgentOpen != 1 || d.Kpi.OverdueOpen != 1 || d.Kpi.TodayScanQty != 4 || d.Kpi.LabelQty != 2 || d.Kpi.TodayScanArea != 6 {
		t.Errorf("kpi: %+v", d.Kpi)
	}
	if len(d.ActiveBatches) != 1 || d.ActiveBatches[0].CurrentSeq != 2 || d.ActiveBatches[0].Progress <= 0 {
		t.Errorf("active: %+v", d.ActiveBatches)
	}
	today := d.ScanTrend[len(d.ScanTrend)-1]
	if today.Qty != 4 || today.ActiveBatches != 1 || today.DoneBatches != 0 || today.DeliveredArea != 0 {
		t.Errorf("scan trend today: %+v", today)
	}
	if len(d.CustomerArea) == 0 || len(d.RecentScans) != 4 {
		t.Errorf("customer / scans: %+v %+v", d.CustomerArea, d.RecentScans)
	}
	if len(d.RecentActivity) == 0 {
		t.Fatal("missing recent activity")
	}
	if first := d.RecentActivity[0]; first.Qty < 1 || first.LastScannedAt.IsZero() || first.StaffNo != "0001" {
		t.Errorf("recent activity first: %+v", first)
	}
	activityQty := 0
	for _, v := range d.RecentActivity {
		activityQty += v.Qty
	}
	if activityQty != d.Kpi.TodayScanQty {
		t.Errorf("recent activity qty: %d, scans: %d", activityQty, d.Kpi.TodayScanQty)
	}
	for _, j := range d.JobLoad {
		switch j.JobCode {
		case "10":
			if j.Qty != 0 || j.CycleSamples != 2 || j.CycleHours == nil || *j.CycleHours < 0 {
				t.Errorf("10: %+v", j)
			}
		case "15":
			if j.Qty != 2 || j.Area != 3 || j.CycleSamples != 2 || j.DwellHours == nil || j.DwellMaxHours == nil {
				t.Errorf("15: %+v", j)
			} else if *j.DwellMaxHours < *j.DwellHours {
				t.Errorf("15 dwell max below average: %+v", j)
			}
		case "20":
			if j.DwellHours != nil {
				t.Errorf("20: %+v", j)
			}
		}
	}
	e.outsourceUser("dashboard-wa", e.outA)
	d = api.Dashboard{} // Optional JSON fields are omitted; discard values from the previous response.
	if code := e.do("GET", "/dashboard", e.login("dashboard-wa", "pw-dashboard-wa"), nil, &d); code != 200 || d.Kpi.InProduction != 0 || len(d.RecentScans) != 0 || len(d.RecentActivity) != 0 || len(d.Alerts) != 0 {
		t.Errorf("scope: %d %+v", code, d)
	}
	for _, v := range d.ScanTrend {
		if v.ActiveBatches != 0 || v.DoneBatches != 0 || v.DeliveredArea != 0 {
			t.Errorf("scoped scan trend: %+v", v)
		}
	}
	if len(d.JobLoad) != 11 {
		t.Errorf("scoped job load: %+v", d.JobLoad)
	}
	for _, j := range d.JobLoad {
		if j.Qty != 0 || j.CycleSamples != 0 || j.CycleHours != nil || j.CycleP90Hours != nil || j.DwellHours != nil || j.DwellMaxHours != nil {
			t.Errorf("scoped job load: %+v", j)
		}
	}
}
