package server_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
)

func TestWorkshopOutput(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	_, _, contract := e.batchContract(admin, "workshop-report")
	var batch api.Batch
	if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: "WS-1"}, &batch); code != 201 {
		t.Fatalf("create: %d", code)
	}
	var items []api.BatchItem
	if code := e.do("PUT", fmt.Sprintf("/batches/%d/items", batch.Id), admin, []api.BatchItemInput{{PieceNo: "A", Qty: 1, PieceArea: floatPtr(2)}}, &items); code != 200 {
		t.Fatalf("items: %d", code)
	}
	var generated api.LabelGenerateResult
	if code := e.do("POST", fmt.Sprintf("/batches/%d/labels/generate", batch.Id), admin, nil, &generated); code != 200 {
		t.Fatalf("labels: %d", code)
	}
	ctx := context.Background()
	var labelID, userID, job20, job40 int64
	for _, query := range []struct {
		sql string
		dst *int64
	}{
		{fmt.Sprintf(`SELECT id FROM label WHERE batch_id=%d`, batch.Id), &labelID},
		{`SELECT id FROM app_user WHERE username='admin'`, &userID},
		{`SELECT id FROM job WHERE code='20'`, &job20},
		{`SELECT id FROM job WHERE code='40'`, &job40},
	} {
		if err := pool.QueryRow(ctx, query.sql).Scan(query.dst); err != nil {
			t.Fatal(err)
		}
	}
	// Exercise every stage, first-audit deduplication, and the unassigned workshop bucket.
	for i, sql := range []string{
		`UPDATE batch SET workshop='',order_area=0 WHERE id=$1`,
		`INSERT INTO plan(kind,factory_id,batch_id,start_date,end_date,audit_state,audited_at,created_by) VALUES ('sheet_week',$2,$1,current_date,current_date,'audited',now(),$3)`,
		`INSERT INTO plan(kind,factory_id,batch_id,start_date,end_date,audit_state,audited_at,created_by) VALUES ('sheet_week',$2,$1,current_date,current_date,'audited',now()+interval '1 hour',$3)`,
		`INSERT INTO scan_record(label_id,batch_id,job_id,scanned_by) VALUES ($1,$2,$3,$4),($1,$2,$5,$4)`,
		`INSERT INTO delivery_note(note_no,factory_id,batch_id,customer_id,qty,area,delivered_at,created_by) SELECT 'WS-NOTE',$2,$1,c.customer_id,3,6,current_date,$3 FROM batch b JOIN contract c ON c.id=b.contract_id WHERE b.id=$1`,
	} {
		args := []any{batch.Id, e.plant, userID, labelID, job20, job40}
		n := []int{1, 3, 3, 5, 3}[i]
		if i == 3 {
			args = []any{labelID, batch.Id, job20, userID, job40}
		}
		if _, err := pool.Exec(ctx, sql, args[:n]...); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO plan(kind,factory_id,batch_id,start_date,end_date,audit_state,stop_state,audited_at,created_by) VALUES ('sheet_week',$1,$2,current_date,current_date,'audited','stopped',now()-interval '1 day',$3)`, e.plant, batch.Id, userID); err != nil {
		t.Fatal(err)
	}
	today := time.Now().Format("2006-01-02")
	var r api.WorkshopOutputReport
	if code := e.do("GET", "/reports/workshop-output?granularity=day&from="+today+"&to="+today, admin, nil, &r); code != 200 {
		t.Fatalf("day: %d", code)
	}
	if len(r.Periods) != 1 || r.Total.BatchCount != 1 || r.Total.BatchArea != 2 || r.Total.ScheduledCount != 1 || r.Total.ScheduledArea != 2 || r.Total.SheetQty != 1 || r.Total.SheetArea != 2 || r.Total.SheetOwnArea != 2 || r.Total.SprayQty != 1 || r.Total.SprayArea != 2 || r.Total.DeliveredQty != 3 || r.Total.DeliveredArea != 6 || len(r.Workshops) != 1 || r.Workshops[0] != "未分配" || r.Total.WorkshopSheetArea[0] != 2 || r.Comparison != nil {
		t.Fatalf("day: %+v", r)
	}
	e.outsourceUser("ws-scope", e.outA)
	scoped := e.login("ws-scope", "pw-ws-scope")
	var isolated api.WorkshopOutputReport
	if code := e.do("GET", "/reports/workshop-output?granularity=day&from="+today+"&to="+today, scoped, nil, &isolated); code != 200 || isolated.Total.BatchCount != 0 || isolated.Total.SheetQty != 0 || isolated.Total.DeliveredQty != 0 {
		t.Fatalf("scope: %d %+v", code, isolated)
	}
	// The same scan switches from own / unassigned to outsource / named without changing its event date.
	if _, err := pool.Exec(ctx, `UPDATE batch SET workshop='钣金一车间',outsource_factory_id=$1 WHERE id=$2`, e.outA, batch.Id); err != nil {
		t.Fatal(err)
	}
	if code := e.do("GET", "/reports/workshop-output?granularity=day&from="+today+"&to="+today+"&workshop=钣金一车间", admin, nil, &r); code != 200 || r.Total.SheetOutsourceArea != 2 || r.Total.SheetOwnArea != 0 || len(r.Workshops) != 1 || r.Workshops[0] != "钣金一车间" {
		t.Fatalf("outsource / workshop: %d %+v", code, r)
	}
	if _, err := pool.Exec(ctx, `UPDATE scan_record SET scanned_at=date_trunc('month',now())-interval '1 day' WHERE job_id=$1 AND batch_id=$2`, job20, batch.Id); err != nil {
		t.Fatal(err)
	}
	month := time.Now().Format("2006-01")
	if code := e.do("GET", fmt.Sprintf("/reports/workshop-output?granularity=month&from=%s-01&to=%s-01", month, month), admin, nil, &r); code != 200 || len(r.Periods) != 1 || r.Comparison == nil || len(*r.Comparison) != 2 || (*r.Comparison)[0].PrevArea != 2 || (*r.Comparison)[0].CurrentArea != 0 || (*r.Comparison)[0].ChainRate == nil || *(*r.Comparison)[0].ChainRate != -100 {
		t.Fatalf("month: %d %+v", code, r)
	}
	if code := e.do("GET", "/reports/workshop-output?granularity=day&from=2020-01-01&to=2020-01-02", admin, nil, &r); code != 200 || len(r.Periods) != 2 || r.Total.BatchCount != 0 {
		t.Fatalf("empty: %d %+v", code, r)
	}
	for _, query := range []string{"granularity=day&from=2026-02-01&to=2026-01-01", "granularity=day&from=2020-01-01&to=2022-01-01", "granularity=month&from=2020-01-01&to=2024-01-01"} {
		var er api.Error
		if code := e.do("GET", "/reports/workshop-output?"+query, admin, nil, &er); code != 400 {
			t.Fatalf("invalid %s: %d", query, code)
		}
	}
}
