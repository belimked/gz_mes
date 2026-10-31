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

type BatchValuation struct {
	BatchID                                  int64
	LotNo                                    string
	ContractID                               int64
	ContractNo                               string
	CustomerID                               int64
	CustomerName, ProjectName                string
	Thickness                                float64
	Color                                    string
	OrderQty                                 int
	OrderArea, ItemArea                      float64
	DeliveredQty                             int
	DeliveredArea                            float64
	Status, Basis                            string
	BasisArea, UnitPrice, IngotPrice, Amount float64
	Remark                                   string
	Audited                                  bool
	AuditedBy                                *string
	AuditedAt                                *time.Time
	UpdatedBy                                *string
	UpdatedAt                                *time.Time
	CreatedAt                                time.Time
}

type BatchValuationFilter struct {
	Keyword                string
	ContractID, CustomerID *int64
	Audited                *bool
	Begin, End             *time.Time
}

type BatchValuationInput struct {
	Basis                 string
	UnitPrice, IngotPrice float64
	Remark                string
}

const valuationFrom = ` FROM batch b JOIN contract c ON c.id=b.contract_id JOIN customer cu ON cu.id=c.customer_id JOIN project p ON p.id=c.project_id
 LEFT JOIN batch_valuation v ON v.batch_id=b.id
 LEFT JOIN app_user au ON au.id=v.audited_by LEFT JOIN app_user uu ON uu.id=v.updated_by
 LEFT JOIN LATERAL (
  SELECT count(dnl.label_id)::int AS qty, coalesce(sum(bi.piece_area),0) AS area
  FROM delivery_note_label dnl JOIN label l ON l.id=dnl.label_id JOIN batch_item bi ON bi.id=l.batch_item_id
  WHERE l.batch_id=b.id
 ) shipped ON true`
const valuationSelect = `SELECT b.id,b.lot_no,b.contract_id,c.contract_no,c.customer_id,cu.name,p.full_name,b.thickness::float8,b.color,b.order_qty,b.order_area::float8,b.item_area::float8,
 shipped.qty,shipped.area::float8,b.status,coalesce(v.basis,'order_area'),coalesce(v.basis_area,b.order_area)::float8,
 coalesce(v.unit_price,0)::float8,coalesce(v.ingot_price,0)::float8,coalesce(v.amount,0)::float8,coalesce(v.remark,''),
 coalesce(v.audited,false),au.display_name,v.audited_at,uu.display_name,v.updated_at,b.created_at` + valuationFrom

func (s *Store) getBatchValuation(ctx context.Context, sc auth.Scope, id int64) (*BatchValuation, error) {
	rows, err := s.db.Query(ctx, valuationSelect+` WHERE b.id=$1 AND ($2 OR b.factory_id=ANY($3))`, id, sc.All, sc.FactoryIDs)
	if err != nil {
		return nil, err
	}
	v, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[BatchValuation])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &v, err
}

func (s *Store) ListBatchValuations(ctx context.Context, sc auth.Scope, f BatchValuationFilter, page, size int) ([]BatchValuation, int64, float64, error) {
	where := ` WHERE ($1 OR b.factory_id=ANY($2))`
	args := []any{sc.All, sc.FactoryIDs}
	add := func(q string, v any) { args = append(args, v); where += fmt.Sprintf(q, len(args)) }
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		add(` AND (b.lot_no ILIKE $%[1]d OR c.contract_no ILIKE $%[1]d OR p.full_name ILIKE $%[1]d OR cu.name ILIKE $%[1]d)`, "%"+escapeLike(kw)+"%")
	}
	if f.ContractID != nil {
		add(` AND b.contract_id=$%d`, *f.ContractID)
	}
	if f.CustomerID != nil {
		add(` AND c.customer_id=$%d`, *f.CustomerID)
	}
	if f.Audited != nil {
		add(` AND coalesce(v.audited,false)=$%d`, *f.Audited)
	}
	if f.Begin != nil {
		add(` AND b.created_at >= $%d`, *f.Begin)
	}
	if f.End != nil {
		add(` AND b.created_at < ($%d::date + interval '1 day')`, *f.End)
	}
	var total int64
	var totalAmount float64
	// The aggregate uses the same joins and filters as the page (including unvalued batches).
	err := s.db.QueryRow(ctx, `SELECT count(*),coalesce(sum(v.amount),0)::float8`+valuationFrom+where, args...).Scan(&total, &totalAmount)
	if err != nil {
		return nil, 0, 0, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, valuationSelect+where+fmt.Sprintf(` ORDER BY b.created_at DESC,b.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, 0, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[BatchValuation])
	return items, total, totalAmount, err
}

func (s *Store) SetBatchValuation(ctx context.Context, sc auth.Scope, batchID int64, in BatchValuationInput, userID int64) (*BatchValuation, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id int64
	err = tx.QueryRow(ctx, `SELECT id FROM batch WHERE id=$1 AND ($2 OR factory_id=ANY($3)) FOR UPDATE`, batchID, sc.All, sc.FactoryIDs).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var audited bool
	err = tx.QueryRow(ctx, `SELECT audited FROM batch_valuation WHERE batch_id=$1`, batchID).Scan(&audited)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if audited {
		return nil, ErrConflict
	}
	// Calculate with PostgreSQL NUMERIC to avoid binary floating-point rounding of money.
	_, err = tx.Exec(ctx, `INSERT INTO batch_valuation(batch_id,basis,basis_area,unit_price,ingot_price,amount,remark,updated_by)
 SELECT $1,$2,priced.area,$3,$4,round(priced.area*$3::numeric,2),$5,$6
 FROM batch b CROSS JOIN LATERAL (SELECT CASE $2 WHEN 'order_area' THEN b.order_area WHEN 'item_area' THEN b.item_area ELSE (
  SELECT coalesce(sum(bi.piece_area),0) FROM delivery_note_label dnl JOIN label l ON l.id=dnl.label_id JOIN batch_item bi ON bi.id=l.batch_item_id WHERE l.batch_id=b.id
 ) END AS area) priced WHERE b.id=$1
 ON CONFLICT(batch_id) DO UPDATE SET basis=excluded.basis,basis_area=excluded.basis_area,unit_price=excluded.unit_price,ingot_price=excluded.ingot_price,amount=excluded.amount,remark=excluded.remark,updated_by=excluded.updated_by,updated_at=now()`, batchID, in.Basis, in.UnitPrice, in.IngotPrice, in.Remark, userID)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.getBatchValuation(ctx, sc, batchID)
}

func (s *Store) AuditBatchValuation(ctx context.Context, sc auth.Scope, batchID, userID int64) (*BatchValuation, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id int64
	err = tx.QueryRow(ctx, `SELECT id FROM batch WHERE id=$1 AND ($2 OR factory_id=ANY($3)) FOR UPDATE`, batchID, sc.All, sc.FactoryIDs).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var audited bool
	err = tx.QueryRow(ctx, `SELECT audited FROM batch_valuation WHERE batch_id=$1`, batchID).Scan(&audited)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if audited {
		return nil, ErrConflict
	}
	if _, err = tx.Exec(ctx, `UPDATE batch_valuation SET audited=true,audited_by=$2,audited_at=now() WHERE batch_id=$1`, batchID, userID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.getBatchValuation(ctx, sc, batchID)
}
