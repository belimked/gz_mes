package server_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/oapi-codegen/runtime/types"
)

func planDate(y int, m time.Month, d int) types.Date {
	return types.Date{Time: time.Date(y, m, d, 0, 0, 0, 0, time.UTC)}
}
func strPtr(s string) *string { return &s }

func TestPlanFlow(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	_, _, contract := e.batchContract(admin, "plan")
	var b api.Batch
	if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: "PLAN-1", SprayingType: strPtr("氟碳")}, &b); code != 201 {
		t.Fatalf("batch: %d", code)
	}
	path := fmt.Sprintf("/batches/%d", b.Id)
	var items []api.BatchItem
	if code := e.do("PUT", path+"/items", admin, []api.BatchItemInput{{PieceNo: "A", Qty: 2, PieceArea: floatPtr(1.5)}, {PieceNo: "B", Qty: 1, PieceArea: floatPtr(2)}}, &items); code != 200 {
		t.Fatalf("items: %d", code)
	}
	var jobs []api.Job
	e.do("GET", "/jobs", admin, nil, &jobs)
	jobID := map[string]int64{}
	for _, j := range jobs {
		jobID[j.Code] = j.Id
	}

	// Past window so the unfinished plans are overdue.
	start, end := planDate(2026, 1, 5), planDate(2026, 1, 11)
	var er api.Error
	bad := []api.PlanCreate{
		{Kind: api.SheetDaily, BatchId: b.Id, StartDate: start, EndDate: end},
		{Kind: api.SprayDaily, BatchId: b.Id, StartDate: start, EndDate: end, Shift: strPtr("早班")},
		{Kind: api.SheetWeek, BatchId: b.Id, StartDate: end, EndDate: start},
	}
	for i, in := range bad {
		if code := e.do("POST", "/plans", admin, in, &er); code != 400 || er.Code != "invalid_request" {
			t.Fatalf("bad %d: %d %+v", i, code, er)
		}
	}
	if code := e.do("POST", "/plans", admin, api.PlanCreate{Kind: api.SheetWeek, BatchId: 999999, StartDate: start, EndDate: end}, &er); code != 404 || er.Code != "not_found" {
		t.Fatalf("missing batch: %d %+v", code, er)
	}
	var week, daily, spray api.Plan
	cut := jobID["10"]
	if code := e.do("POST", "/plans", admin, api.PlanCreate{Kind: api.SheetWeek, BatchId: b.Id, StartDate: start, EndDate: end, Workshop: strPtr("一车间")}, &week); code != 201 || week.TargetJobName != "钣金下线" || week.LabelQty != 0 || week.Progress != 0 || !week.Overdue || week.AuditState != "draft" || week.StopState != "normal" {
		t.Fatalf("week: %d %+v", code, week)
	}
	if code := e.do("POST", "/plans", admin, api.PlanCreate{Kind: api.SheetDaily, BatchId: b.Id, StartDate: start, EndDate: start, JobId: &cut}, &daily); code != 201 || daily.TargetJobId != cut || daily.JobName == nil || *daily.JobName != "开料" {
		t.Fatalf("daily: %d %+v", code, daily)
	}
	if code := e.do("POST", "/plans", admin, api.PlanCreate{Kind: api.SprayDaily, BatchId: b.Id, StartDate: start, EndDate: start, PaintLine: strPtr("1号线"), Shift: strPtr("早班")}, &spray); code != 201 || spray.TargetJobName != "下挂冷却" || spray.SprayType == nil || *spray.SprayType != "氟碳" {
		t.Fatalf("spray: %d %+v", code, spray)
	}

	var page api.PlanPage
	for _, q := range []string{"kind=sheet_week", "kind=sheet_week&from=2026-01-10&to=2026-01-20", "kind=sheet_week&keyword=PLAN-1", "kind=sheet_week&workshop=%E4%B8%80%E8%BD%A6%E9%97%B4", "kind=sheet_week&auditState=draft&stopState=normal"} {
		if code := e.do("GET", "/plans?"+q, admin, nil, &page); code != 200 || page.Total != 1 || len(page.Items) != 1 || page.Items[0].Id != week.Id || page.Summary.Plans != 1 || page.Summary.Qty != 3 || page.Summary.Area != 5 || page.Summary.Overdue != 1 {
			t.Fatalf("filter %s: %d %+v", q, code, page)
		}
	}
	for _, q := range []string{"kind=sheet_week&from=2026-01-12", "kind=sheet_week&to=2026-01-04", "kind=sheet_week&keyword=nope", "kind=sheet_week&completed=true"} {
		if code := e.do("GET", "/plans?"+q, admin, nil, &page); code != 200 || page.Total != 0 || len(page.Items) != 0 || page.Summary.Qty != 0 {
			t.Fatalf("empty filter %s: %d %+v", q, code, page)
		}
	}
	if code := e.do("GET", fmt.Sprintf("/plans?kind=sheet_daily&jobId=%d", cut), admin, nil, &page); code != 200 || page.Total != 1 || page.Items[0].Id != daily.Id {
		t.Fatalf("job filter: %d %+v", code, page)
	}
	if code := e.do("GET", "/plans?kind=spray_daily&paintLine=1%E5%8F%B7%E7%BA%BF", admin, nil, &page); code != 200 || page.Total != 1 || page.Items[0].Id != spray.Id {
		t.Fatalf("line filter: %d %+v", code, page)
	}
	if code := e.do("GET", "/plans?kind=sheet_week&from=2026-02-01&to=2026-01-01", admin, nil, &er); code != 400 {
		t.Fatalf("bad range: %d", code)
	}

	// Labels + scans through 钣金下线 "20" complete the weekly plan.
	var gen api.LabelGenerateResult
	if code := e.do("POST", path+"/labels/generate", admin, nil, &gen); code != 200 || gen.Total != 3 {
		t.Fatalf("generate: %d %+v", code, gen)
	}
	var labels api.LabelPage
	e.do("GET", fmt.Sprintf("/labels?batchId=%d", b.Id), admin, nil, &labels)
	for _, l := range labels.Items {
		for _, job := range []string{"10", "15", "20"} {
			var r api.ScanResult
			if code := e.do("POST", "/scans", admin, scanReq(l.BarCode, job), &r); code != 201 {
				t.Fatalf("scan %s: %d", job, code)
			}
		}
	}
	wp := fmt.Sprintf("/plans/%d", week.Id)
	if code := e.do("GET", wp, admin, nil, &week); code != 200 || week.LabelQty != 3 || week.DoneQty != 3 || week.DoneArea != 5 || week.Progress != 100 || week.Overdue || week.LastScanAt == nil {
		t.Fatalf("week progress: %d %+v", code, week)
	}
	if code := e.do("GET", "/plans?kind=sheet_week&completed=true", admin, nil, &page); code != 200 || page.Total != 1 || page.Summary.DoneQty != 3 || page.Summary.DoneArea != 5 || page.Summary.Overdue != 0 {
		t.Fatalf("completed: %d %+v", code, page)
	}
	if code := e.do("GET", "/plans?kind=spray_daily&completed=false", admin, nil, &page); code != 200 || page.Total != 1 || page.Items[0].Progress != 0 || !page.Items[0].Overdue {
		t.Fatalf("spray not done: %d %+v", code, page)
	}

	// Edit → audit → locked → unaudit → delete.
	upd := api.PlanUpdate{StartDate: start, EndDate: planDate(2026, 1, 12), Remark: strPtr("改期"), Workshop: strPtr("二车间")}
	if code := e.do("PUT", wp, admin, upd, &week); code != 200 || week.Remark != "改期" || week.Workshop != "二车间" || week.EndDate.Day() != 12 {
		t.Fatalf("update: %d %+v", code, week)
	}
	upd.EndDate = planDate(2026, 1, 1)
	if code := e.do("PUT", wp, admin, upd, &er); code != 400 || er.Code != "invalid_request" {
		t.Fatalf("update bad dates: %d %+v", code, er)
	}
	if code := e.do("POST", wp+"/audit", admin, map[string]string{"action": "audit"}, &week); code != 200 || week.AuditState != "audited" || week.AuditedBy == nil || week.AuditedAt == nil {
		t.Fatalf("audit: %d %+v", code, week)
	}
	if code := e.do("POST", wp+"/audit", admin, map[string]string{"action": "audit"}, &er); code != 409 || er.Code != "already_audited" {
		t.Fatalf("audit twice: %d %+v", code, er)
	}
	upd.EndDate = end
	if code := e.do("PUT", wp, admin, upd, &er); code != 409 || er.Code != "already_audited" {
		t.Fatalf("update audited: %d %+v", code, er)
	}
	if code := e.do("DELETE", wp, admin, nil, &er); code != 409 || er.Code != "already_audited" {
		t.Fatalf("delete audited: %d %+v", code, er)
	}
	var draft api.Plan // fresh value: nil auditedBy is omitted from JSON
	if code := e.do("POST", wp+"/audit", admin, map[string]string{"action": "unaudit"}, &draft); code != 200 || draft.AuditState != "draft" || draft.AuditedBy != nil || draft.AuditedAt != nil {
		t.Fatalf("unaudit: %d %+v", code, draft)
	}
	if code := e.do("POST", wp+"/audit", admin, map[string]string{"action": "unaudit"}, &er); code != 409 || er.Code != "not_audited" {
		t.Fatalf("unaudit twice: %d %+v", code, er)
	}
	if code := e.do("DELETE", wp, admin, nil, nil); code != 204 {
		t.Fatalf("delete: %d", code)
	}
	if code := e.do("GET", wp, admin, nil, &er); code != 404 || er.Code != "not_found" {
		t.Fatalf("deleted: %d %+v", code, er)
	}

	// Stop-state machine on the daily plan.
	dp := fmt.Sprintf("/plans/%d/stop-state", daily.Id)
	steps := []struct {
		action, want string
		code         int
	}{{"resume", "", 409}, {"pause", "paused", 200}, {"pause", "", 409}, {"resume", "normal", 200}, {"stop", "stopped", 200}, {"resume", "", 409}, {"stop", "", 409}}
	for _, s := range steps {
		var p api.Plan
		body := map[string]string{"action": s.action, "reason": "缺料"}
		if s.code == 200 {
			if code := e.do("POST", dp, admin, body, &p); code != 200 || string(p.StopState) != s.want {
				t.Fatalf("%s: %d %+v", s.action, code, p)
			}
			continue
		}
		if code := e.do("POST", dp, admin, body, &er); code != s.code || er.Code != "invalid_transition" {
			t.Fatalf("%s: %d %+v", s.action, code, er)
		}
	}
	if code := e.do("GET", fmt.Sprintf("/plans/%d", daily.Id), admin, nil, &daily); code != 200 || daily.StopReason == nil || *daily.StopReason != "缺料" || daily.Overdue {
		t.Fatalf("stopped daily: %d %+v", code, daily)
	}

	// Duplicates: overlapping un-stopped plan of the same kind+batch(+job).
	if code := e.do("POST", "/plans", admin, api.PlanCreate{Kind: api.SprayDaily, BatchId: b.Id, StartDate: start, EndDate: end, PaintLine: strPtr("2号线"), Shift: strPtr("晚班")}, &er); code != 409 || er.Code != "duplicate_plan" {
		t.Fatalf("duplicate spray: %d %+v", code, er)
	}
	var again api.Plan
	if code := e.do("POST", "/plans", admin, api.PlanCreate{Kind: api.SheetDaily, BatchId: b.Id, StartDate: start, EndDate: start, JobId: &cut}, &again); code != 201 {
		t.Fatalf("daily after stop: %d %+v", code, again)
	}
	if code := e.do("POST", "/plans", admin, api.PlanCreate{Kind: api.SheetDaily, BatchId: b.Id, StartDate: start, EndDate: end, JobId: &cut}, &er); code != 409 || er.Code != "duplicate_plan" {
		t.Fatalf("duplicate daily: %d %+v", code, er)
	}
	bend := jobID["15"]
	if code := e.do("POST", "/plans", admin, api.PlanCreate{Kind: api.SheetDaily, BatchId: b.Id, StartDate: start, EndDate: end, JobId: &bend}, &again); code != 201 || again.Progress != 100 {
		t.Fatalf("other job: %d %+v", code, again)
	}

	// Outsource accounts see nothing of the plant's plans and cannot create there.
	e.outsourceUser("plan-wa", e.outA)
	wa := e.login("plan-wa", "pw-plan-wa")
	if code := e.do("GET", "/plans?kind=spray_daily", wa, nil, &page); code != 200 || page.Total != 0 || len(page.Items) != 0 {
		t.Fatalf("scope list: %d %+v", code, page)
	}
	if code := e.do("GET", fmt.Sprintf("/plans/%d", spray.Id), wa, nil, &er); code != 404 {
		t.Fatalf("scope get: %d", code)
	}
	if code := e.do("POST", "/plans", wa, api.PlanCreate{Kind: api.SheetWeek, BatchId: b.Id, StartDate: planDate(2026, 3, 1), EndDate: planDate(2026, 3, 2)}, &er); code != 403 || er.Code != "forbidden_factory" {
		t.Fatalf("scope create: %d %+v", code, er)
	}
}
