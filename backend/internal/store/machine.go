package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

type Machine struct {
	ID, FactoryID, JobID             int64
	Code, Name, FactoryName, JobName string
	JobSeq, CapacityQty              int
	CapacityArea                     float64
	IsStop                           bool
	Remark                           string
	CreatedAt                        time.Time
}

type MachineInput struct {
	FactoryID, JobID   int64
	Code, Name, Remark string
	CapacityQty        int
	CapacityArea       float64
	IsStop             bool
}

const machineSelect = `SELECT m.id,m.factory_id,m.job_id,m.code,m.name,f.name,j.name,j.seq,m.capacity_qty,m.capacity_area::float8,m.is_stop,m.remark,m.created_at FROM machine m JOIN factory f ON f.id=m.factory_id JOIN job j ON j.id=m.job_id`

func (s *Store) OwnFactoryInScope(ctx context.Context, sc auth.Scope, factoryID int64) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM factory WHERE id=$1 AND kind='own' AND ($2 OR id=ANY($3)))`, factoryID, sc.All, sc.FactoryIDs).Scan(&ok)
	return ok, err
}

func (s *Store) Machines(ctx context.Context, sc auth.Scope, factoryID, jobID *int64, stopped bool) ([]Machine, error) {
	rows, err := s.db.Query(ctx, machineSelect+` WHERE ($1 OR m.factory_id=ANY($2)) AND ($3::bigint IS NULL OR m.factory_id=$3) AND ($4::bigint IS NULL OR m.job_id=$4) AND ($5 OR NOT m.is_stop) ORDER BY f.name,j.seq,m.code`, sc.All, sc.FactoryIDs, factoryID, jobID, stopped)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Machine])
}

func (s *Store) SaveMachine(ctx context.Context, sc auth.Scope, id *int64, in MachineInput) (*Machine, error) {
	var valid bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM factory WHERE id=$1 AND kind='own' AND ($2 OR id=ANY($3))) AND EXISTS(SELECT 1 FROM job WHERE id=$4)`, in.FactoryID, sc.All, sc.FactoryIDs, in.JobID).Scan(&valid)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, ErrBadRef
	}
	var machineID int64
	if id == nil {
		err = s.db.QueryRow(ctx, `INSERT INTO machine(factory_id,job_id,code,name,capacity_qty,capacity_area,is_stop,remark) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, in.FactoryID, in.JobID, in.Code, in.Name, in.CapacityQty, in.CapacityArea, in.IsStop, in.Remark).Scan(&machineID)
	} else {
		err = s.db.QueryRow(ctx, `UPDATE machine SET factory_id=$1,job_id=$2,code=$3,name=$4,capacity_qty=$5,capacity_area=$6,is_stop=$7,remark=$8 WHERE id=$9 AND ($10 OR factory_id=ANY($11)) RETURNING id`, in.FactoryID, in.JobID, in.Code, in.Name, in.CapacityQty, in.CapacityArea, in.IsStop, in.Remark, *id, sc.All, sc.FactoryIDs).Scan(&machineID)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, mapErr(err)
	}
	rows, err := s.db.Query(ctx, machineSelect+` WHERE m.id=$1`, machineID)
	if err != nil {
		return nil, err
	}
	m, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Machine])
	return &m, err
}

type MachineOutputFilter struct {
	From, To                    time.Time
	FactoryID, MachineID, JobID *int64
	Keyword, Mode               string
}
type MachineOutputRow struct {
	MachineID                                   int64
	Code, Name, FactoryName, JobName            string
	BatchID                                     *int64
	LotNo, ProjectName                          *string
	Qty                                         int
	PieceArea, SprayingArea                     float64
	Days                                        int
	CapacityQty                                 *int
	CapacityArea, QtyLoad, AreaLoad, LoadFactor *float64
}
type MachineOutputSummary struct {
	Machines, Qty           int
	PieceArea, SprayingArea float64
	UnspecifiedQty          int
}

func (s *Store) MachineOutput(ctx context.Context, sc auth.Scope, f MachineOutputFilter, page, size int) ([]MachineOutputRow, int64, MachineOutputSummary, error) {
	summary := MachineOutputSummary{}
	args := []any{sc.All, sc.FactoryIDs, f.From, f.To.AddDate(0, 0, 1), f.FactoryID, f.MachineID, f.JobID, "%" + escapeLike(strings.TrimSpace(f.Keyword)) + "%", f.To.Sub(f.From).Hours()/24 + 1}
	// The CTE includes enabled idle machines in summary mode, but only scanned combinations in detail mode.
	base := ` WITH selected AS (SELECT m.id,m.code,m.name,f.name AS factory_name,j.name AS job_name,j.seq,m.capacity_qty,m.capacity_area FROM machine m JOIN factory f ON f.id=m.factory_id JOIN job j ON j.id=m.job_id WHERE ($1 OR m.factory_id=ANY($2)) AND ($5::bigint IS NULL OR m.factory_id=$5) AND ($6::bigint IS NULL OR m.id=$6) AND ($7::bigint IS NULL OR m.job_id=$7) AND NOT m.is_stop AND (`
	if f.Mode == "detail" {
		base += `m.code ILIKE $8 OR m.name ILIKE $8 OR EXISTS(SELECT 1 FROM scan_record sr JOIN batch b ON b.id=sr.batch_id JOIN contract c ON c.id=b.contract_id JOIN project p ON p.id=c.project_id WHERE sr.machine_id=m.id AND sr.scanned_at >= $3 AND sr.scanned_at < $4 AND (b.lot_no ILIKE $8 OR p.full_name ILIKE $8))`
	} else {
		base += `m.code ILIKE $8 OR m.name ILIKE $8`
	}
	base += `)` // close predicate
	base += `), grouped AS (SELECT m.id,m.code,m.name,m.factory_name,m.job_name,m.seq,`
	if f.Mode == "detail" {
		base += `b.id AS batch_id,b.lot_no,p.full_name AS project_name,`
	} else {
		base += `NULL::bigint AS batch_id,NULL::text AS lot_no,NULL::text AS project_name,`
	}
	base += `count(sr.id)::int AS qty,coalesce(sum(bi.piece_area),0)::float8 AS piece_area,coalesce(sum(bi.spraying_area),0)::float8 AS spraying_area,($9::float8)::int AS days,`
	if f.Mode == "detail" {
		base += `NULL::int AS capacity_qty,NULL::float8 AS capacity_area,NULL::float8 AS qty_load,NULL::float8 AS area_load,NULL::float8 AS load_factor`
	} else {
		base += `(m.capacity_qty*$9)::int AS capacity_qty,(m.capacity_area*$9)::float8 AS capacity_area,CASE WHEN m.capacity_qty>0 THEN count(sr.id)*100.0/(m.capacity_qty*$9) END::float8 AS qty_load,CASE WHEN m.capacity_area>0 THEN coalesce(sum(bi.piece_area),0)*100.0/(m.capacity_area*$9) END::float8 AS area_load,CASE WHEN m.capacity_area>0 THEN coalesce(sum(bi.piece_area),0)*100.0/(m.capacity_area*$9) WHEN m.capacity_qty>0 THEN count(sr.id)*100.0/(m.capacity_qty*$9) END::float8 AS load_factor`
	}
	base += ` FROM selected m LEFT JOIN scan_record sr ON sr.machine_id=m.id AND sr.scanned_at >= $3 AND sr.scanned_at < $4 LEFT JOIN label l ON l.id=sr.label_id AND NOT l.cancelled LEFT JOIN batch_item bi ON bi.id=l.batch_item_id`
	// Cancelled labels must not count, so exclude them in the join itself.
	base = strings.Replace(base, `sr.machine_id=m.id AND sr.scanned_at >= $3`, `sr.machine_id=m.id AND EXISTS(SELECT 1 FROM label active WHERE active.id=sr.label_id AND NOT active.cancelled) AND sr.scanned_at >= $3`, 1)
	if f.Mode == "detail" {
		base += ` LEFT JOIN batch b ON b.id=sr.batch_id LEFT JOIN contract c ON c.id=b.contract_id LEFT JOIN project p ON p.id=c.project_id WHERE (m.code ILIKE $8 OR m.name ILIKE $8 OR b.lot_no ILIKE $8 OR p.full_name ILIKE $8) GROUP BY m.id,m.code,m.name,m.factory_name,m.job_name,m.seq,m.capacity_qty,m.capacity_area,b.id,p.full_name HAVING count(sr.id)>0`
	} else {
		base += ` GROUP BY m.id,m.code,m.name,m.factory_name,m.job_name,m.seq,m.capacity_qty,m.capacity_area`
	}
	base += `)`
	err := s.db.QueryRow(ctx, base+` SELECT count(DISTINCT id),coalesce(sum(qty),0)::int,coalesce(sum(piece_area),0)::float8,coalesce(sum(spraying_area),0)::float8 FROM grouped`, args...).Scan(&summary.Machines, &summary.Qty, &summary.PieceArea, &summary.SprayingArea)
	if err != nil {
		return nil, 0, summary, err
	}
	err = s.db.QueryRow(ctx, `SELECT count(*) FROM scan_record sr JOIN label l ON l.id=sr.label_id WHERE sr.machine_id IS NULL AND NOT l.cancelled AND sr.scanned_at >= $3 AND sr.scanned_at < $4 AND ($1 OR l.factory_id=ANY($2)) AND ($5::bigint IS NULL OR l.factory_id=$5) AND ($6::bigint IS NULL) AND ($7::bigint IS NULL OR sr.job_id=$7)`, args[:7]...).Scan(&summary.UnspecifiedQty)
	if err != nil {
		return nil, 0, summary, err
	}
	var total int64
	err = s.db.QueryRow(ctx, base+` SELECT count(*) FROM grouped`, args...).Scan(&total)
	if err != nil {
		return nil, 0, summary, err
	}
	order := `seq,code`
	if f.Mode == "detail" {
		order = `code,lot_no`
	}
	rows, err := s.db.Query(ctx, base+` SELECT id,code,name,factory_name,job_name,batch_id,lot_no,project_name,qty,piece_area,spraying_area,days,capacity_qty,capacity_area,qty_load,area_load,load_factor FROM grouped ORDER BY `+order+` LIMIT $10 OFFSET $11`, append(args, size, (page-1)*size)...)
	if err != nil {
		return nil, 0, summary, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[MachineOutputRow])
	return items, total, summary, err
}
