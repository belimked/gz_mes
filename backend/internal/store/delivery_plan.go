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

type DeliveryPlan struct {
	ID                                                                                                          int64
	PlanNo                                                                                                      string
	FactoryID                                                                                                   int64
	LogisticsCompany, Contact, Phone, ShippingAddress, ReceivingAddress, CarModel, PlateNo, Driver, DriverPhone string
	Cost, ExtraCost                                                                                             float64
	EstimatedDepartureDate                                                                                      time.Time
	ConfirmedDepartureDate                                                                                      *time.Time
	ConfirmedBy                                                                                                 *string
	ConfirmedAt                                                                                                 *time.Time
	Remark                                                                                                      string
	CreatedAt                                                                                                   time.Time
	NoteCount, Qty                                                                                              int
	Area                                                                                                        float64
}
type DeliveryPlanInput struct {
	FactoryID                                                                                                   int64
	LogisticsCompany, Contact, Phone, ShippingAddress, ReceivingAddress, CarModel, PlateNo, Driver, DriverPhone string
	Cost, ExtraCost                                                                                             float64
	EstimatedDepartureDate                                                                                      time.Time
	Remark                                                                                                      string
	NoteIDs                                                                                                     []int64
}
type DeliveryPlanFilter struct {
	CreatedFrom, CreatedTo, EstimatedFrom, EstimatedTo *time.Time
	Confirmed                                          *bool
	Keyword                                            string
}
type DeliveryPlanSummary struct {
	Plans, Confirmed int
	Cost, ExtraCost  float64
	Qty              int
	Area             float64
}

const planDeliveryFrom = ` FROM delivery_plan p LEFT JOIN app_user u ON u.id=p.confirmed_by LEFT JOIN LATERAL (SELECT count(*)::int note_count,coalesce(sum(qty),0)::int qty,coalesce(sum(area),0)::float8 area FROM delivery_note WHERE plan_id=p.id) n ON true`
const planDeliverySelect = `SELECT p.id,p.plan_no,p.factory_id,p.logistics_company,p.contact,p.phone,p.shipping_address,p.receiving_address,p.car_model,p.plate_no,p.driver,p.driver_phone,p.cost::float8,p.extra_cost::float8,p.estimated_departure_date,p.confirmed_departure_date,u.display_name,p.confirmed_at,p.remark,p.created_at,n.note_count,n.qty,n.area` + planDeliveryFrom

func (s *Store) ListDeliveryPlans(ctx context.Context, sc auth.Scope, f DeliveryPlanFilter, page, size int) ([]DeliveryPlan, DeliveryPlanSummary, error) {
	where := ` WHERE ($1 OR p.factory_id=ANY($2))`
	args := []any{sc.All, sc.FactoryIDs}
	add := func(q string, v any) { args = append(args, v); where += fmt.Sprintf(q, len(args)) }
	if f.CreatedFrom != nil {
		add(` AND p.created_at >= $%d`, *f.CreatedFrom)
	}
	if f.CreatedTo != nil {
		add(` AND p.created_at < $%d`, f.CreatedTo.AddDate(0, 0, 1))
	}
	if f.EstimatedFrom != nil {
		add(` AND p.estimated_departure_date >= $%d`, *f.EstimatedFrom)
	}
	if f.EstimatedTo != nil {
		add(` AND p.estimated_departure_date <= $%d`, *f.EstimatedTo)
	}
	if f.Confirmed != nil {
		add(` AND (p.confirmed_at IS NOT NULL)=$%d`, *f.Confirmed)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		add(` AND (p.plan_no ILIKE $%[1]d OR p.logistics_company ILIKE $%[1]d OR p.plate_no ILIKE $%[1]d OR p.driver ILIKE $%[1]d OR EXISTS (SELECT 1 FROM delivery_note d WHERE d.plan_id=p.id AND d.note_no ILIKE $%[1]d))`, "%"+escapeLike(kw)+"%")
	}
	var sum DeliveryPlanSummary
	err := s.db.QueryRow(ctx, `SELECT count(*)::int,count(*) FILTER (WHERE p.confirmed_at IS NOT NULL)::int,coalesce(sum(p.cost),0)::float8,coalesce(sum(p.extra_cost),0)::float8,coalesce(sum(n.qty),0)::int,coalesce(sum(n.area),0)::float8`+planDeliveryFrom+where, args...).Scan(&sum.Plans, &sum.Confirmed, &sum.Cost, &sum.ExtraCost, &sum.Qty, &sum.Area)
	if err != nil {
		return nil, sum, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, planDeliverySelect+where+fmt.Sprintf(` ORDER BY p.created_at DESC,p.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, sum, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[DeliveryPlan])
	return items, sum, err
}
func (s *Store) GetDeliveryPlan(ctx context.Context, sc auth.Scope, id int64) (*DeliveryPlan, []DeliveryNote, error) {
	rows, err := s.db.Query(ctx, planDeliverySelect+` WHERE p.id=$1 AND ($2 OR p.factory_id=ANY($3))`, id, sc.All, sc.FactoryIDs)
	if err != nil {
		return nil, nil, err
	}
	v, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[DeliveryPlan])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	rows, err = s.db.Query(ctx, noteSelect+` WHERE d.plan_id=$1 ORDER BY d.id`, id)
	if err != nil {
		return nil, nil, err
	}
	notes, err := pgx.CollectRows(rows, pgx.RowToStructByPos[DeliveryNote])
	return &v, notes, err
}

// Assignment is serialized by locking the selected note rows. The unique id list is checked before mutation.
func assignPlanNotes(ctx context.Context, tx pgx.Tx, id, factoryID int64, ids []int64) error {
	seen := map[int64]bool{}
	for _, n := range ids {
		if n <= 0 || seen[n] {
			return fmt.Errorf("%w: 发货单编号重复或无效", ErrBadRef)
		}
		seen[n] = true
	}
	rows, err := tx.Query(ctx, `SELECT id,note_no,factory_id,plan_id FROM delivery_note WHERE id=ANY($1) ORDER BY id FOR UPDATE`, ids)
	if err != nil {
		return err
	}
	count := 0
	for rows.Next() {
		var n, f int64
		var no string
		var other *int64
		if err = rows.Scan(&n, &no, &f, &other); err != nil {
			rows.Close()
			return err
		}
		count++
		if f != factoryID || (other != nil && *other != id) {
			rows.Close()
			return fmt.Errorf("%w: 发货单 %s 不属于该厂区或已关联其他计划", ErrBadRef, no)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if count != len(ids) {
		return fmt.Errorf("%w: 发货单号不存在", ErrBadRef)
	}
	if _, err = tx.Exec(ctx, `UPDATE delivery_note SET plan_id=NULL WHERE plan_id=$1 AND NOT (id=ANY($2))`, id, ids); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE delivery_note SET plan_id=$1 WHERE id=ANY($2)`, id, ids)
	return err
}
func (s *Store) SaveDeliveryPlan(ctx context.Context, sc auth.Scope, id int64, in DeliveryPlanInput, userID int64) (*DeliveryPlan, []DeliveryNote, error) {
	if in.FactoryID <= 0 || in.EstimatedDepartureDate.IsZero() || in.Cost < 0 || in.ExtraCost < 0 {
		return nil, nil, ErrBadRef
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if !sc.All {
		valid := false
		for _, f := range sc.FactoryIDs {
			if f == in.FactoryID {
				valid = true
			}
		}
		if !valid {
			return nil, nil, ErrBadRef
		}
	}
	if id == 0 {
		no, e := nextDocNo(ctx, tx, "WL", "delivery_plan")
		if e != nil {
			return nil, nil, e
		}
		err = tx.QueryRow(ctx, `INSERT INTO delivery_plan(plan_no,factory_id,created_by,estimated_departure_date) VALUES($1,$2,$3,$4) RETURNING id`, no, in.FactoryID, userID, in.EstimatedDepartureDate).Scan(&id)
	} else {
		var confirmed bool
		err = tx.QueryRow(ctx, `SELECT confirmed_at IS NOT NULL FROM delivery_plan WHERE id=$1 AND ($2 OR factory_id=ANY($3)) FOR UPDATE`, id, sc.All, sc.FactoryIDs).Scan(&confirmed)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		if err == nil && confirmed {
			return nil, nil, ErrConflict
		}
	}
	if err != nil {
		return nil, nil, mapErr(err)
	}
	_, err = tx.Exec(ctx, `UPDATE delivery_plan SET factory_id=$2,logistics_company=$3,contact=$4,phone=$5,shipping_address=$6,receiving_address=$7,car_model=$8,plate_no=$9,driver=$10,driver_phone=$11,cost=$12,extra_cost=$13,estimated_departure_date=$14,remark=$15 WHERE id=$1`, id, in.FactoryID, in.LogisticsCompany, in.Contact, in.Phone, in.ShippingAddress, in.ReceivingAddress, in.CarModel, in.PlateNo, in.Driver, in.DriverPhone, in.Cost, in.ExtraCost, in.EstimatedDepartureDate, in.Remark)
	if err != nil {
		return nil, nil, err
	}
	if err = assignPlanNotes(ctx, tx, id, in.FactoryID, in.NoteIDs); err != nil {
		return nil, nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return s.GetDeliveryPlan(ctx, sc, id)
}
func (s *Store) ConfirmDeliveryPlan(ctx context.Context, sc auth.Scope, id, userID int64, date time.Time) (*DeliveryPlan, []DeliveryNote, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var confirmed bool
	err = tx.QueryRow(ctx, `SELECT confirmed_at IS NOT NULL FROM delivery_plan WHERE id=$1 AND ($2 OR factory_id=ANY($3)) FOR UPDATE`, id, sc.All, sc.FactoryIDs).Scan(&confirmed)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	if confirmed {
		return nil, nil, ErrConflict
	}
	var count int
	err = tx.QueryRow(ctx, `SELECT count(*) FROM delivery_note WHERE plan_id=$1`, id).Scan(&count)
	if err != nil {
		return nil, nil, err
	}
	if count == 0 {
		return nil, nil, ErrBadRef
	}
	_, err = tx.Exec(ctx, `UPDATE delivery_plan SET confirmed_departure_date=$2,confirmed_by=$3,confirmed_at=now() WHERE id=$1`, id, date, userID)
	if err != nil {
		return nil, nil, err
	}
	_, err = tx.Exec(ctx, `UPDATE delivery_note d SET logistics_company=CASE WHEN d.logistics_company='' THEN p.logistics_company ELSE d.logistics_company END,plate_no=CASE WHEN d.plate_no='' THEN p.plate_no ELSE d.plate_no END,driver=CASE WHEN d.driver='' THEN p.driver ELSE d.driver END,driver_phone=CASE WHEN d.driver_phone='' THEN p.driver_phone ELSE d.driver_phone END FROM delivery_plan p WHERE d.plan_id=p.id AND p.id=$1`, id)
	if err != nil {
		return nil, nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return s.GetDeliveryPlan(ctx, sc, id)
}
func (s *Store) DeleteDeliveryPlan(ctx context.Context, sc auth.Scope, id int64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var confirmed bool
	err = tx.QueryRow(ctx, `SELECT confirmed_at IS NOT NULL FROM delivery_plan WHERE id=$1 AND ($2 OR factory_id=ANY($3)) FOR UPDATE`, id, sc.All, sc.FactoryIDs).Scan(&confirmed)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if confirmed {
		return ErrConflict
	}
	if _, err = tx.Exec(ctx, `UPDATE delivery_note SET plan_id=NULL WHERE plan_id=$1`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM delivery_plan WHERE id=$1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
