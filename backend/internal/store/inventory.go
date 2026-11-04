package store

import (
	"context"
	"time"

	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

type InventoryFilter struct {
	Kind     string
	From, To *time.Time
	Keyword  string
	BatchID  *int64
	HasStock bool
}
type InventoryRow struct {
	BatchItemID, BatchID              int64
	LotNo, ProjectName, ClientOrderNo string
	OrderDate                         time.Time
	Thickness                         float64
	Color, PieceNo                    string
	Seq                               int
	DrawingNo                         string
	PlanQty                           int
	PieceArea, PlanArea               float64
	InQty, OutQty, StockQty           int
	StockArea                         float64
}
type InventoryTotals struct {
	StockQty      int
	StockArea     float64
	InQty, OutQty int
}

func (s *Store) Inventory(ctx context.Context, sc auth.Scope, f InventoryFilter, page, size int) ([]InventoryRow, int, InventoryTotals, error) {
	inCode, outCode := "20", "40"
	if f.Kind == "finished" {
		inCode, outCode = "40", "60"
	}
	// Aggregation per label prevents a historical outbound scan from being mistaken for current stock.
	query := `WITH counts AS (
 SELECT bi.id AS item_id, count(l.id)::int AS labels,
 count(l.id) FILTER (WHERE si.id IS NOT NULL AND so.id IS NULL)::int AS stock_qty,
 count(si.id) FILTER (WHERE $3::date IS NOT NULL AND $4::date IS NOT NULL AND si.scanned_at::date BETWEEN $3::date AND $4::date)::int AS in_qty,
 count(so.id) FILTER (WHERE $3::date IS NOT NULL AND $4::date IS NOT NULL AND so.scanned_at::date BETWEEN $3::date AND $4::date)::int AS out_qty
 FROM batch_item bi JOIN batch b ON b.id=bi.batch_id
 JOIN label l ON l.batch_item_id=bi.id AND NOT l.cancelled
 LEFT JOIN scan_record si ON si.label_id=l.id AND si.job_id=(SELECT id FROM job WHERE code=$5)
 LEFT JOIN scan_record so ON so.label_id=l.id AND so.job_id=(SELECT id FROM job WHERE code=$6)
 WHERE ($1 OR b.factory_id=ANY($2)) GROUP BY bi.id
 ), matched AS (
 SELECT bi.id AS item_id,b.id AS batch_id,b.lot_no,p.full_name,b.client_order_no,b.created_at::date AS order_date,
 bi.thickness::float8,bi.color,bi.piece_no,bi.seq,bi.drawing_no,bi.qty,
 bi.piece_area::float8,(bi.qty*bi.piece_area)::float8 AS plan_area,
 c.in_qty,c.out_qty,c.stock_qty,(c.stock_qty*bi.piece_area)::float8 AS stock_area
 FROM counts c JOIN batch_item bi ON bi.id=c.item_id JOIN batch b ON b.id=bi.batch_id
 JOIN contract ct ON ct.id=b.contract_id JOIN project p ON p.id=ct.project_id
 WHERE ($7::bigint IS NULL OR b.id=$7) AND ($8::text='' OR b.lot_no ILIKE $8 OR p.full_name ILIKE $8 OR b.client_order_no ILIKE $8 OR bi.piece_no ILIKE $8 OR bi.drawing_no ILIKE $8)
 AND (NOT $9::bool OR c.stock_qty>0 OR c.in_qty>0 OR c.out_qty>0)
 )`
	args := []any{sc.All, sc.FactoryIDs, f.From, f.To, inCode, outCode, f.BatchID, likeKeyword(f.Keyword), f.HasStock}
	var total int
	var sums InventoryTotals
	err := s.db.QueryRow(ctx, query+` SELECT count(*)::int,coalesce(sum(stock_qty),0)::int,coalesce(sum(stock_area),0)::float8,coalesce(sum(in_qty),0)::int,coalesce(sum(out_qty),0)::int FROM matched`, args...).Scan(&total, &sums.StockQty, &sums.StockArea, &sums.InQty, &sums.OutQty)
	if err != nil {
		return nil, 0, sums, err
	}
	rows, err := s.db.Query(ctx, query+` SELECT * FROM matched ORDER BY lot_no,seq,item_id LIMIT $10 OFFSET $11`, append(args, size, (page-1)*size)...)
	if err != nil {
		return nil, 0, sums, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[InventoryRow])
	return items, total, sums, err
}

func likeKeyword(s string) string {
	if s == "" {
		return ""
	}
	return "%" + escapeLike(s) + "%"
}

type ProjectSummaryFilter struct {
	From, To        *time.Time
	Keyword, Status string
}
type ProjectSummaryRow struct {
	ProjectID                              int64
	ProjectCode, ProjectName, CustomerName string
	BatchCount, PlanQty                    int
	PlanArea                               float64
	LabelQty                               int
	Status                                 string
	Cells                                  []JobCell
}
type ProjectSummaryTotals struct {
	BatchCount, PlanQty int
	PlanArea            float64
	LabelQty            int
	Cells               []JobCell
}

func (s *Store) ProjectSummary(ctx context.Context, sc auth.Scope, f ProjectSummaryFilter, page, size int) ([]Job, []ProjectSummaryRow, int, ProjectSummaryTotals, error) {
	jobs, err := s.Jobs(ctx)
	if err != nil {
		return nil, nil, 0, ProjectSummaryTotals{}, err
	}
	// Aggregate batch facts before joining labels/scans so plan quantities are never multiplied.
	base := `WITH selected AS (
 SELECT b.id,b.status,b.item_qty,b.item_area,b.contract_id FROM batch b JOIN contract ct ON ct.id=b.contract_id JOIN project p ON p.id=ct.project_id
 WHERE ($1 OR b.factory_id=ANY($2)) AND ($3::date IS NULL OR b.created_at::date >= $3::date) AND ($4::date IS NULL OR b.created_at::date <= $4::date)
 AND ($5::text='' OR p.full_name ILIKE $5 OR p.code ILIKE $5)
 ), grouped AS (
 SELECT p.id,p.code,p.full_name,string_agg(DISTINCT cu.name,', ' ORDER BY cu.name) AS customer_name,
 count(*)::int AS batch_count,sum(b.item_qty)::int AS plan_qty,sum(b.item_area)::float8 AS plan_area,
 CASE WHEN bool_and(b.status='done') THEN 'done' WHEN bool_or(b.status='in_production') THEN 'in_production' ELSE 'pending' END AS status
 FROM selected b JOIN contract ct ON ct.id=b.contract_id JOIN project p ON p.id=ct.project_id JOIN customer cu ON cu.id=ct.customer_id
 GROUP BY p.id,p.code,p.full_name
 ), matched AS (SELECT * FROM grouped WHERE $6='all' OR ($6='done' AND status='done') OR ($6='not_done' AND status<>'done'))`
	args := []any{sc.All, sc.FactoryIDs, f.From, f.To, likeKeyword(f.Keyword), f.Status}
	var total int
	var totals ProjectSummaryTotals
	err = s.db.QueryRow(ctx, base+` SELECT count(*)::int,coalesce(sum(batch_count),0)::int,coalesce(sum(plan_qty),0)::int,coalesce(sum(plan_area),0)::float8 FROM matched`, args...).Scan(&total, &totals.BatchCount, &totals.PlanQty, &totals.PlanArea)
	if err != nil {
		return nil, nil, 0, totals, err
	}
	rows, err := s.db.Query(ctx, base+` SELECT id,code,full_name,customer_name,batch_count,plan_qty,plan_area,status FROM matched ORDER BY code,id LIMIT $7 OFFSET $8`, append(args, size, (page-1)*size)...)
	if err != nil {
		return nil, nil, 0, totals, err
	}
	type head struct {
		ID                   int64
		Code, Name, Customer string
		Batches, Qty         int
		Area                 float64
		Status               string
	}
	heads, err := pgx.CollectRows(rows, pgx.RowToStructByPos[head])
	if err != nil {
		return nil, nil, 0, totals, err
	}
	items := make([]ProjectSummaryRow, 0, len(heads))
	index := map[int64]int{}
	for _, h := range heads {
		index[h.ID] = len(items)
		items = append(items, ProjectSummaryRow{ProjectID: h.ID, ProjectCode: h.Code, ProjectName: h.Name, CustomerName: h.Customer, BatchCount: h.Batches, PlanQty: h.Qty, PlanArea: h.Area, Status: h.Status, Cells: make([]JobCell, len(jobs))})
	}
	for k, j := range jobs {
		totals.Cells = append(totals.Cells, JobCell{JobID: j.ID})
		for i := range items {
			items[i].Cells[k].JobID = j.ID
		}
	}
	// Counts over all matched projects, independent of pagination.
	rows, err = s.db.Query(ctx, base+` SELECT m.id,l.id,sr.job_id,bi.piece_area::float8 FROM matched m
 JOIN selected b ON b.contract_id IN (SELECT id FROM contract WHERE project_id=m.id)
 JOIN label l ON l.batch_id=b.id AND NOT l.cancelled
 JOIN batch_item bi ON bi.id=l.batch_item_id
 LEFT JOIN scan_record sr ON sr.label_id=l.id ORDER BY m.id,l.id`, args...)
	if err != nil {
		return nil, nil, 0, totals, err
	}
	defer rows.Close()
	jobIndex := map[int64]int{}
	for k, j := range jobs {
		jobIndex[j.ID] = k
	}
	seen := map[int64]bool{}
	pageSeen := map[int64]bool{}
	for rows.Next() {
		var projectID, labelID int64
		var jobID *int64
		var area float64
		if err = rows.Scan(&projectID, &labelID, &jobID, &area); err != nil {
			return nil, nil, 0, totals, err
		}
		if !seen[labelID] {
			totals.LabelQty++
			seen[labelID] = true
		}
		if i, ok := index[projectID]; ok && !pageSeen[labelID] {
			items[i].LabelQty++
			pageSeen[labelID] = true
		}
		if jobID != nil {
			if k, ok := jobIndex[*jobID]; ok {
				totals.Cells[k].Qty++
				totals.Cells[k].Area += area
				if i, ok := index[projectID]; ok {
					items[i].Cells[k].Qty++
					items[i].Cells[k].Area += area
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, 0, totals, err
	}
	for k := range totals.Cells {
		totals.Cells[k].Pct = pct(totals.Cells[k].Qty, totals.LabelQty)
		for i := range items {
			items[i].Cells[k].Pct = pct(items[i].Cells[k].Qty, items[i].LabelQty)
		}
	}
	return jobs, items, total, totals, nil
}
