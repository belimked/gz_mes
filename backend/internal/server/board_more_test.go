package server_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/belimked/gz_mes/backend/internal/api"
)

func TestMoreBoards(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	_, _, contract := e.batchContract(admin, "more-board")
	makeBatch := func(lot string, days int) api.Batch {
		t.Helper()
		var b api.Batch
		if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: lot}, &b); code != 201 {
			t.Fatalf("batch: %d", code)
		}
		var items []api.BatchItem
		if code := e.do("PUT", fmt.Sprintf("/batches/%d/items", b.Id), admin, []api.BatchItemInput{{PieceNo: "A", Qty: 2, PieceArea: floatPtr(2)}}, &items); code != 200 {
			t.Fatalf("items: %d", code)
		}
		if _, err := pool.Exec(context.Background(), `UPDATE batch SET required_date=current_date + $2::int,workshop='一车间' WHERE id=$1`, b.Id, days); err != nil {
			t.Fatal(err)
		}
		return b
	}
	regular := makeBatch("MORE-REG", 3)
	late := makeBatch("MORE-LATE", -1)
	var generated api.LabelGenerateResult
	if code := e.do("POST", fmt.Sprintf("/batches/%d/labels/generate", late.Id), admin, nil, &generated); code != 200 {
		t.Fatalf("labels: %d", code)
	}
	ctx := context.Background()
	var uid, job, label int64
	for _, r := range []struct {
		q    string
		dest *int64
	}{{`SELECT id FROM app_user WHERE username='admin'`, &uid}, {`SELECT id FROM job WHERE code='20'`, &job}, {fmt.Sprintf(`SELECT id FROM label WHERE batch_id=%d LIMIT 1`, late.Id), &label}} {
		if err := pool.QueryRow(ctx, r.q).Scan(r.dest); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO scan_record(label_id,batch_id,job_id,scanned_by,scanned_at) VALUES ($1,$2,$3,$4,current_date)`, label, late.Id, job, uid); err != nil {
		t.Fatal(err)
	}
	var board api.WorkOrderBoard
	if code := e.do("GET", "/boards/work-orders?size=1", admin, nil, &board); code != 200 || board.Total != 2 || len(board.Items) != 1 || board.Items[0].BatchId != late.Id || !board.Items[0].Overdue || board.Items[0].PrintState != "all" {
		t.Fatalf("ordering: %d %+v", code, board)
	}
	for _, j := range board.Items[0].Jobs {
		if j.JobId == job && (j.Qty != 1 || j.Pct != 50) {
			t.Fatalf("job: %+v", j)
		}
	}
	if code := e.do("GET", "/boards/work-orders?status=pending&keyword=MORE-REG", admin, nil, &board); code != 200 || board.Total != 1 || board.Items[0].BatchId != regular.Id || board.Items[0].PrintState != "none" {
		t.Fatalf("filter: %d %+v", code, board)
	}
	var cockpit api.CockpitBoard
	if code := e.do("GET", "/boards/cockpit", admin, nil, &cockpit); code != 200 || cockpit.Kpi.SemiStockQty != 1 || cockpit.Kpi.TodayScanQty != 1 || len(cockpit.WeeklySheet) != 7 || len(cockpit.WeeklySpray) != 7 || len(cockpit.WeeklyShipment) != 7 || len(cockpit.WorkshopBacklog) != 1 || cockpit.WorkshopBacklog[0].NoScanQty != 1 {
		t.Fatalf("cockpit: %d %+v", code, cockpit)
	}
	for _, v := range cockpit.WeeklySheet[:6] {
		if v.Qty != 0 || v.Area != 0 {
			t.Fatalf("weekly zero fill: %+v", cockpit.WeeklySheet)
		}
	}
	if cockpit.WeeklySheet[6].Qty != 1 || cockpit.WeeklySheet[6].Area != 2 {
		t.Fatalf("today weekly output: %+v", cockpit.WeeklySheet)
	}
	e.outsourceUser("more-board-scope", e.outA)
	scoped := e.login("more-board-scope", "pw-more-board-scope")
	if code := e.do("GET", "/boards/work-orders", scoped, nil, &board); code != 200 || board.Total != 0 {
		t.Fatalf("scope board: %d %+v", code, board)
	}
	if code := e.do("GET", "/boards/cockpit", scoped, nil, &cockpit); code != 200 || cockpit.Kpi.TodayScanQty != 0 || cockpit.Kpi.SemiStockQty != 0 || len(cockpit.WeeklySheet) != 7 {
		t.Fatalf("scope cockpit: %d %+v", code, cockpit)
	}
}
