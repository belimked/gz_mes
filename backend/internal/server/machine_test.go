package server_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
)

func TestMachineCRUDReportAndScope(t *testing.T) {
	e := newEnv(t)
	token := e.login("admin", "admin-pass")
	var jobs []api.Job
	if code := e.do("GET", "/jobs", token, nil, &jobs); code != 200 {
		t.Fatal(code)
	}
	input := api.MachineInput{FactoryId: e.plant, JobId: jobs[0].Id, Code: "A01", Name: "测试机台", CapacityQty: 10, CapacityArea: 20}
	var machine api.Machine
	if code := e.do("POST", "/machines", token, input, &machine); code != 201 {
		t.Fatalf("create: %d", code)
	}
	if code := e.do("POST", "/machines", token, input, nil); code != 409 {
		t.Fatalf("duplicate: %d", code)
	}
	input.FactoryId = e.outA
	if code := e.do("POST", "/machines", token, input, nil); code != 400 {
		t.Fatalf("outsource: %d", code)
	}
	input.FactoryId = e.plant
	var list []api.Machine
	if code := e.do("GET", "/machines", token, nil, &list); code != 200 || len(list) != 1 {
		t.Fatalf("list: %d %+v", code, list)
	}
	today := time.Now().Format("2006-01-02")
	var report api.MachineOutputReport
	if code := e.do("GET", fmt.Sprintf("/reports/machine-output?from=%s&to=%s", today, today), token, nil, &report); code != 200 {
		t.Fatalf("report: %d", code)
	}
	if report.Total != 1 || report.Items[0].Qty != 0 || report.Items[0].Days != 1 || *report.Items[0].CapacityQty != 10 {
		t.Fatalf("idle: %+v", report)
	}
	if code := e.do("GET", "/reports/machine-output?from=2026-11-03&to=2026-11-02", token, nil, nil); code != 400 {
		t.Fatalf("dates: %d", code)
	}
	e.outsourceUser("restricted", e.outA)
	restricted := e.login("restricted", "pw-restricted")
	if code := e.do("GET", "/machines", restricted, nil, &list); code != 200 || len(list) != 0 {
		t.Fatalf("scope: %d %+v", code, list)
	}
	if code := e.do("GET", fmt.Sprintf("/machines?factoryId=%d", e.plant), restricted, nil, nil); code != 400 {
		t.Fatalf("scope factory: %d", code)
	}
	input.FactoryId = e.outA
	if code := e.do("PUT", fmt.Sprintf("/machines/%d", machine.Id), restricted, input, nil); code != 400 {
		t.Fatalf("scope update: %d", code)
	}
}

func TestMachineOutputScans(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	_, _, contract := e.batchContract(admin, "machine-output")
	var b api.Batch
	if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: "MACHINE-LOT"}, &b); code != 201 {
		t.Fatal(code)
	}
	var items []api.BatchItem
	if code := e.do("PUT", fmt.Sprintf("/batches/%d/items", b.Id), admin, []api.BatchItemInput{{PieceNo: "A", Qty: 2, PieceArea: floatPtr(1.5)}}, &items); code != 200 {
		t.Fatal(code)
	}
	if code := e.do("POST", fmt.Sprintf("/batches/%d/labels/generate", b.Id), admin, nil, nil); code != 200 {
		t.Fatal(code)
	}
	var labels api.LabelPage
	if code := e.do("GET", fmt.Sprintf("/labels?batchId=%d", b.Id), admin, nil, &labels); code != 200 {
		t.Fatal(code)
	}
	var jobs []api.Job
	e.do("GET", "/jobs", admin, nil, &jobs)
	machineInput := api.MachineInput{FactoryId: e.plant, JobId: jobs[0].Id, Code: "M-OUTPUT", Name: "产线", CapacityQty: 1, CapacityArea: 2}
	var machine api.Machine
	if code := e.do("POST", "/machines", admin, machineInput, &machine); code != 201 {
		t.Fatal(code)
	}
	var result api.ScanResult
	for _, label := range labels.Items {
		req := scanReq(label.BarCode, "10")
		req.MachineId = &machine.Id
		if code := e.do("POST", "/scans", admin, req, &result); code != 201 {
			t.Fatal(code)
		}
	}
	today := time.Now().Format("2006-01-02")
	var report api.MachineOutputReport
	if code := e.do("GET", fmt.Sprintf("/reports/machine-output?from=%s&to=%s", today, today), admin, nil, &report); code != 200 {
		t.Fatal(code)
	}
	if report.Total != 1 || report.Summary.Qty != 2 || report.Items[0].PieceArea != 3 || *report.Items[0].QtyLoad != 200 || *report.Items[0].AreaLoad != 150 {
		t.Fatalf("summary %+v", report)
	}
	if code := e.do("GET", fmt.Sprintf("/reports/machine-output?mode=detail&from=%s&to=%s", today, today), admin, nil, &report); code != 200 {
		t.Fatal(code)
	}
	if report.Total != 1 || report.Items[0].LotNo == nil || *report.Items[0].LotNo != "MACHINE-LOT" || report.Items[0].CapacityQty != nil {
		t.Fatalf("detail %+v", report)
	}
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if code := e.do("GET", fmt.Sprintf("/reports/machine-output?from=%s&to=%s", yesterday, today), admin, nil, &report); code != 200 {
		t.Fatal(code)
	}
	if report.Items[0].Days != 2 || *report.Items[0].CapacityQty != 2 || *report.Items[0].QtyLoad != 100 || *report.Items[0].AreaLoad != 75 {
		t.Fatalf("two days: %+v", report)
	}
	machineInput.JobId = jobs[1].Id
	var another api.Machine
	if code := e.do("POST", "/machines", admin, machineInput, &another); code != 409 { // code must be unique within factory
		t.Fatalf("duplicate job: %d", code)
	}
	machineInput.Code = "M-IDLE"
	if code := e.do("POST", "/machines", admin, machineInput, &another); code != 201 {
		t.Fatal(code)
	}
	otherFactory := must(e.st.CreateFactory(context.Background(), "OTHER", "其他自有厂", "own"))
	crossInput := machineInput
	crossInput.FactoryId = otherFactory.ID
	crossInput.Code = "M-CROSS"
	crossInput.IsStop = false
	var cross api.Machine
	if code := e.do("POST", "/machines", admin, crossInput, &cross); code != 201 {
		t.Fatalf("other factory: %d", code)
	}
	req := scanReq(labels.Items[0].BarCode, "15")
	req.MachineId = &cross.Id
	if code := e.do("POST", "/scans", admin, req, nil); code != 400 {
		t.Fatalf("cross factory: %d", code)
	}
	req.MachineId = &machine.Id
	if code := e.do("POST", "/scans", admin, req, nil); code != 400 {
		t.Fatalf("job mismatch: %d", code)
	}
	machineInput.IsStop = true
	if code := e.do("PUT", fmt.Sprintf("/machines/%d", another.Id), admin, machineInput, nil); code != 200 {
		t.Fatal(code)
	}
	req.MachineId = &another.Id
	if code := e.do("POST", "/scans", admin, req, nil); code != 400 {
		t.Fatalf("stopped: %d", code)
	}
}
