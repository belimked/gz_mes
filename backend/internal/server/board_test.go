package server_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
)

func TestDailyBoards(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	_, _, contract := e.batchContract(admin, "board")
	var batch api.Batch
	if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: "BOARD-1"}, &batch); code != 201 {
		t.Fatalf("batch: %d", code)
	}
	var items []api.BatchItem
	if code := e.do("PUT", fmt.Sprintf("/batches/%d/items", batch.Id), admin, []api.BatchItemInput{{PieceNo: "A", Qty: 2, PieceArea: floatPtr(2)}}, &items); code != 200 {
		t.Fatalf("items: %d", code)
	}
	var generated api.LabelGenerateResult
	if code := e.do("POST", fmt.Sprintf("/batches/%d/labels/generate", batch.Id), admin, nil, &generated); code != 200 {
		t.Fatalf("labels: %d", code)
	}
	ctx := context.Background()
	var userID, job20, job40, labelID int64
	for _, v := range []struct {
		q   string
		dst *int64
	}{
		{`SELECT id FROM app_user WHERE username='admin'`, &userID}, {`SELECT id FROM job WHERE code='20'`, &job20}, {`SELECT id FROM job WHERE code='40'`, &job40}, {fmt.Sprintf(`SELECT id FROM label WHERE batch_id=%d ORDER BY id LIMIT 1`, batch.Id), &labelID},
	} {
		if err := pool.QueryRow(ctx, v.q).Scan(v.dst); err != nil {
			t.Fatal(err)
		}
	}
	for _, kind := range []string{"spray_daily", "sheet_daily"} {
		for _, status := range []string{"audited", "draft", "stopped"} {
			audit, stop := "audited", "normal"
			if status == "draft" {
				audit = "draft"
			}
			if status == "stopped" {
				stop = "stopped"
			}
			if _, err := pool.Exec(ctx, `INSERT INTO plan(kind,factory_id,batch_id,start_date,end_date,job_id,paint_line,workshop,urgency,audit_state,stop_state,created_by) VALUES ($1,$2,$3,current_date-1,current_date+1,$4,'1号线','一车间','urgent',$5,$6,$7)`, kind, e.plant, batch.Id, job20, audit, stop, userID); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, job := range []int64{job20, job40} {
		if _, err := pool.Exec(ctx, `INSERT INTO scan_record(label_id,batch_id,job_id,scanned_by,scanned_at) VALUES ($1,$2,$3,$4,current_date-1)`, labelID, batch.Id, job, userID); err != nil {
			t.Fatal(err)
		}
	}
	today := time.Now().Format("2006-01-02")
	var spray api.SprayDailyBoard
	if code := e.do("GET", "/boards/spray-daily?date="+today, admin, nil, &spray); code != 200 {
		t.Fatalf("spray: %d", code)
	}
	if len(spray.Rows) != 1 || spray.Summary.Count != 1 || spray.Rows[0].SheetArea != 2 || spray.Rows[0].SprayedArea != 2 || spray.Rows[0].DaySprayedArea != 0 || spray.Rows[0].State != "in_progress" {
		t.Fatalf("spray: %+v", spray)
	}
	var sheet api.SheetDailyBoard
	if code := e.do("GET", "/boards/sheet-daily?date="+today, admin, nil, &sheet); code != 200 {
		t.Fatalf("sheet: %d", code)
	}
	if len(sheet.Rows) != 1 || sheet.Rows[0].DayPlanArea != 1.33 || sheet.Rows[0].DoneQty != 1 || sheet.Rows[0].DayDoneQty != 0 || sheet.Rows[0].State != "in_progress" {
		t.Fatalf("sheet: %+v", sheet)
	}
	for _, job := range []int64{job20, job40} {
		if _, err := pool.Exec(ctx, `UPDATE scan_record SET scanned_at=current_date WHERE batch_id=$1 AND job_id=$2`, batch.Id, job); err != nil {
			t.Fatal(err)
		}
	}
	if code := e.do("GET", "/boards/spray-daily", admin, nil, &spray); code != 200 || spray.Rows[0].DaySprayedArea != 2 {
		t.Fatalf("day: %d %+v", code, spray)
	}
	if code := e.do("GET", "/boards/sheet-daily", admin, nil, &sheet); code != 200 || sheet.Rows[0].DayDoneArea != 2 {
		t.Fatalf("day: %d %+v", code, sheet)
	}
	// A normal-priority plan sorts behind the urgent plan even on an earlier line.
	if _, err := pool.Exec(ctx, `INSERT INTO plan(kind,factory_id,batch_id,start_date,end_date,paint_line,urgency,audit_state,created_by) VALUES ('spray_daily',$1,$2,current_date,current_date,'0号线','normal','audited',$3)`, e.plant, batch.Id, userID); err != nil {
		t.Fatal(err)
	}
	if code := e.do("GET", "/boards/spray-daily", admin, nil, &spray); code != 200 || len(spray.Rows) != 2 || spray.Rows[0].Urgency != "urgent" || spray.Rows[1].Urgency != "normal" || spray.Summary.Count != 2 || spray.Summary.PlannedArea != 8 {
		t.Fatalf("sort / summary: %d %+v", code, spray)
	}
	e.outsourceUser("board-scope", e.outA)
	scoped := e.login("board-scope", "pw-board-scope")
	if code := e.do("GET", "/boards/spray-daily", scoped, nil, &spray); code != 200 || len(spray.Rows) != 0 {
		t.Fatalf("scope: %d %+v", code, spray)
	}
	for _, path := range []string{"/boards/spray-daily?date=invalid", "/boards/sheet-daily?date=2026-02-30"} {
		if code := e.do("GET", path, admin, nil, nil); code != 400 {
			t.Fatalf("bad date %s: %d", path, code)
		}
	}
	if code := e.do("GET", "/boards/sheet-daily?date=2020-01-01", admin, nil, &sheet); code != 200 || len(sheet.Rows) != 0 {
		t.Fatalf("date: %d %+v", code, sheet)
	}
	// A plan ending today is due only while its target is incomplete.
	if _, err := pool.Exec(ctx, `UPDATE plan SET end_date=current_date WHERE batch_id=$1`, batch.Id); err != nil {
		t.Fatal(err)
	}
	if code := e.do("GET", "/boards/sheet-daily", admin, nil, &sheet); code != 200 || !sheet.Rows[0].DueToday {
		t.Fatalf("due today: %d %+v", code, sheet)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM scan_record WHERE batch_id=$1 AND job_id=$2`, batch.Id, job40); err != nil {
		t.Fatal(err)
	}
	if code := e.do("GET", "/boards/spray-daily?paintLine=1号线", admin, nil, &spray); code != 200 || spray.Rows[0].State != "ready" {
		t.Fatalf("ready: %d %+v", code, spray)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM scan_record WHERE batch_id=$1 AND job_id=$2`, batch.Id, job20); err != nil {
		t.Fatal(err)
	}
	if code := e.do("GET", "/boards/spray-daily?paintLine=1号线", admin, nil, &spray); code != 200 || spray.Rows[0].State != "waiting_sheet" {
		t.Fatalf("waiting: %d %+v", code, spray)
	}
	if code := e.do("GET", "/boards/sheet-daily", admin, nil, &sheet); code != 200 || sheet.Rows[0].State != "not_started" {
		t.Fatalf("not started: %d %+v", code, sheet)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO scan_record(label_id,batch_id,job_id,scanned_by) SELECT id,$1,$2,$3 FROM label WHERE batch_id=$1 AND NOT cancelled`, batch.Id, job40, userID); err != nil {
		t.Fatal(err)
	}
	if code := e.do("GET", "/boards/spray-daily?paintLine=1号线", admin, nil, &spray); code != 200 || spray.Rows[0].State != "done" || spray.Rows[0].DueToday {
		t.Fatalf("done: %d %+v", code, spray)
	}
}
