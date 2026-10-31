package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

// Report rows are computed live from scan_record; nothing here is persisted.

type JobCell struct {
	JobID int64
	Qty   int
	Area  float64
	Pct   int
}

type BatchJobRow struct {
	BatchID                          int64
	LotNo, ProjectName, CustomerName string
	ClientOrderNo                    string
	Thickness                        float64
	Color, Status, Urgency, Workshop string
	OutsourceFactoryName             *string
	RequiredDate                     *time.Time
	ItemQty                          int
	ItemArea                         float64
	LabelQty                         int
	CurrentJobName                   string
	LastScanAt                       *time.Time
	Cells                            []JobCell
}

type BatchJobSummary struct {
	Batches, ItemQty int
	ItemArea         float64
	LabelQty         int
	Cells            []JobCell
}

type BatchJobFilter struct {
	Mode               string // daily | progress
	From, To           *time.Time
	Status, Workshop   string
	OutsourceFactoryID *int64
	JobID              *int64
	Scanned            *bool
	Keyword            string
}

func pct(qty, total int) int {
	if total <= 0 {
		return 0
	}
	return qty * 100 / total
}

// BatchJobReport powers 工程日报 (daily: scans in the window) and 工程进度 (progress: all scans).
func (s *Store) BatchJobReport(ctx context.Context, sc auth.Scope, f BatchJobFilter, page, size int) ([]Job, []BatchJobRow, BatchJobSummary, error) {
	jobs, err := s.Jobs(ctx)
	if err != nil {
		return nil, nil, BatchJobSummary{}, err
	}
	daily := f.Mode == "daily"
	// $3/$4: scan window (daily) or batch created window (progress). Nil bounds are open.
	args := []any{sc.All, sc.FactoryIDs, f.From, f.To}
	scanWindow := ` TRUE`
	if daily {
		scanWindow = ` ($3::date IS NULL OR sr.scanned_at::date >= $3::date) AND ($4::date IS NULL OR sr.scanned_at::date <= $4::date)`
	}
	// counted: scans of active labels, restricted to the window in daily mode.
	from := ` FROM batch b JOIN contract c ON c.id=b.contract_id JOIN customer cu ON cu.id=c.customer_id JOIN project p ON p.id=c.project_id
 LEFT JOIN factory of ON of.id=b.outsource_factory_id
 CROSS JOIN LATERAL (SELECT count(*)::int AS qty FROM label l WHERE l.batch_id=b.id AND NOT l.cancelled) lb
 CROSS JOIN LATERAL (
  SELECT count(*)::int AS qty, max(sr.scanned_at) AS last, max(j.seq) AS seq
  FROM scan_record sr JOIN label l ON l.id=sr.label_id JOIN job j ON j.id=sr.job_id
  WHERE sr.batch_id=b.id AND NOT l.cancelled AND` + scanWindow + `
 ) cs
 LEFT JOIN job cj ON cj.seq=(SELECT max(j.seq) FROM scan_record sr JOIN label l ON l.id=sr.label_id JOIN job j ON j.id=sr.job_id WHERE sr.batch_id=b.id AND NOT l.cancelled)`
	where := ` WHERE ($1 OR b.factory_id=ANY($2))`
	if daily {
		where += ` AND cs.qty>0`
	} else {
		where += ` AND ($3::date IS NULL OR b.created_at::date >= $3::date) AND ($4::date IS NULL OR b.created_at::date <= $4::date)`
	}
	add := func(q string, v any) { args = append(args, v); where += fmt.Sprintf(q, len(args)) }
	if f.Status != "" {
		add(` AND b.status=$%d`, f.Status)
	}
	if f.Workshop != "" {
		add(` AND b.workshop=$%d`, f.Workshop)
	}
	if f.OutsourceFactoryID != nil {
		add(` AND b.outsource_factory_id=$%d`, *f.OutsourceFactoryID)
	}
	if f.JobID != nil {
		add(` AND EXISTS (SELECT 1 FROM scan_record sr JOIN label l ON l.id=sr.label_id WHERE sr.batch_id=b.id AND NOT l.cancelled AND sr.job_id=$%d AND`+scanWindow+`)`, *f.JobID)
	}
	if f.Scanned != nil {
		if *f.Scanned {
			where += ` AND cs.qty>0`
		} else {
			where += ` AND cs.qty=0`
		}
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		add(` AND (b.lot_no ILIKE $%[1]d OR p.full_name ILIKE $%[1]d OR cu.name ILIKE $%[1]d OR b.client_order_no ILIKE $%[1]d)`, "%"+escapeLike(kw)+"%")
	}
	var sum BatchJobSummary
	err = s.db.QueryRow(ctx, `SELECT count(*)::int,coalesce(sum(b.item_qty),0)::int,coalesce(sum(b.item_area),0)::float8,coalesce(sum(lb.qty),0)::int`+from+where, args...).
		Scan(&sum.Batches, &sum.ItemQty, &sum.ItemArea, &sum.LabelQty)
	if err != nil {
		return nil, nil, sum, err
	}
	// Per-job totals over every matching batch (not only the page).
	cellArgs := append(append([]any{}, args...), daily)
	rows, err := s.db.Query(ctx, `WITH matched AS (SELECT b.id`+from+where+`)
 SELECT j.id,count(sr.id)::int,coalesce(sum(bi.piece_area),0)::float8,0 FROM job j
 LEFT JOIN scan_record sr ON sr.job_id=j.id AND sr.batch_id IN (SELECT id FROM matched)
  AND (NOT $`+fmt.Sprint(len(cellArgs))+`::bool OR (($3::date IS NULL OR sr.scanned_at::date >= $3::date) AND ($4::date IS NULL OR sr.scanned_at::date <= $4::date)))
 LEFT JOIN label l ON l.id=sr.label_id AND NOT l.cancelled
 LEFT JOIN batch_item bi ON bi.id=l.batch_item_id
 WHERE sr.id IS NULL OR l.id IS NOT NULL
 GROUP BY j.id ORDER BY j.seq`, cellArgs...)
	if err != nil {
		return nil, nil, sum, err
	}
	sum.Cells, err = pgx.CollectRows(rows, pgx.RowToStructByPos[JobCell])
	if err != nil {
		return nil, nil, sum, err
	}
	for i := range sum.Cells {
		sum.Cells[i].Pct = pct(sum.Cells[i].Qty, sum.LabelQty)
	}
	order := ` ORDER BY b.required_date NULLS LAST,b.id DESC`
	if daily {
		order = ` ORDER BY cs.last DESC,b.id DESC`
	}
	args = append(args, size, (page-1)*size)
	rows, err = s.db.Query(ctx, `SELECT b.id,b.lot_no,p.full_name,cu.name,b.client_order_no,b.thickness::float8,b.color,b.status,b.urgency,b.workshop,of.name,b.required_date,
 b.item_qty,b.item_area::float8,lb.qty,coalesce(cj.name,''),cs.last`+from+where+order+fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, nil, sum, err
	}
	type rowHead struct {
		BatchID                          int64
		LotNo, ProjectName, CustomerName string
		ClientOrderNo                    string
		Thickness                        float64
		Color, Status, Urgency, Workshop string
		OutsourceFactoryName             *string
		RequiredDate                     *time.Time
		ItemQty                          int
		ItemArea                         float64
		LabelQty                         int
		CurrentJobName                   string
		LastScanAt                       *time.Time
	}
	heads, err := pgx.CollectRows(rows, pgx.RowToStructByPos[rowHead])
	if err != nil {
		return nil, nil, sum, err
	}
	out := make([]BatchJobRow, 0, len(heads))
	ids := make([]int64, 0, len(heads))
	index := map[int64]int{}
	for i, h := range heads {
		cells := make([]JobCell, len(jobs))
		for k, j := range jobs {
			cells[k] = JobCell{JobID: j.ID}
		}
		out = append(out, BatchJobRow{BatchID: h.BatchID, LotNo: h.LotNo, ProjectName: h.ProjectName, CustomerName: h.CustomerName, ClientOrderNo: h.ClientOrderNo, Thickness: h.Thickness, Color: h.Color, Status: h.Status, Urgency: h.Urgency, Workshop: h.Workshop, OutsourceFactoryName: h.OutsourceFactoryName, RequiredDate: h.RequiredDate, ItemQty: h.ItemQty, ItemArea: h.ItemArea, LabelQty: h.LabelQty, CurrentJobName: h.CurrentJobName, LastScanAt: h.LastScanAt, Cells: cells})
		ids = append(ids, h.BatchID)
		index[h.BatchID] = i
	}
	if len(ids) == 0 {
		return jobs, out, sum, nil
	}
	jobIndex := map[int64]int{}
	for k, j := range jobs {
		jobIndex[j.ID] = k
	}
	rows, err = s.db.Query(ctx, `SELECT sr.batch_id,sr.job_id,count(*)::int,coalesce(sum(bi.piece_area),0)::float8
 FROM scan_record sr JOIN label l ON l.id=sr.label_id JOIN batch_item bi ON bi.id=l.batch_item_id
 WHERE sr.batch_id=ANY($1) AND NOT l.cancelled
 AND (NOT $4::bool OR (($2::date IS NULL OR sr.scanned_at::date >= $2::date) AND ($3::date IS NULL OR sr.scanned_at::date <= $3::date)))
 GROUP BY sr.batch_id,sr.job_id`, ids, f.From, f.To, daily)
	if err != nil {
		return nil, nil, sum, err
	}
	type cellRow struct {
		BatchID, JobID int64
		Qty            int
		Area           float64
	}
	cells, err := pgx.CollectRows(rows, pgx.RowToStructByPos[cellRow])
	if err != nil {
		return nil, nil, sum, err
	}
	for _, c := range cells {
		r := &out[index[c.BatchID]]
		k, ok := jobIndex[c.JobID]
		if !ok {
			continue
		}
		r.Cells[k].Qty, r.Cells[k].Area = c.Qty, c.Area
		r.Cells[k].Pct = pct(c.Qty, r.LabelQty)
	}
	return jobs, out, sum, nil
}

type StaffOutputRow struct {
	UserID               int64
	StaffNo, DisplayName string
	JobID                int64
	JobName              string
	BatchID              *int64
	LotNo, ProjectName   *string
	Qty                  int
	Area, SprayArea      float64
	WeldLine             float64
	FirstScanAt          time.Time
	LastScanAt           time.Time
}

type StaffOutputSummary struct {
	Rows, Staff, Qty int
	Area, SprayArea  float64
	WeldLine         float64
}

type StaffOutputFilter struct {
	From, To            *time.Time
	JobID               *int64
	Staff, BatchKeyword string
	Detail              bool
}

// StaffOutputReport groups scans by operator × job (× batch when Detail).
func (s *Store) StaffOutputReport(ctx context.Context, sc auth.Scope, f StaffOutputFilter, page, size int) ([]StaffOutputRow, StaffOutputSummary, error) {
	args := []any{sc.All, sc.FactoryIDs, f.From, f.To}
	where := ` WHERE ($1 OR b.factory_id=ANY($2)) AND NOT l.cancelled
 AND ($3::date IS NULL OR sr.scanned_at::date >= $3::date) AND ($4::date IS NULL OR sr.scanned_at::date <= $4::date)`
	add := func(q string, v any) { args = append(args, v); where += fmt.Sprintf(q, len(args)) }
	if f.JobID != nil {
		add(` AND sr.job_id=$%d`, *f.JobID)
	}
	if kw := strings.TrimSpace(f.Staff); kw != "" {
		add(` AND (u.display_name ILIKE $%[1]d OR u.staff_no ILIKE $%[1]d OR u.username ILIKE $%[1]d)`, "%"+escapeLike(kw)+"%")
	}
	if kw := strings.TrimSpace(f.BatchKeyword); kw != "" {
		add(` AND (b.lot_no ILIKE $%[1]d OR p.full_name ILIKE $%[1]d)`, "%"+escapeLike(kw)+"%")
	}
	from := ` FROM scan_record sr JOIN label l ON l.id=sr.label_id JOIN batch_item bi ON bi.id=l.batch_item_id
 JOIN batch b ON b.id=sr.batch_id JOIN contract c ON c.id=b.contract_id JOIN project p ON p.id=c.project_id
 JOIN job j ON j.id=sr.job_id JOIN app_user u ON u.id=sr.scanned_by`
	group := `u.id,u.staff_no,u.display_name,j.id,j.name,j.seq`
	selectBatch := `NULL::bigint,NULL::text,NULL::text`
	orderTail := ``
	if f.Detail {
		group += `,b.id,b.lot_no,p.full_name`
		selectBatch = `b.id,b.lot_no,p.full_name`
		orderTail = `,b.lot_no`
	}
	var sum StaffOutputSummary
	err := s.db.QueryRow(ctx, `SELECT count(*)::int,count(DISTINCT u.id)::int,coalesce(sum(qty),0)::int,coalesce(sum(area),0)::float8,coalesce(sum(spray),0)::float8,coalesce(sum(weld),0)::float8
 FROM (SELECT u.id,count(*) qty,sum(bi.piece_area) area,sum(bi.spraying_area) spray,sum(bi.weld_line) weld`+from+where+` GROUP BY `+group+`) g JOIN app_user u ON u.id=g.id`, args...).
		Scan(&sum.Rows, &sum.Staff, &sum.Qty, &sum.Area, &sum.SprayArea, &sum.WeldLine)
	if err != nil {
		return nil, sum, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, `SELECT u.id,u.staff_no,u.display_name,j.id,j.name,`+selectBatch+`,count(*)::int,
 coalesce(sum(bi.piece_area),0)::float8,coalesce(sum(bi.spraying_area),0)::float8,coalesce(sum(bi.weld_line),0)::float8,min(sr.scanned_at),max(sr.scanned_at)`+from+where+
		` GROUP BY `+group+` ORDER BY count(*) DESC,u.staff_no,u.id,j.seq`+orderTail+fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, sum, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[StaffOutputRow])
	return items, sum, err
}
