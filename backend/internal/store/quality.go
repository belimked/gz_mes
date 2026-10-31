package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/belimked/gz_mes/backend/internal/auth"
)

// Quality data lives in qc_record (one row per inspection). A pass also writes the flow scan_record;
// a fail does not. "Reinspected" means a later pass exists for the same label at the same QC job.

type QcRecord struct {
	ID, LabelID          int64
	BarCode              string
	BatchID              int64
	LotNo, ProjectName   string
	OutsourceFactoryName *string
	PieceNo              string
	PieceIndex           int
	PieceArea            float64
	Color                string
	JobID                int64
	JobName, Result      string
	DefectJobID          *int64
	DefectJobName        *string
	DefectCategory       string
	DefectReason         string
	InspectedBy, StaffNo string
	InspectedAt          time.Time
	Device               string
	Reinspected          bool
	ReinspectedAt        *time.Time
}

const qcFrom = ` FROM qc_record q JOIN label l ON l.id=q.label_id JOIN batch b ON b.id=q.batch_id JOIN contract c ON c.id=b.contract_id JOIN project p ON p.id=c.project_id
 JOIN batch_item bi ON bi.id=l.batch_item_id JOIN job j ON j.id=q.job_id LEFT JOIN job dj ON dj.id=q.defect_job_id JOIN app_user u ON u.id=q.inspected_by
 LEFT JOIN factory of ON of.id=b.outsource_factory_id
 LEFT JOIN LATERAL (SELECT min(q2.inspected_at) at FROM qc_record q2 WHERE q2.label_id=q.label_id AND q2.job_id=q.job_id AND q2.result='pass' AND q2.inspected_at>q.inspected_at) re ON q.result='fail'`
const qcSelect = `SELECT q.id,l.id,l.bar_code,b.id,b.lot_no,p.full_name,of.name,bi.piece_no,l.piece_index,bi.piece_area::float8,bi.color,j.id,j.name,q.result,q.defect_job_id,dj.name,q.defect_category,q.defect_reason,
 u.display_name,u.staff_no,q.inspected_at,q.device,re.at IS NOT NULL,re.at` + qcFrom

func (s *Store) getQcRecord(ctx context.Context, id int64) (*QcRecord, error) {
	rows, err := s.db.Query(ctx, qcSelect+` WHERE q.id=$1`, id)
	if err != nil {
		return nil, err
	}
	v, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[QcRecord])
	if err != nil {
		return nil, err
	}
	return &v, nil
}

type QcFilter struct {
	BatchID, JobID, DefectJobID *int64
	Result                      string
	Reinspected                 *bool
	Begin, End                  *time.Time
	Keyword, Staff              string
}

type QcSummary struct {
	Inspected, Passed, Failed int
	FailedArea                float64
	Reinspected, Pending      int
}

func (s *Store) ListQcRecords(ctx context.Context, sc auth.Scope, f QcFilter, page, size int) ([]QcRecord, int64, QcSummary, error) {
	where := ` WHERE ($1 OR b.factory_id=ANY($2))`
	args := []any{sc.All, sc.FactoryIDs}
	add := func(q string, v any) { args = append(args, v); where += fmt.Sprintf(q, len(args)) }
	if f.BatchID != nil {
		add(` AND q.batch_id=$%d`, *f.BatchID)
	}
	if f.JobID != nil {
		add(` AND q.job_id=$%d`, *f.JobID)
	}
	if f.DefectJobID != nil {
		add(` AND q.defect_job_id=$%d`, *f.DefectJobID)
	}
	if f.Result != "" {
		add(` AND q.result=$%d`, f.Result)
	}
	if f.Reinspected != nil {
		if *f.Reinspected {
			where += ` AND q.result='fail' AND re.at IS NOT NULL`
		} else {
			where += ` AND q.result='fail' AND re.at IS NULL`
		}
	}
	if f.Begin != nil {
		add(` AND q.inspected_at >= $%d`, *f.Begin)
	}
	if f.End != nil {
		add(` AND q.inspected_at < $%d`, *f.End)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		add(` AND (l.bar_code ILIKE $%[1]d OR b.lot_no ILIKE $%[1]d OR p.full_name ILIKE $%[1]d OR bi.piece_no ILIKE $%[1]d OR q.defect_category ILIKE $%[1]d OR q.defect_reason ILIKE $%[1]d)`, "%"+escapeLike(kw)+"%")
	}
	if st := strings.TrimSpace(f.Staff); st != "" {
		add(` AND (u.display_name ILIKE $%[1]d OR u.staff_no ILIKE $%[1]d OR u.username ILIKE $%[1]d)`, "%"+escapeLike(st)+"%")
	}
	var sum QcSummary
	err := s.db.QueryRow(ctx, `SELECT count(*)::int,count(*) FILTER (WHERE q.result='pass')::int,count(*) FILTER (WHERE q.result='fail')::int,
 coalesce(sum(bi.piece_area) FILTER (WHERE q.result='fail'),0)::float8,
 count(*) FILTER (WHERE q.result='fail' AND re.at IS NOT NULL)::int,count(*) FILTER (WHERE q.result='fail' AND re.at IS NULL)::int`+qcFrom+where, args...).
		Scan(&sum.Inspected, &sum.Passed, &sum.Failed, &sum.FailedArea, &sum.Reinspected, &sum.Pending)
	if err != nil {
		return nil, 0, sum, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, qcSelect+where+fmt.Sprintf(` ORDER BY q.inspected_at DESC,q.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, sum, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[QcRecord])
	return items, int64(sum.Inspected), sum, err
}

type QualityReportRow struct {
	Date                 time.Time
	JobID                int64
	JobName              string
	BatchID              int64
	LotNo, ProjectName   string
	OutsourceFactoryName *string
	DefectJobID          *int64
	DefectJobName        *string
	DefectCategory       string
	DefectReason         string
	Qty                  int
	Area                 float64
	ReinspectedQty       int
	PieceNos, Inspectors string
}

type QualityReportSummary struct {
	Inspected, Passed, Failed int
	FailedArea                float64
	Reinspected, PassRate     int
}

type QualityReportFilter struct {
	From, To           *time.Time
	JobID, DefectJobID *int64
	Keyword            string
}

// QualityReport groups fail inspections by date × QC job × batch × defect.
func (s *Store) QualityReport(ctx context.Context, sc auth.Scope, f QualityReportFilter, page, size int) ([]QualityReportRow, int, QualityReportSummary, error) {
	args := []any{sc.All, sc.FactoryIDs, f.From, f.To}
	where := ` WHERE ($1 OR b.factory_id=ANY($2)) AND ($3::date IS NULL OR q.inspected_at::date >= $3::date) AND ($4::date IS NULL OR q.inspected_at::date <= $4::date)`
	add := func(q string, v any) { args = append(args, v); where += fmt.Sprintf(q, len(args)) }
	if f.JobID != nil {
		add(` AND q.job_id=$%d`, *f.JobID)
	}
	if f.DefectJobID != nil {
		add(` AND q.defect_job_id=$%d`, *f.DefectJobID)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		add(` AND (b.lot_no ILIKE $%[1]d OR p.full_name ILIKE $%[1]d OR q.defect_category ILIKE $%[1]d OR q.defect_reason ILIKE $%[1]d)`, "%"+escapeLike(kw)+"%")
	}
	var sum QualityReportSummary
	err := s.db.QueryRow(ctx, `SELECT count(*)::int,count(*) FILTER (WHERE q.result='pass')::int,count(*) FILTER (WHERE q.result='fail')::int,
 coalesce(sum(bi.piece_area) FILTER (WHERE q.result='fail'),0)::float8,count(*) FILTER (WHERE q.result='fail' AND re.at IS NOT NULL)::int`+qcFrom+where, args...).
		Scan(&sum.Inspected, &sum.Passed, &sum.Failed, &sum.FailedArea, &sum.Reinspected)
	if err != nil {
		return nil, 0, sum, err
	}
	if sum.Inspected > 0 {
		sum.PassRate = sum.Passed * 100 / sum.Inspected
	}
	group := ` GROUP BY q.inspected_at::date,j.id,j.name,j.seq,b.id,b.lot_no,p.full_name,of.name,q.defect_job_id,dj.name,q.defect_category,q.defect_reason`
	failWhere := where + ` AND q.result='fail'`
	var total int
	if err = s.db.QueryRow(ctx, `SELECT count(*) FROM (SELECT 1`+qcFrom+failWhere+group+`) g`, args...).Scan(&total); err != nil {
		return nil, 0, sum, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, `SELECT q.inspected_at::date,j.id,j.name,b.id,b.lot_no,p.full_name,of.name,q.defect_job_id,dj.name,q.defect_category,q.defect_reason,
 count(*)::int,coalesce(sum(bi.piece_area),0)::float8,count(*) FILTER (WHERE re.at IS NOT NULL)::int,
 string_agg(DISTINCT bi.piece_no,',' ORDER BY bi.piece_no),string_agg(DISTINCT u.display_name,',' ORDER BY u.display_name)`+qcFrom+failWhere+group+
		fmt.Sprintf(` ORDER BY q.inspected_at::date DESC,count(*) DESC,j.seq,b.lot_no LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, sum, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[QualityReportRow])
	return items, total, sum, err
}

type DefectJobCell struct {
	JobID           int64
	JobName         string
	FailedQty       int
	FailedArea      float64
	ReinspectedQty  int
	ReinspectedArea float64
	PendingQty      int
	PendingArea     float64
}

type DefectDailyRow struct {
	BatchID              int64
	LotNo, ProjectName   string
	CustomerName         string
	ClientOrderNo        string
	Thickness            float64
	Color, Workshop      string
	OutsourceFactoryName *string
	ItemQty              int
	ItemArea             float64
	FailedQty            int
	FailedArea           float64
	PendingQty           int
	Cells                []DefectJobCell
}

type DefectDailySummary struct {
	Batches, FailedQty int
	FailedArea         float64
	ReinspectedQty     int
	PendingQty         int
	PendingArea        float64
}

type DefectDailyFilter struct {
	From, To           *time.Time
	Workshop           string
	OutsourceFactoryID *int64
	Keyword            string
}

// DefectDailyReport lists batches with fail inspections in range, one cell per QC job.
func (s *Store) DefectDailyReport(ctx context.Context, sc auth.Scope, f DefectDailyFilter, page, size int) ([]Job, []DefectDailyRow, DefectDailySummary, error) {
	all, err := s.Jobs(ctx)
	if err != nil {
		return nil, nil, DefectDailySummary{}, err
	}
	qcJobs := make([]Job, 0, 2)
	for _, j := range all {
		if j.IsQc {
			qcJobs = append(qcJobs, j)
		}
	}
	args := []any{sc.All, sc.FactoryIDs, f.From, f.To}
	// fails: fail inspections in range, each with whether the label later passed that job.
	from := ` FROM batch b JOIN contract c ON c.id=b.contract_id JOIN customer cu ON cu.id=c.customer_id JOIN project p ON p.id=c.project_id
 LEFT JOIN factory of ON of.id=b.outsource_factory_id
 JOIN LATERAL (
  SELECT count(*)::int qty,coalesce(sum(bi.piece_area),0)::float8 area,count(*) FILTER (WHERE re.at IS NOT NULL)::int re_qty,
   count(*) FILTER (WHERE re.at IS NULL)::int pend_qty,coalesce(sum(bi.piece_area) FILTER (WHERE re.at IS NULL),0)::float8 pend_area
  FROM qc_record q JOIN label l ON l.id=q.label_id JOIN batch_item bi ON bi.id=l.batch_item_id
  LEFT JOIN LATERAL (SELECT min(q2.inspected_at) at FROM qc_record q2 WHERE q2.label_id=q.label_id AND q2.job_id=q.job_id AND q2.result='pass' AND q2.inspected_at>q.inspected_at) re ON true
  WHERE q.batch_id=b.id AND q.result='fail' AND NOT l.cancelled
   AND ($3::date IS NULL OR q.inspected_at::date >= $3::date) AND ($4::date IS NULL OR q.inspected_at::date <= $4::date)
 ) fs ON fs.qty>0`
	where := ` WHERE ($1 OR b.factory_id=ANY($2))`
	add := func(q string, v any) { args = append(args, v); where += fmt.Sprintf(q, len(args)) }
	if f.Workshop != "" {
		add(` AND b.workshop=$%d`, f.Workshop)
	}
	if f.OutsourceFactoryID != nil {
		add(` AND b.outsource_factory_id=$%d`, *f.OutsourceFactoryID)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		add(` AND (b.lot_no ILIKE $%[1]d OR p.full_name ILIKE $%[1]d OR cu.name ILIKE $%[1]d OR b.client_order_no ILIKE $%[1]d)`, "%"+escapeLike(kw)+"%")
	}
	var sum DefectDailySummary
	err = s.db.QueryRow(ctx, `SELECT count(*)::int,coalesce(sum(fs.qty),0)::int,coalesce(sum(fs.area),0)::float8,coalesce(sum(fs.re_qty),0)::int,coalesce(sum(fs.pend_qty),0)::int,coalesce(sum(fs.pend_area),0)::float8`+from+where, args...).
		Scan(&sum.Batches, &sum.FailedQty, &sum.FailedArea, &sum.ReinspectedQty, &sum.PendingQty, &sum.PendingArea)
	if err != nil {
		return nil, nil, sum, err
	}
	pageArgs := append(append([]any{}, args...), size, (page-1)*size)
	rows, err := s.db.Query(ctx, `SELECT b.id,b.lot_no,p.full_name,cu.name,b.client_order_no,b.thickness::float8,b.color,b.workshop,of.name,b.item_qty,b.item_area::float8,fs.qty,fs.area,fs.pend_qty`+from+where+
		fmt.Sprintf(` ORDER BY fs.qty DESC,b.lot_no LIMIT $%d OFFSET $%d`, len(pageArgs)-1, len(pageArgs)), pageArgs...)
	if err != nil {
		return nil, nil, sum, err
	}
	type rowHead struct {
		BatchID              int64
		LotNo, ProjectName   string
		CustomerName         string
		ClientOrderNo        string
		Thickness            float64
		Color, Workshop      string
		OutsourceFactoryName *string
		ItemQty              int
		ItemArea             float64
		FailedQty            int
		FailedArea           float64
		PendingQty           int
	}
	heads, err := pgx.CollectRows(rows, pgx.RowToStructByPos[rowHead])
	if err != nil {
		return nil, nil, sum, err
	}
	items := make([]DefectDailyRow, 0, len(heads))
	index := map[int64]int{}
	ids := make([]int64, 0, len(heads))
	for i, h := range heads {
		cells := make([]DefectJobCell, len(qcJobs))
		for k, j := range qcJobs {
			cells[k] = DefectJobCell{JobID: j.ID, JobName: j.Name}
		}
		items = append(items, DefectDailyRow{BatchID: h.BatchID, LotNo: h.LotNo, ProjectName: h.ProjectName, CustomerName: h.CustomerName, ClientOrderNo: h.ClientOrderNo, Thickness: h.Thickness, Color: h.Color, Workshop: h.Workshop, OutsourceFactoryName: h.OutsourceFactoryName, ItemQty: h.ItemQty, ItemArea: h.ItemArea, FailedQty: h.FailedQty, FailedArea: h.FailedArea, PendingQty: h.PendingQty, Cells: cells})
		index[h.BatchID] = i
		ids = append(ids, h.BatchID)
	}
	if len(ids) == 0 {
		return qcJobs, items, sum, nil
	}
	cellRows, err := s.db.Query(ctx, `SELECT q.batch_id,q.job_id,count(*)::int,coalesce(sum(bi.piece_area),0)::float8,
 count(*) FILTER (WHERE re.at IS NOT NULL)::int,coalesce(sum(bi.piece_area) FILTER (WHERE re.at IS NOT NULL),0)::float8,
 count(*) FILTER (WHERE re.at IS NULL)::int,coalesce(sum(bi.piece_area) FILTER (WHERE re.at IS NULL),0)::float8
 FROM qc_record q JOIN label l ON l.id=q.label_id JOIN batch_item bi ON bi.id=l.batch_item_id
 LEFT JOIN LATERAL (SELECT min(q2.inspected_at) at FROM qc_record q2 WHERE q2.label_id=q.label_id AND q2.job_id=q.job_id AND q2.result='pass' AND q2.inspected_at>q.inspected_at) re ON true
 WHERE q.batch_id=ANY($1) AND q.result='fail' AND NOT l.cancelled
  AND ($2::date IS NULL OR q.inspected_at::date >= $2::date) AND ($3::date IS NULL OR q.inspected_at::date <= $3::date)
 GROUP BY q.batch_id,q.job_id`, ids, f.From, f.To)
	if err != nil {
		return nil, nil, sum, err
	}
	defer cellRows.Close()
	for cellRows.Next() {
		var batchID, jobID int64
		var c DefectJobCell
		if err := cellRows.Scan(&batchID, &jobID, &c.FailedQty, &c.FailedArea, &c.ReinspectedQty, &c.ReinspectedArea, &c.PendingQty, &c.PendingArea); err != nil {
			return nil, nil, sum, err
		}
		row := &items[index[batchID]]
		for k := range row.Cells {
			if row.Cells[k].JobID == jobID {
				row.Cells[k].FailedQty, row.Cells[k].FailedArea = c.FailedQty, c.FailedArea
				row.Cells[k].ReinspectedQty, row.Cells[k].ReinspectedArea = c.ReinspectedQty, c.ReinspectedArea
				row.Cells[k].PendingQty, row.Cells[k].PendingArea = c.PendingQty, c.PendingArea
			}
		}
	}
	return qcJobs, items, sum, cellRows.Err()
}
