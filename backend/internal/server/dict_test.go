package server_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/oapi-codegen/runtime/types"
)

func TestDictLifecycle(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	var list []api.DictItem
	if code := e.do("GET", "/dicts?kind=shift", admin, nil, &list); code != 200 || len(list) != 3 || list[0].Name != "早班" || list[1].Name != "中班" {
		t.Fatalf("seed: %d %+v", code, list)
	}
	in := api.DictInput{Kind: "workshop", Name: "测试车间", Sort: 0}
	var d api.DictItem
	if code := e.do("POST", "/dicts", admin, in, &d); code != 201 {
		t.Fatalf("create %d", code)
	}
	if code := e.do("POST", "/dicts", admin, in, nil); code != 409 {
		t.Fatalf("duplicate %d", code)
	}
	e.outsourceUser("dict-restricted", e.outA)
	restricted := e.login("dict-restricted", "pw-dict-restricted")
	if code := e.do("GET", "/dicts", restricted, nil, &list); code != 200 {
		t.Fatalf("read %d", code)
	}
	if code := e.do("POST", "/dicts", restricted, in, nil); code != 403 {
		t.Fatalf("write %d", code)
	}
	_, _, contract := e.batchContract(admin, "dict-test")
	var batch api.Batch
	if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: "DICT-LOT"}, &batch); code != 201 {
		t.Fatal(code)
	}
	var er api.Error
	path := fmt.Sprintf("/batches/%d/schedule", batch.Id)
	if code := e.do("PUT", path, admin, api.BatchSchedule{Workshop: strPtr("未知车间")}, &er); code != 400 || er.Code != "invalid_request" {
		t.Fatalf("bad batch %d %+v", code, er)
	}
	if code := e.do("PUT", path, admin, api.BatchSchedule{Workshop: &in.Name}, nil); code != 200 {
		t.Fatalf("batch %d", code)
	}
	date := types.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	var plan api.Plan
	if code := e.do("POST", "/plans", admin, api.PlanCreate{Kind: api.SheetWeek, BatchId: batch.Id, StartDate: date, EndDate: date, Workshop: &in.Name}, &plan); code != 201 {
		t.Fatalf("plan %d", code)
	}
	in.Name = "重命名车间"
	if code := e.do("PUT", fmt.Sprintf("/dicts/%d", d.Id), admin, in, &d); code != 200 {
		t.Fatalf("rename %d", code)
	}
	if code := e.do("GET", fmt.Sprintf("/batches/%d", batch.Id), admin, nil, &batch); code != 200 || batch.Workshop != in.Name {
		t.Fatalf("cascade batch %d %+v", code, batch)
	}
	if code := e.do("GET", fmt.Sprintf("/plans/%d", plan.Id), admin, nil, &plan); code != 200 || plan.Workshop != in.Name {
		t.Fatalf("cascade plan %d %+v", code, plan)
	}
	in.IsStop = true
	if code := e.do("PUT", fmt.Sprintf("/dicts/%d", d.Id), admin, in, &d); code != 200 {
		t.Fatalf("stop %d", code)
	}
	if code := e.do("GET", "/dicts?kind=workshop", admin, nil, &list); code != 200 || len(list) != 2 {
		t.Fatalf("stopped hidden %d %+v", code, list)
	}
	if code := e.do("GET", "/dicts?kind=workshop&includeStopped=true", admin, nil, &list); code != 200 || len(list) != 3 || list[0].Id != d.Id {
		t.Fatalf("stopped listed %d %+v", code, list)
	}
	if code := e.do("POST", "/plans", admin, api.PlanCreate{Kind: api.SheetDaily, BatchId: batch.Id, StartDate: date, EndDate: date, Workshop: &in.Name}, &er); code != 400 || er.Code != "invalid_request" {
		t.Fatalf("stopped plan %d %+v", code, er)
	}
	if code := e.do("PUT", fmt.Sprintf("/plans/%d", plan.Id), admin, api.PlanUpdate{StartDate: date, EndDate: date, Workshop: &in.Name}, &plan); code != 200 {
		t.Fatalf("unchanged plan %d", code)
	}
	if code := e.do("PUT", path, admin, api.BatchSchedule{Workshop: &in.Name}, nil); code != 200 {
		t.Fatalf("unchanged batch %d", code)
	}
}
