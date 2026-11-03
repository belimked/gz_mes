package server_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/oapi-codegen/runtime/types"
)

func TestOutsourceOrderScopeAndProgress(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	_, _, contract := e.batchContract(admin, "outsource-order")
	var b api.Batch
	if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: "OS-1"}, &b); code != 201 {
		t.Fatalf("batch: %d", code)
	}
	var items []api.BatchItem
	if code := e.do("PUT", fmt.Sprintf("/batches/%d/items", b.Id), admin, []api.BatchItemInput{{PieceNo: "P", Qty: 2, PieceArea: floatPtr(2)}}, &items); code != 200 {
		t.Fatalf("items: %d", code)
	}
	factory := e.outA
	if code := e.do("PUT", fmt.Sprintf("/batches/%d/schedule", b.Id), admin, api.BatchSchedule{OutsourceFactoryId: &factory}, &b); code != 200 {
		t.Fatalf("schedule: %d", code)
	}
	var gen api.LabelGenerateResult
	if code := e.do("POST", fmt.Sprintf("/batches/%d/labels/generate", b.Id), admin, nil, &gen); code != 200 {
		t.Fatalf("generate: %d", code)
	}
	var labels api.LabelPage
	e.do("GET", fmt.Sprintf("/labels?batchId=%d", b.Id), admin, nil, &labels)
	var scanned api.ScanResult
	e.do("POST", "/scans", admin, scanReq(labels.Items[0].BarCode, "10"), &scanned)
	e.outsourceUser("os-a", e.outA)
	e.outsourceUser("os-b", e.outB)
	var page api.OutsourceOrderPage
	if code := e.do("GET", "/reports/outsource-orders?keyword=OS-1", e.login("os-a", "pw-os-a"), nil, &page); code != 200 || page.Total != 1 || page.Items[0].Cutting.ScannedQty != 1 || page.Items[0].Cutting.UnscannedQty != 1 {
		t.Fatalf("outsource scope/progress: %d %+v", code, page)
	}
	if code := e.do("GET", "/reports/outsource-orders", e.login("os-b", "pw-os-b"), nil, &page); code != 200 || page.Total != 0 {
		t.Fatalf("other factory: %d %+v", code, page)
	}
}
func TestDeliveryPlanState(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	input := api.DeliveryPlanInput{FactoryId: e.plant, EstimatedDepartureDate: types.Date{Time: time.Now()}, NoteIds: []int64{}}
	var detail api.DeliveryPlanDetail
	if code := e.do("POST", "/delivery-plans", admin, input, &detail); code != 201 || detail.PlanNo == "" {
		t.Fatalf("create: %d %+v", code, detail)
	}
	var er api.Error
	path := fmt.Sprintf("/delivery-plans/%d", detail.Id)
	if code := e.do("POST", path+"/confirm", admin, map[string]any{"departureDate": time.Now().Format("2006-01-02")}, &er); code != 400 {
		t.Fatalf("empty confirm: %d %+v", code, er)
	}
	if code := e.do("PUT", path, admin, input, &detail); code != 200 {
		t.Fatalf("update: %d", code)
	}
	if code := e.do("DELETE", path, admin, nil, nil); code != 204 {
		t.Fatalf("delete: %d", code)
	}
}
