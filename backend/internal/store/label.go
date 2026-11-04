package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNoItems        = errors.New("no items")
	ErrHasScans       = errors.New("label has scans")
	ErrLabelCancelled = errors.New("label cancelled")
	ErrAlreadyScanned = errors.New("already scanned")
	ErrOutOfOrder     = errors.New("out of order")
)

type Label struct {
	ID                 int64
	BarCode            string
	FactoryID, BatchID int64
	LotNo, ProjectName string
	BatchItemID        int64
	PieceNo            string
	PieceIndex, Qty    int
	DrawingNo, Pattern string
	Thickness          float64
	Color              string
	PieceArea          float64
	Cancelled          bool
	LastJobID          *int64
	LastJobName        *string
	LastScannedAt      *time.Time
	CreatedAt          time.Time
}

const labelFrom = ` FROM label l JOIN batch b ON b.id=l.batch_id JOIN contract c ON c.id=b.contract_id JOIN project p ON p.id=c.project_id JOIN batch_item bi ON bi.id=l.batch_item_id LEFT JOIN LATERAL (SELECT j.id,j.name,sr.scanned_at FROM scan_record sr JOIN job j ON j.id=sr.job_id WHERE sr.label_id=l.id ORDER BY j.seq DESC LIMIT 1) last ON true`
const labelSelect = `SELECT l.id,l.bar_code,l.factory_id,l.batch_id,b.lot_no,p.full_name,l.batch_item_id,bi.piece_no,l.piece_index,bi.qty,bi.drawing_no,bi.pattern,bi.thickness::float8,bi.color,bi.piece_area::float8,l.cancelled,last.id,last.name,last.scanned_at,l.created_at` + labelFrom

func (s *Store) GenerateLabels(ctx context.Context, sc auth.Scope, batchID, userID int64) (int, int, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var count int
	err = tx.QueryRow(ctx, `SELECT item_count FROM batch WHERE id=$1 AND ($2 OR factory_id=ANY($3)) FOR UPDATE`, batchID, sc.All, sc.FactoryIDs).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, ErrNotFound
	}
	if err != nil {
		return 0, 0, err
	}
	if count == 0 {
		return 0, 0, ErrNoItems
	}
	// ON CONFLICT only on the item/index key: barcode collisions must not be silently ignored.
	tag, err := tx.Exec(ctx, `WITH ins AS (INSERT INTO label (bar_code,factory_id,batch_id,batch_item_id,piece_index,created_by)
 SELECT lpad((b.id%10000)::text,4,'0') || lpad(bi.seq::text,3,'0') || lpad(g.i::text,3,'0') || lpad(((b.id/10000)%1000)::text,3,'0'), b.factory_id,b.id,bi.id,g.i,$2
 FROM batch b JOIN batch_item bi ON bi.batch_id=b.id CROSS JOIN LATERAL generate_series(1,bi.qty) AS g(i)
 WHERE b.id=$1 ON CONFLICT (batch_item_id,piece_index) DO NOTHING RETURNING id) INSERT INTO label_log(label_id,op_type,operator_id) SELECT id,'generate',$2 FROM ins`, batchID, userID)
	if err != nil {
		return 0, 0, mapErr(err)
	}
	var total int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM label WHERE batch_id=$1 AND NOT cancelled`, batchID).Scan(&total); err != nil {
		return 0, 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	return int(tag.RowsAffected()), total, nil
}

type LabelFilter struct {
	BatchID   *int64
	Keyword   string
	Cancelled *bool
}

func (s *Store) ListLabels(ctx context.Context, sc auth.Scope, f LabelFilter, page, size int) ([]Label, int64, error) {
	where := ` WHERE ($1 OR l.factory_id=ANY($2))`
	args := []any{sc.All, sc.FactoryIDs}
	add := func(clause string, v any) { args = append(args, v); where += fmt.Sprintf(clause, len(args)) }
	if f.BatchID != nil {
		add(` AND l.batch_id=$%d`, *f.BatchID)
	}
	if f.Cancelled != nil {
		add(` AND l.cancelled=$%d`, *f.Cancelled)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		add(` AND (l.bar_code ILIKE $%[1]d OR bi.piece_no ILIKE $%[1]d OR b.lot_no ILIKE $%[1]d OR p.full_name ILIKE $%[1]d)`, "%"+escapeLike(kw)+"%")
	}
	var total int64
	if err := s.db.QueryRow(ctx, `SELECT count(*)`+labelFrom+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, labelSelect+where+fmt.Sprintf(` ORDER BY l.created_at DESC,l.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Label])
	return items, total, err
}
func (s *Store) GetLabel(ctx context.Context, sc auth.Scope, barCode string) (*Label, []Scan, error) {
	rows, err := s.db.Query(ctx, labelSelect+` WHERE l.bar_code=$1 AND ($2 OR l.factory_id=ANY($3))`, barCode, sc.All, sc.FactoryIDs)
	if err != nil {
		return nil, nil, err
	}
	l, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Label])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	scans, _, err := s.ListScans(ctx, sc, ScanFilter{LabelID: &l.ID}, 1, 1000000)
	return &l, scans, err
}
func (s *Store) CancelLabel(ctx context.Context, sc auth.Scope, barCode string, userID int64, reason string) (*Label, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id int64
	var cancelled bool
	err = tx.QueryRow(ctx, `SELECT l.id,l.cancelled FROM label l JOIN batch b ON b.id=l.batch_id WHERE l.bar_code=$1 AND ($2 OR b.factory_id=ANY($3)) FOR UPDATE OF b`, barCode, sc.All, sc.FactoryIDs).Scan(&id, &cancelled)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if cancelled {
		return nil, ErrConflict
	}
	var hasScans bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM scan_record WHERE label_id=$1)`, id).Scan(&hasScans); err != nil {
		return nil, err
	}
	if hasScans {
		return nil, ErrHasScans
	}
	if _, err = tx.Exec(ctx, `UPDATE label SET cancelled=true WHERE id=$1`, id); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO label_log(label_id,op_type,reason,operator_id) VALUES($1,'cancel',$2,$3)`, id, reason, userID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	l, _, err := s.GetLabel(ctx, sc, barCode)
	return l, err
}

type Scan struct {
	ID, LabelID        int64
	BarCode            string
	BatchID            int64
	LotNo, ProjectName string
	PieceNo            string
	PieceIndex         int
	PieceArea          float64
	JobID              int64
	JobName, ScannedBy string
	StaffNo            string
	ScannedAt          time.Time
	Device             string
	MachineID          *int64
	MachineName        *string
}

const scanFrom = ` FROM scan_record sr JOIN label l ON l.id=sr.label_id JOIN batch b ON b.id=sr.batch_id JOIN contract c ON c.id=b.contract_id JOIN project p ON p.id=c.project_id JOIN batch_item bi ON bi.id=l.batch_item_id JOIN job j ON j.id=sr.job_id JOIN app_user u ON u.id=sr.scanned_by LEFT JOIN machine m ON m.id=sr.machine_id`
const scanSelect = `SELECT sr.id,l.id,l.bar_code,b.id,b.lot_no,p.full_name,bi.piece_no,l.piece_index,bi.piece_area::float8,j.id,j.name,u.display_name,u.staff_no,sr.scanned_at,sr.device,sr.machine_id,m.name` + scanFrom

type ScanFilter struct {
	BatchID, JobID, LabelID, MachineID *int64
	Begin, End                         *time.Time
	Keyword, Staff                     string
	// Newest sorts by scanned_at desc (list screens); default is chronological (label timeline).
	Newest bool
}

func (s *Store) ListScans(ctx context.Context, sc auth.Scope, f ScanFilter, page, size int) ([]Scan, int64, error) {
	where := ` WHERE ($1 OR b.factory_id=ANY($2))`
	args := []any{sc.All, sc.FactoryIDs}
	add := func(clause string, v any) { args = append(args, v); where += fmt.Sprintf(clause, len(args)) }
	if f.BatchID != nil {
		add(` AND sr.batch_id=$%d`, *f.BatchID)
	}
	if f.JobID != nil {
		add(` AND sr.job_id=$%d`, *f.JobID)
	}
	if f.LabelID != nil {
		add(` AND sr.label_id=$%d`, *f.LabelID)
	}
	if f.MachineID != nil {
		add(` AND sr.machine_id=$%d`, *f.MachineID)
	}
	if f.Begin != nil {
		add(` AND sr.scanned_at >= $%d`, *f.Begin)
	}
	if f.End != nil {
		add(` AND sr.scanned_at < $%d`, *f.End)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		add(` AND (l.bar_code ILIKE $%d OR b.lot_no ILIKE $%[1]d OR p.full_name ILIKE $%[1]d OR bi.piece_no ILIKE $%[1]d)`, "%"+kw+"%")
	}
	if st := strings.TrimSpace(f.Staff); st != "" {
		add(` AND (u.display_name ILIKE $%d OR u.staff_no ILIKE $%[1]d OR u.username ILIKE $%[1]d)`, "%"+st+"%")
	}
	var total int64
	if err := s.db.QueryRow(ctx, `SELECT count(*)`+scanFrom+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	order := ` ORDER BY sr.scanned_at ASC,sr.id ASC`
	if f.Newest {
		order = ` ORDER BY sr.scanned_at DESC,sr.id DESC`
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, scanSelect+where+order+fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Scan])
	return items, total, err
}

type JobProgress struct {
	JobID            int64
	JobCode, JobName string
	Seq, ScannedQty  int
	ScannedArea      float64
}
type Progress struct {
	BatchID   int64
	Status    string
	TotalQty  int
	TotalArea float64
	LabelQty  int
	Jobs      []JobProgress
}

func (s *Store) BatchProgress(ctx context.Context, sc auth.Scope, batchID int64) (*Progress, error) {
	p := &Progress{BatchID: batchID, Jobs: []JobProgress{}}
	err := s.db.QueryRow(ctx, `SELECT status,item_qty,item_area::float8,(SELECT count(*) FROM label WHERE batch_id=b.id AND NOT cancelled) FROM batch b WHERE id=$1 AND ($2 OR factory_id=ANY($3))`, batchID, sc.All, sc.FactoryIDs).Scan(&p.Status, &p.TotalQty, &p.TotalArea, &p.LabelQty)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `SELECT j.id,j.code,j.name,j.seq,count(sr.id)::int,coalesce(sum(bi.piece_area),0)::float8 FROM job j LEFT JOIN scan_record sr ON sr.job_id=j.id AND sr.batch_id=$1 LEFT JOIN label l ON l.id=sr.label_id LEFT JOIN batch_item bi ON bi.id=l.batch_item_id GROUP BY j.id ORDER BY j.seq`, batchID)
	if err != nil {
		return nil, err
	}
	jobs, err := pgx.CollectRows(rows, pgx.RowToStructByPos[JobProgress])
	if err != nil {
		return nil, err
	}
	p.Jobs = jobs
	return p, nil
}

// ScanInput is one scan at a job. For QC jobs Result must be pass or fail; a fail records a qc_record only
// and leaves the piece at the previous job so it can be re-inspected later.
type ScanInput struct {
	BarCode, JobCode, Device     string
	UserID                       int64
	Result                       string
	DefectJobID                  *int64
	MachineID                    *int64
	DefectCategory, DefectReason string
}

var ErrQcResultRequired = errors.New("qc result required")
var ErrInvalidMachine = errors.New("invalid machine")

func (s *Store) Scan(ctx context.Context, sc auth.Scope, in ScanInput) (*Scan, *QcRecord, *Label, *Progress, error) {
	barCode := in.BarCode
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var labelID, batchID, factoryID int64
	var cancelled bool
	err = tx.QueryRow(ctx, `SELECT l.id,b.id,l.factory_id,l.cancelled FROM label l JOIN batch b ON b.id=l.batch_id WHERE l.bar_code=$1 AND ($2 OR b.factory_id=ANY($3)) FOR UPDATE OF b`, barCode, sc.All, sc.FactoryIDs).Scan(&labelID, &batchID, &factoryID, &cancelled)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if cancelled {
		return nil, nil, nil, nil, ErrLabelCancelled
	}
	var jobID int64
	var seq int
	var isQc bool
	err = tx.QueryRow(ctx, `SELECT id,seq,is_qc FROM job WHERE code=$1`, in.JobCode).Scan(&jobID, &seq, &isQc)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil, nil, ErrBadRef
	}
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if in.MachineID != nil {
		var machineJob, machineFactory int64
		var stopped bool
		err = tx.QueryRow(ctx, `SELECT job_id,factory_id,is_stop FROM machine WHERE id=$1`, *in.MachineID).Scan(&machineJob, &machineFactory, &stopped)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && (stopped || machineJob != jobID || machineFactory != factoryID)) {
			return nil, nil, nil, nil, ErrInvalidMachine
		}
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}
	if isQc && in.Result != "pass" && in.Result != "fail" {
		return nil, nil, nil, nil, ErrQcResultRequired
	}
	if in.DefectJobID != nil {
		var ok bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM job WHERE id=$1)`, *in.DefectJobID).Scan(&ok); err != nil {
			return nil, nil, nil, nil, err
		}
		if !ok {
			return nil, nil, nil, nil, ErrBadRef
		}
	}
	var already bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM scan_record WHERE label_id=$1 AND job_id=$2)`, labelID, jobID).Scan(&already); err != nil {
		return nil, nil, nil, nil, err
	}
	if already {
		return nil, nil, nil, nil, ErrAlreadyScanned
	}
	if seq > 1 {
		var previous string
		var scanned bool
		err = tx.QueryRow(ctx, `SELECT j.name,EXISTS(SELECT 1 FROM scan_record sr WHERE sr.label_id=$1 AND sr.job_id=j.id) FROM job j WHERE j.seq=$2`, labelID, seq-1).Scan(&previous, &scanned)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if !scanned {
			return nil, nil, nil, nil, fmt.Errorf("%w: %s", ErrOutOfOrder, previous)
		}
	}
	var qcID *int64
	if isQc {
		var id int64
		err = tx.QueryRow(ctx, `INSERT INTO qc_record (label_id,batch_id,job_id,result,defect_job_id,defect_category,defect_reason,inspected_by,device) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
			labelID, batchID, jobID, in.Result, in.DefectJobID, in.DefectCategory, in.DefectReason, in.UserID, in.Device).Scan(&id)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		qcID = &id
	}
	var scanID *int64
	if !isQc || in.Result == "pass" {
		var id int64
		err = tx.QueryRow(ctx, `INSERT INTO scan_record (label_id,batch_id,job_id,scanned_by,device,machine_id) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`, labelID, batchID, jobID, in.UserID, in.Device, in.MachineID).Scan(&id)
		if err != nil {
			return nil, nil, nil, nil, mapErr(err)
		}
		scanID = &id
		if err = updateBatchProgress(ctx, tx, batchID, jobID); err != nil {
			return nil, nil, nil, nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, nil, nil, nil, err
	}
	var scan *Scan
	if scanID != nil {
		rows, err := s.db.Query(ctx, scanSelect+` WHERE sr.id=$1`, *scanID)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		v, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Scan])
		if err != nil {
			return nil, nil, nil, nil, err
		}
		scan = &v
	}
	var qc *QcRecord
	if qcID != nil {
		if qc, err = s.getQcRecord(ctx, *qcID); err != nil {
			return nil, nil, nil, nil, err
		}
	}
	label, _, err := s.GetLabel(ctx, sc, barCode)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	progress, err := s.BatchProgress(ctx, sc, batchID)
	return scan, qc, label, progress, err
}
