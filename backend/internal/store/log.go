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

type LabelLog struct {
	ID, LabelID                           int64
	OpType, Reason, OperatorName, StaffNo string
	CreatedAt                             time.Time
	BarCode                               string
	FactoryID, BatchID                    int64
	LotNo, ProjectName, PieceNo           string
	PieceIndex, Qty                       int
	DrawingNo, Pattern                    string
	Thickness                             float64
	Color                                 string
	PieceArea                             float64
}
type LabelLogFilter struct {
	From, To          *time.Time
	OpType            string
	BatchID           *int64
	Keyword, Operator string
	LabelID           *int64
}
type LabelLogTotals struct {
	GenerateQty, CancelQty, ReprintQty int64
	Area                               float64
}

const labelLogFrom = ` FROM label_log ll JOIN label l ON l.id=ll.label_id JOIN batch b ON b.id=l.batch_id JOIN contract c ON c.id=b.contract_id JOIN project p ON p.id=c.project_id JOIN batch_item bi ON bi.id=l.batch_item_id JOIN app_user u ON u.id=ll.operator_id`
const labelLogSelect = `SELECT ll.id,ll.label_id,ll.op_type,ll.reason,u.display_name,u.staff_no,ll.created_at,l.bar_code,l.factory_id,l.batch_id,b.lot_no,p.full_name,bi.piece_no,l.piece_index,bi.qty,bi.drawing_no,bi.pattern,bi.thickness::float8,bi.color,bi.piece_area::float8` + labelLogFrom

func (s *Store) ListLabelLogs(ctx context.Context, sc auth.Scope, f LabelLogFilter, page, size int) ([]LabelLog, int64, LabelLogTotals, error) {
	where := ` WHERE ($1 OR l.factory_id=ANY($2))`
	args := []any{sc.All, sc.FactoryIDs}
	add := func(clause string, v any) { args = append(args, v); where += fmt.Sprintf(clause, len(args)) }
	if f.From != nil {
		add(` AND ll.created_at >= $%d`, *f.From)
	}
	if f.To != nil {
		add(` AND ll.created_at < $%d`, f.To.AddDate(0, 0, 1))
	}
	if f.OpType != "" {
		add(` AND ll.op_type=$%d`, f.OpType)
	}
	if f.BatchID != nil {
		add(` AND l.batch_id=$%d`, *f.BatchID)
	}
	if f.LabelID != nil {
		add(` AND ll.label_id=$%d`, *f.LabelID)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		add(` AND (l.bar_code ILIKE $%[1]d OR b.lot_no ILIKE $%[1]d OR p.full_name ILIKE $%[1]d OR bi.piece_no ILIKE $%[1]d OR bi.drawing_no ILIKE $%[1]d)`, "%"+escapeLike(kw)+"%")
	}
	if kw := strings.TrimSpace(f.Operator); kw != "" {
		add(` AND (u.display_name ILIKE $%[1]d OR u.staff_no ILIKE $%[1]d OR u.username ILIKE $%[1]d)`, "%"+escapeLike(kw)+"%")
	}
	var total int64
	var totals LabelLogTotals
	err := s.db.QueryRow(ctx, `SELECT count(*),count(*) FILTER (WHERE ll.op_type='generate'),count(*) FILTER (WHERE ll.op_type='cancel'),count(*) FILTER (WHERE ll.op_type='reprint'),coalesce(sum(bi.piece_area),0)::float8`+labelLogFrom+where, args...).Scan(&total, &totals.GenerateQty, &totals.CancelQty, &totals.ReprintQty, &totals.Area)
	if err != nil {
		return nil, 0, totals, err
	}
	order := ` ORDER BY ll.created_at DESC,ll.id DESC`
	if f.LabelID != nil {
		order = ` ORDER BY ll.created_at ASC,ll.id ASC`
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, labelLogSelect+where+order+fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, totals, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[LabelLog])
	return items, total, totals, err
}
func (s *Store) ReprintLabel(ctx context.Context, sc auth.Scope, barCode string, userID int64, reason string) (*Label, error) {
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
		return nil, ErrLabelCancelled
	}
	_, err = tx.Exec(ctx, `INSERT INTO label_log(label_id,op_type,reason,operator_id) VALUES($1,'reprint',$2,$3)`, id, reason, userID)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	l, _, err := s.GetLabel(ctx, sc, barCode)
	return l, err
}

type AuditLog struct {
	ID                                         int64
	UserID                                     *int64
	Username, DisplayName, Method, Route, Path string
	Status, DurationMs                         int
	IP                                         string
	CreatedAt                                  time.Time
}
type AuditFilter struct {
	From, To                *time.Time
	Keyword, Method, Result string
}

func (s *Store) WriteAudit(ctx context.Context, a AuditLog) error {
	_, err := s.db.Exec(ctx, `INSERT INTO audit_log(user_id,username,display_name,method,route,path,status,duration_ms,ip) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, a.UserID, a.Username, a.DisplayName, a.Method, a.Route, a.Path, a.Status, a.DurationMs, a.IP)
	return err
}
func (s *Store) ListAuditLogs(ctx context.Context, f AuditFilter, page, size int) ([]AuditLog, int64, error) {
	where := ` WHERE true`
	args := []any{}
	add := func(clause string, v any) { args = append(args, v); where += fmt.Sprintf(clause, len(args)) }
	if f.From != nil {
		add(` AND created_at >= $%d`, *f.From)
	}
	if f.To != nil {
		add(` AND created_at < $%d`, f.To.AddDate(0, 0, 1))
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		add(` AND (username ILIKE $%[1]d OR display_name ILIKE $%[1]d OR path ILIKE $%[1]d)`, "%"+escapeLike(kw)+"%")
	}
	if f.Method != "" {
		add(` AND method=$%d`, f.Method)
	}
	switch f.Result {
	case "ok":
		where += ` AND status<400`
	case "fail":
		where += ` AND status>=400`
	}
	var total int64
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM audit_log`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, `SELECT id,user_id,username,display_name,method,route,path,status,duration_ms,ip,created_at FROM audit_log`+where+fmt.Sprintf(` ORDER BY created_at DESC,id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[AuditLog])
	return items, total, err
}
