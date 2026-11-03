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

// Plan errors mapped by the server layer.
var (
	ErrInvalidPlan       = errors.New("invalid plan")
	ErrForbidden         = errors.New("forbidden factory")
	ErrDuplicatePlan     = errors.New("duplicate plan")
	ErrAlreadyAudited    = errors.New("already audited")
	ErrNotAudited        = errors.New("not audited")
	ErrInvalidTransition = errors.New("invalid transition")
)

type Plan struct {
	ID                    int64
	Kind                  string
	FactoryID, BatchID    int64
	LotNo, ProjectName    string
	CustomerName          string
	Thickness             float64
	Color                 string
	OrderQty              int
	OrderArea             float64
	ItemQty               int
	ItemArea              float64
	LabelQty              int
	StartDate, EndDate    time.Time
	Workshop              string
	JobID                 *int64
	JobName               *string
	PaintLine, Shift      string
	SprayType, Urgency    string
	AuditState            string
	AuditedBy             *string
	AuditedAt             *time.Time
	StopState, StopReason string
	Remark                string
	TargetJobID           int64
	TargetJobName         string
	DoneQty               int
	DoneArea              float64
	Progress              int
	LastScanAt            *time.Time
	Overdue               bool
	CreatedBy             string
	CreatedAt, UpdatedAt  time.Time
}

// Progress is computed live: target job by kind (sheet_week → "20", sheet_daily → plan.job_id,
// spray_daily → "40"); done = active labels of the batch scanned at that job.
const planFrom = ` FROM plan pl JOIN batch b ON b.id=pl.batch_id JOIN contract c ON c.id=b.contract_id
 JOIN customer cu ON cu.id=c.customer_id JOIN project p ON p.id=c.project_id
 JOIN app_user u ON u.id=pl.created_by LEFT JOIN app_user au ON au.id=pl.audited_by LEFT JOIN job pj ON pj.id=pl.job_id
 JOIN job tj ON (pl.kind='sheet_daily' AND tj.id=pl.job_id) OR (pl.kind='sheet_week' AND tj.code='20') OR (pl.kind='spray_daily' AND tj.code='40')
 CROSS JOIN LATERAL (SELECT count(*)::int AS qty FROM label l WHERE l.batch_id=b.id AND NOT l.cancelled) lb
 CROSS JOIN LATERAL (
  SELECT count(*)::int AS qty, coalesce(sum(bi.piece_area),0)::float8 AS area, max(sr.scanned_at) AS last
  FROM scan_record sr JOIN label l ON l.id=sr.label_id JOIN batch_item bi ON bi.id=l.batch_item_id
  WHERE sr.batch_id=b.id AND sr.job_id=tj.id AND NOT l.cancelled
 ) dn
 CROSS JOIN LATERAL (SELECT CASE WHEN lb.qty>0 THEN dn.qty*100/lb.qty ELSE 0 END AS progress) pr
 CROSS JOIN LATERAL (SELECT current_date>pl.end_date AND pr.progress<100 AND pl.stop_state<>'stopped' AS overdue) od`
const planSelect = `SELECT pl.id,pl.kind,pl.factory_id,pl.batch_id,b.lot_no,p.full_name,cu.name,b.thickness::float8,b.color,b.order_qty,b.order_area::float8,b.item_qty,b.item_area::float8,lb.qty,
 pl.start_date,pl.end_date,pl.workshop,pl.job_id,pj.name,pl.paint_line,pl.shift,pl.spray_type,pl.urgency,pl.audit_state,au.display_name,pl.audited_at,pl.stop_state,pl.stop_reason,pl.remark,
 tj.id,tj.name,dn.qty,dn.area,pr.progress,dn.last,od.overdue,u.display_name,pl.created_at,pl.updated_at` + planFrom

type PlanFilter struct {
	Kind                         string
	From, To                     *time.Time
	AuditState, StopState        string
	Workshop, PaintLine, Keyword string
	JobID                        *int64
	Completed                    *bool
}

type PlanSummary struct {
	Plans, Qty int
	Area       float64
	DoneQty    int
	DoneArea   float64
	Overdue    int
}

func (s *Store) ListPlans(ctx context.Context, sc auth.Scope, f PlanFilter, page, size int) ([]Plan, PlanSummary, error) {
	where := ` WHERE ($1 OR pl.factory_id=ANY($2)) AND pl.kind=$3`
	args := []any{sc.All, sc.FactoryIDs, f.Kind}
	add := func(q string, v any) { args = append(args, v); where += fmt.Sprintf(q, len(args)) }
	if f.From != nil {
		add(` AND pl.end_date >= $%d`, *f.From)
	}
	if f.To != nil {
		add(` AND pl.start_date <= $%d`, *f.To)
	}
	if f.AuditState != "" {
		add(` AND pl.audit_state=$%d`, f.AuditState)
	}
	if f.StopState != "" {
		add(` AND pl.stop_state=$%d`, f.StopState)
	}
	if f.Workshop != "" {
		add(` AND pl.workshop=$%d`, f.Workshop)
	}
	if f.PaintLine != "" {
		add(` AND pl.paint_line=$%d`, f.PaintLine)
	}
	if f.JobID != nil {
		add(` AND pl.job_id=$%d`, *f.JobID)
	}
	if f.Completed != nil {
		if *f.Completed {
			where += ` AND pr.progress=100`
		} else {
			where += ` AND pr.progress<100`
		}
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		add(` AND (b.lot_no ILIKE $%[1]d OR p.full_name ILIKE $%[1]d OR cu.name ILIKE $%[1]d)`, "%"+escapeLike(kw)+"%")
	}
	var sum PlanSummary
	err := s.db.QueryRow(ctx, `SELECT count(*)::int,coalesce(sum(b.item_qty),0)::int,coalesce(sum(b.item_area),0)::float8,coalesce(sum(dn.qty),0)::int,coalesce(sum(dn.area),0)::float8,(count(*) FILTER (WHERE od.overdue))::int`+planFrom+where, args...).
		Scan(&sum.Plans, &sum.Qty, &sum.Area, &sum.DoneQty, &sum.DoneArea, &sum.Overdue)
	if err != nil {
		return nil, sum, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, planSelect+where+fmt.Sprintf(` ORDER BY pl.start_date DESC,pl.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, sum, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Plan])
	return items, sum, err
}

func (s *Store) GetPlan(ctx context.Context, sc auth.Scope, id int64) (*Plan, error) {
	rows, err := s.db.Query(ctx, planSelect+` WHERE pl.id=$1 AND ($2 OR pl.factory_id=ANY($3))`, id, sc.All, sc.FactoryIDs)
	if err != nil {
		return nil, err
	}
	p, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Plan])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &p, err
}

type PlanCreate struct {
	Kind                        string
	BatchID                     int64
	StartDate, EndDate          time.Time
	Workshop                    string
	JobID                       *int64
	PaintLine, Shift, SprayType string
	Urgency, Remark             string
	CreatedBy                   int64
}

func invalidPlan(msg string) error { return fmt.Errorf("%w: %s", ErrInvalidPlan, msg) }

// validatePlan checks dates and the kind-specific required fields.
func validatePlan(ctx context.Context, tx pgx.Tx, kind string, start, end time.Time, jobID *int64, paintLine, shift string) error {
	if end.Before(start) {
		return invalidPlan("结束日期不能早于开始日期")
	}
	switch kind {
	case "sheet_daily":
		if jobID == nil {
			return invalidPlan("钣金日计划必须指定工序")
		}
		var ok bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM job WHERE id=$1)`, *jobID).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return invalidPlan("工序不存在")
		}
	case "spray_daily":
		if paintLine == "" || shift == "" {
			return invalidPlan("喷涂日计划必须指定喷涂线和班次")
		}
	case "sheet_week":
	default:
		return invalidPlan("计划类型无效")
	}
	return nil
}

// checkDuplicatePlan rejects another un-stopped plan of the same kind+batch(+job for sheet_daily) overlapping the dates.
func checkDuplicatePlan(ctx context.Context, tx pgx.Tx, selfID int64, kind string, batchID int64, jobID *int64, start, end time.Time) error {
	var dup bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM plan WHERE id<>$1 AND kind=$2 AND batch_id=$3 AND ($2<>'sheet_daily' OR job_id=$4)
 AND stop_state<>'stopped' AND start_date<=$6 AND end_date>=$5)`, selfID, kind, batchID, jobID, start, end).Scan(&dup)
	if err != nil {
		return err
	}
	if dup {
		return ErrDuplicatePlan
	}
	return nil
}

func (s *Store) CreatePlan(ctx context.Context, sc auth.Scope, in PlanCreate) (*Plan, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if in.Kind != "sheet_daily" {
		in.JobID = nil
	}
	if err = validatePlan(ctx, tx, in.Kind, in.StartDate, in.EndDate, in.JobID, in.PaintLine, in.Shift); err != nil {
		return nil, err
	}
	for _, v := range []struct{ kind, value string }{{"workshop", in.Workshop}, {"paint_line", in.PaintLine}, {"shift", in.Shift}} {
		if err = CheckDictValue(ctx, tx, v.kind, v.value, ""); err != nil {
			return nil, err
		}
	}
	var factoryID int64
	var sprayingType string
	// Locking the batch serializes concurrent creates on the duplicate check.
	err = tx.QueryRow(ctx, `SELECT factory_id,spraying_type FROM batch WHERE id=$1 FOR UPDATE`, in.BatchID).Scan(&factoryID, &sprayingType)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !sc.Allows(factoryID) {
		return nil, ErrForbidden
	}
	if in.Kind == "spray_daily" && in.SprayType == "" {
		in.SprayType = sprayingType
	}
	if err = checkDuplicatePlan(ctx, tx, 0, in.Kind, in.BatchID, in.JobID, in.StartDate, in.EndDate); err != nil {
		return nil, err
	}
	var id int64
	err = tx.QueryRow(ctx, `INSERT INTO plan (kind,factory_id,batch_id,start_date,end_date,workshop,job_id,paint_line,shift,spray_type,urgency,remark,created_by)
 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id`, in.Kind, factoryID, in.BatchID, in.StartDate, in.EndDate, in.Workshop, in.JobID, in.PaintLine, in.Shift, in.SprayType, in.Urgency, in.Remark, in.CreatedBy).Scan(&id)
	if err != nil {
		return nil, mapErr(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetPlan(ctx, sc, id)
}

// PlanUpdate: nil optional fields keep their current value.
type PlanUpdate struct {
	StartDate, EndDate                    time.Time
	Workshop, PaintLine, Shift, SprayType *string
	JobID                                 *int64
	Urgency, Remark                       *string
}

type planRow struct {
	kind, auditState, stopState string
	batchID                     int64
	jobID                       *int64
	paintLine, shift, workshop  string
}

func lockPlan(ctx context.Context, tx pgx.Tx, sc auth.Scope, id int64) (*planRow, error) {
	var r planRow
	err := tx.QueryRow(ctx, `SELECT kind,audit_state,stop_state,batch_id,job_id,paint_line,shift,workshop FROM plan WHERE id=$1 AND ($2 OR factory_id=ANY($3)) FOR UPDATE`, id, sc.All, sc.FactoryIDs).
		Scan(&r.kind, &r.auditState, &r.stopState, &r.batchID, &r.jobID, &r.paintLine, &r.shift, &r.workshop)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &r, err
}

func (s *Store) UpdatePlan(ctx context.Context, sc auth.Scope, id int64, in PlanUpdate) (*Plan, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	r, err := lockPlan(ctx, tx, sc, id)
	if err != nil {
		return nil, err
	}
	if r.auditState == "audited" {
		return nil, ErrAlreadyAudited
	}
	jobID, paintLine, shift := r.jobID, r.paintLine, r.shift
	if r.kind == "sheet_daily" && in.JobID != nil {
		jobID = in.JobID
	}
	if in.PaintLine != nil {
		paintLine = *in.PaintLine
	}
	if in.Shift != nil {
		shift = *in.Shift
	}
	if err = validatePlan(ctx, tx, r.kind, in.StartDate, in.EndDate, jobID, paintLine, shift); err != nil {
		return nil, err
	}
	workshop := r.workshop
	if in.Workshop != nil {
		workshop = *in.Workshop
	}
	for _, v := range []struct{ kind, value, old string }{{"workshop", workshop, r.workshop}, {"paint_line", paintLine, r.paintLine}, {"shift", shift, r.shift}} {
		if err = CheckDictValue(ctx, tx, v.kind, v.value, v.old); err != nil {
			return nil, err
		}
	}
	if _, err = tx.Exec(ctx, `SELECT 1 FROM batch WHERE id=$1 FOR UPDATE`, r.batchID); err != nil {
		return nil, err
	}
	if r.stopState != "stopped" {
		if err = checkDuplicatePlan(ctx, tx, id, r.kind, r.batchID, jobID, in.StartDate, in.EndDate); err != nil {
			return nil, err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE plan SET start_date=$2,end_date=$3,job_id=$4,paint_line=$5,shift=$6,workshop=coalesce($7,workshop),spray_type=coalesce($8,spray_type),
 urgency=coalesce($9,urgency),remark=coalesce($10,remark),updated_at=now() WHERE id=$1`, id, in.StartDate, in.EndDate, jobID, paintLine, shift, in.Workshop, in.SprayType, in.Urgency, in.Remark)
	if err != nil {
		return nil, mapErr(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetPlan(ctx, sc, id)
}

func (s *Store) DeletePlan(ctx context.Context, sc auth.Scope, id int64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	r, err := lockPlan(ctx, tx, sc, id)
	if err != nil {
		return err
	}
	if r.auditState == "audited" {
		return ErrAlreadyAudited
	}
	if _, err = tx.Exec(ctx, `DELETE FROM plan WHERE id=$1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// AuditPlan audits (audit=true) or reverts to draft (audit=false).
func (s *Store) AuditPlan(ctx context.Context, sc auth.Scope, id int64, audit bool, userID int64) (*Plan, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	r, err := lockPlan(ctx, tx, sc, id)
	if err != nil {
		return nil, err
	}
	if audit {
		if r.auditState == "audited" {
			return nil, ErrAlreadyAudited
		}
		_, err = tx.Exec(ctx, `UPDATE plan SET audit_state='audited',audited_by=$2,audited_at=now(),updated_at=now() WHERE id=$1`, id, userID)
	} else {
		if r.auditState != "audited" {
			return nil, ErrNotAudited
		}
		_, err = tx.Exec(ctx, `UPDATE plan SET audit_state='draft',audited_by=NULL,audited_at=NULL,updated_at=now() WHERE id=$1`, id)
	}
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetPlan(ctx, sc, id)
}

// SetPlanStopState: pause normal→paused, resume paused→normal, stop any non-stopped→stopped.
func (s *Store) SetPlanStopState(ctx context.Context, sc auth.Scope, id int64, action, reason string) (*Plan, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	r, err := lockPlan(ctx, tx, sc, id)
	if err != nil {
		return nil, err
	}
	var next string
	switch {
	case action == "pause" && r.stopState == "normal":
		next = "paused"
	case action == "resume" && r.stopState == "paused":
		next, reason = "normal", ""
	case action == "stop" && r.stopState != "stopped":
		next = "stopped"
	default:
		return nil, ErrInvalidTransition
	}
	if _, err = tx.Exec(ctx, `UPDATE plan SET stop_state=$2,stop_reason=$3,updated_at=now() WHERE id=$1`, id, next, reason); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetPlan(ctx, sc, id)
}
