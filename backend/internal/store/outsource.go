package store

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

type OutsourceProgress struct {
	ScannedQty             int
	ScannedArea            float64
	UnscannedQty           int
	UnscannedArea, Percent float64
}
type OutsourceOrder struct {
	BatchID                                         int64
	LotNo, ProjectName, CustomerName, ClientOrderNo string
	CreatedAt                                       time.Time
	Thickness                                       float64
	Color, SprayingType                             string
	OutsourceFactoryID                              int64
	OutsourceFactoryName, Status                    string
	ItemQty                                         int
	ItemArea                                        float64
	CuttingQty                                      int
	CuttingArea                                     float64
	ShippingQty                                     int
	ShippingArea                                    float64
}

func progress(qty int, area float64, scanned int, scannedArea float64) OutsourceProgress {
	return OutsourceProgress{ScannedQty: scanned, ScannedArea: scannedArea, UnscannedQty: max(0, qty-scanned), UnscannedArea: math.Max(0, area-scannedArea), Percent: func() float64 {
		if qty == 0 {
			return 0
		}
		return math.Min(100, float64(scanned)*100/float64(qty))
	}()}
}
func (o OutsourceOrder) Cutting() OutsourceProgress {
	return progress(o.ItemQty, o.ItemArea, o.CuttingQty, o.CuttingArea)
}
func (o OutsourceOrder) Shipping() OutsourceProgress {
	return progress(o.ItemQty, o.ItemArea, o.ShippingQty, o.ShippingArea)
}

type OutsourceFilter struct {
	From, To  *time.Time
	FactoryID *int64
	Keyword   string
}
type OutsourceSummary struct {
	Batches, PlannedQty                   int
	PlannedArea                           float64
	CuttingScannedQty, ShippingScannedQty int
}

const outsourceFrom = ` FROM batch b JOIN contract c ON c.id=b.contract_id JOIN customer cu ON cu.id=c.customer_id JOIN project p ON p.id=c.project_id JOIN factory f ON f.id=b.outsource_factory_id
 LEFT JOIN LATERAL (SELECT count(*) FILTER (WHERE j.code='10')::int cut_qty,coalesce(sum(bi.piece_area) FILTER (WHERE j.code='10'),0)::float8 cut_area,
 count(*) FILTER (WHERE j.code='20')::int ship_qty,coalesce(sum(bi.piece_area) FILTER (WHERE j.code='20'),0)::float8 ship_area
 FROM scan_record sr JOIN job j ON j.id=sr.job_id JOIN label l ON l.id=sr.label_id AND NOT l.cancelled JOIN batch_item bi ON bi.id=l.batch_item_id
 WHERE sr.batch_id=b.id AND j.code IN ('10','20')) scans ON true`
const outsourceSelect = `SELECT b.id,b.lot_no,p.full_name,cu.name,b.client_order_no,b.created_at,b.thickness::float8,b.color,b.spraying_type,b.outsource_factory_id,f.name,b.status,b.item_qty,b.item_area::float8,scans.cut_qty,scans.cut_area,scans.ship_qty,scans.ship_area` + outsourceFrom

func (s *Store) OutsourceOrders(ctx context.Context, sc auth.Scope, f OutsourceFilter, page, size int) ([]OutsourceOrder, OutsourceSummary, error) {
	where := ` WHERE b.outsource_factory_id IS NOT NULL AND ($1 OR b.factory_id=ANY($2) OR b.outsource_factory_id=ANY($2))`
	args := []any{sc.All, sc.FactoryIDs}
	add := func(q string, v any) { args = append(args, v); where += fmt.Sprintf(q, len(args)) }
	if f.From != nil {
		add(` AND b.created_at >= $%d`, *f.From)
	}
	if f.To != nil {
		add(` AND b.created_at < $%d`, f.To.AddDate(0, 0, 1))
	}
	if f.FactoryID != nil {
		add(` AND b.outsource_factory_id=$%d`, *f.FactoryID)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		add(` AND (b.lot_no ILIKE $%[1]d OR p.full_name ILIKE $%[1]d OR cu.name ILIKE $%[1]d OR b.client_order_no ILIKE $%[1]d)`, "%"+escapeLike(kw)+"%")
	}
	var sum OutsourceSummary
	err := s.db.QueryRow(ctx, `SELECT count(*)::int,coalesce(sum(b.item_qty),0)::int,coalesce(sum(b.item_area),0)::float8,coalesce(sum(scans.cut_qty),0)::int,coalesce(sum(scans.ship_qty),0)::int`+outsourceFrom+where, args...).Scan(&sum.Batches, &sum.PlannedQty, &sum.PlannedArea, &sum.CuttingScannedQty, &sum.ShippingScannedQty)
	if err != nil {
		return nil, sum, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, outsourceSelect+where+fmt.Sprintf(` ORDER BY b.created_at DESC,b.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, sum, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[OutsourceOrder])
	return items, sum, err
}
