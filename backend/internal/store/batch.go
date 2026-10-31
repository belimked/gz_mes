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

var ErrDuplicateItem = errors.New("duplicate item")
var ErrHasLabels = errors.New("batch has labels")

type Job struct {
	ID         int64
	Code, Name string
	Seq        int
	Stage      string
	IsQc       bool
}

func (s *Store) Jobs(ctx context.Context) ([]Job, error) {
	rows, err := s.db.Query(ctx, "SELECT id, code, name, seq, stage, is_qc FROM job ORDER BY seq")
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Job])
}

type Batch struct {
	ID, FactoryID, ContractID                                                                                        int64
	ContractNo                                                                                                       string
	CustomerID                                                                                                       int64
	CustomerName                                                                                                     string
	ProjectID                                                                                                        int64
	ProjectName, LotNo, ClientOrderNo                                                                                string
	Thickness                                                                                                        float64
	Color, ColorNo, Texture, SprayingType                                                                            string
	OrderQty                                                                                                         int
	OrderArea                                                                                                        float64
	DeliveryAddress, SpecialRequest                                                                                  string
	RequiredDate                                                                                                     *time.Time
	Urgency, Status                                                                                                  string
	ItemCount, ItemQty                                                                                               int
	ItemArea                                                                                                         float64
	PlanTechDate, PlanDrawingDate, PlanCutDate, PlanSheetStart, PlanSheetDelivery, PlanSprayStart, PlanSprayDelivery *time.Time
	OutsourceFactoryID                                                                                               *int64
	OutsourceFactoryName                                                                                             *string
	Workshop, Remark, CreatedBy                                                                                      string
	CreatedAt, UpdatedAt                                                                                             time.Time
}

const batchFrom = ` FROM batch b JOIN contract c ON c.id=b.contract_id JOIN customer cu ON cu.id=c.customer_id JOIN project p ON p.id=c.project_id JOIN app_user u ON u.id=b.created_by LEFT JOIN factory f ON f.id=b.outsource_factory_id`
const batchSelect = `SELECT b.id,b.factory_id,b.contract_id,c.contract_no,c.customer_id,cu.name,c.project_id,p.full_name,b.lot_no,b.client_order_no,b.thickness::float8,b.color,b.color_no,b.texture,b.spraying_type,b.order_qty,b.order_area::float8,b.delivery_address,b.special_request,b.required_date,b.urgency,b.status,b.item_count,b.item_qty,b.item_area::float8,b.plan_tech_date,b.plan_drawing_date,b.plan_cut_date,b.plan_sheet_start,b.plan_sheet_delivery,b.plan_spray_start,b.plan_spray_delivery,b.outsource_factory_id,f.name,b.workshop,b.remark,u.display_name,b.created_at,b.updated_at` + batchFrom

type BatchFilter struct {
	Keyword                           string
	ContractID, ProjectID, CustomerID *int64
	Status, Urgency                   string
	Begin, End                        *time.Time
}

func (s *Store) ListBatches(ctx context.Context, sc auth.Scope, f BatchFilter, page, size int) ([]Batch, int64, error) {
	where := " WHERE ($1 OR b.factory_id=ANY($2))"
	args := []any{sc.All, sc.FactoryIDs}
	add := func(clause string, value any) { args = append(args, value); where += fmt.Sprintf(clause, len(args)) }
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		add(" AND (b.lot_no ILIKE $%[1]d OR b.client_order_no ILIKE $%[1]d OR c.contract_no ILIKE $%[1]d OR cu.name ILIKE $%[1]d OR p.full_name ILIKE $%[1]d)", "%"+escapeLike(kw)+"%")
	}
	if f.ContractID != nil {
		add(" AND b.contract_id=$%d", *f.ContractID)
	}
	if f.ProjectID != nil {
		add(" AND c.project_id=$%d", *f.ProjectID)
	}
	if f.CustomerID != nil {
		add(" AND c.customer_id=$%d", *f.CustomerID)
	}
	if f.Status != "" {
		add(" AND b.status=$%d", f.Status)
	}
	if f.Urgency != "" {
		add(" AND b.urgency=$%d", f.Urgency)
	}
	if f.Begin != nil {
		add(" AND b.required_date >= $%d", *f.Begin)
	}
	if f.End != nil {
		add(" AND b.required_date <= $%d", *f.End)
	}
	var total int64
	if err := s.db.QueryRow(ctx, "SELECT count(*)"+batchFrom+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, batchSelect+where+fmt.Sprintf(" ORDER BY b.created_at DESC, b.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Batch])
	return items, total, err
}
func (s *Store) GetBatch(ctx context.Context, sc auth.Scope, id int64) (*Batch, error) {
	rows, err := s.db.Query(ctx, batchSelect+" WHERE b.id=$1 AND ($2 OR b.factory_id=ANY($3))", id, sc.All, sc.FactoryIDs)
	if err != nil {
		return nil, err
	}
	b, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Batch])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &b, err
}

type BatchCreate struct {
	ContractID                            int64
	LotNo, ClientOrderNo                  string
	Thickness                             float64
	Color, ColorNo, Texture, SprayingType string
	OrderQty                              int
	OrderArea                             float64
	DeliveryAddress, SpecialRequest       string
	RequiredDate                          *time.Time
	Urgency, Remark                       string
	CreatedBy                             int64
}

func (s *Store) CreateBatch(ctx context.Context, sc auth.Scope, in BatchCreate) (*Batch, error) {
	var id int64
	err := s.db.QueryRow(ctx, `INSERT INTO batch (factory_id,contract_id,lot_no,client_order_no,thickness,color,color_no,texture,spraying_type,order_qty,order_area,delivery_address,special_request,required_date,urgency,remark,created_by)
 SELECT c.factory_id,c.id,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16 FROM contract c WHERE c.id=$1 AND ($17 OR c.factory_id=ANY($18)) RETURNING id`, in.ContractID, in.LotNo, in.ClientOrderNo, in.Thickness, in.Color, in.ColorNo, in.Texture, in.SprayingType, in.OrderQty, in.OrderArea, in.DeliveryAddress, in.SpecialRequest, in.RequiredDate, in.Urgency, in.Remark, in.CreatedBy, sc.All, sc.FactoryIDs).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, mapErr(err)
	}
	return s.GetBatch(ctx, sc, id)
}

// Optional distinguishes an omitted update from an explicit SQL NULL.
type Optional[T any] struct{ Value T }
type BatchSchedule struct {
	PlanTechDate, PlanDrawingDate, PlanCutDate, PlanSheetStart, PlanSheetDelivery, PlanSprayStart, PlanSprayDelivery *Optional[*time.Time]
	OutsourceFactoryID                                                                                               *Optional[*int64]
	Workshop, Remark, Urgency                                                                                        *Optional[string]
}

func (s *Store) ScheduleBatch(ctx context.Context, sc auth.Scope, id int64, in BatchSchedule) (*Batch, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	var sheet *time.Time
	err = tx.QueryRow(ctx, "SELECT status,plan_sheet_start FROM batch WHERE id=$1 AND ($2 OR factory_id=ANY($3)) FOR UPDATE", id, sc.All, sc.FactoryIDs).Scan(&status, &sheet)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status == "done" {
		return nil, ErrConflict
	}
	if in.OutsourceFactoryID != nil && in.OutsourceFactoryID.Value != nil {
		var valid bool
		if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM factory WHERE id=$1 AND kind='outsource')", *in.OutsourceFactoryID.Value).Scan(&valid); err != nil {
			return nil, err
		}
		if !valid {
			return nil, ErrBadRef
		}
	}
	sets := []string{"updated_at=now()"}
	args := []any{id}
	add := func(col string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s=$%d", col, len(args)))
	}
	for _, v := range []struct {
		col  string
		date *Optional[*time.Time]
	}{{"plan_tech_date", in.PlanTechDate}, {"plan_drawing_date", in.PlanDrawingDate}, {"plan_cut_date", in.PlanCutDate}, {"plan_sheet_start", in.PlanSheetStart}, {"plan_sheet_delivery", in.PlanSheetDelivery}, {"plan_spray_start", in.PlanSprayStart}, {"plan_spray_delivery", in.PlanSprayDelivery}} {
		if v.date != nil {
			add(v.col, v.date.Value)
		}
	}
	if in.OutsourceFactoryID != nil {
		add("outsource_factory_id", in.OutsourceFactoryID.Value)
	}
	if in.Workshop != nil {
		add("workshop", in.Workshop.Value)
	}
	if in.Remark != nil {
		add("remark", in.Remark.Value)
	}
	if in.Urgency != nil {
		add("urgency", in.Urgency.Value)
	}
	if in.PlanSheetStart != nil {
		sheet = in.PlanSheetStart.Value
	}
	if status == "pending" && sheet != nil {
		sets = append(sets, "status='scheduled'")
	}
	if _, err = tx.Exec(ctx, "UPDATE batch SET "+strings.Join(sets, ",")+" WHERE id=$1", args...); err != nil {
		return nil, mapErr(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetBatch(ctx, sc, id)
}

type BatchItem struct {
	ID, BatchID                            int64
	Seq                                    int
	PieceNo, DrawingNo, Pattern            string
	Thickness                              float64
	Color                                  string
	Length, Width, PieceArea, SprayingArea float64
	Qty, PackageQty                        int
	WeldLine                               float64
	BendingTimes                           int
	Remark                                 string
}
type BatchItemInput struct {
	PieceNo, DrawingNo, Pattern            string
	Thickness                              float64
	Color                                  string
	Length, Width, PieceArea, SprayingArea float64
	Qty, PackageQty                        int
	WeldLine                               float64
	BendingTimes                           int
	Remark                                 string
}

const itemSelect = `SELECT id,batch_id,seq,piece_no,drawing_no,pattern,thickness::float8,color,length::float8,width::float8,piece_area::float8,spraying_area::float8,qty,package_qty,weld_line::float8,bending_times,remark FROM batch_item WHERE batch_id=$1 ORDER BY seq`

func (s *Store) ListBatchItems(ctx context.Context, sc auth.Scope, id int64) ([]BatchItem, error) {
	if _, err := s.GetBatch(ctx, sc, id); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, itemSelect, id)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[BatchItem])
}
func (s *Store) ReplaceBatchItems(ctx context.Context, sc auth.Scope, id int64, items []BatchItemInput) ([]BatchItem, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	err = tx.QueryRow(ctx, "SELECT status FROM batch WHERE id=$1 AND ($2 OR factory_id=ANY($3)) FOR UPDATE", id, sc.All, sc.FactoryIDs).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status == "done" {
		return nil, ErrConflict
	}
	var hasLabels bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM label WHERE batch_id=$1 AND NOT cancelled)", id).Scan(&hasLabels); err != nil {
		return nil, err
	}
	if hasLabels {
		return nil, ErrHasLabels
	}
	seen := make(map[string]bool, len(items))
	rows := make([][]any, 0, len(items))
	for i, v := range items {
		if seen[v.PieceNo] {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateItem, v.PieceNo)
		}
		seen[v.PieceNo] = true
		rows = append(rows, []any{id, i + 1, v.PieceNo, v.DrawingNo, v.Pattern, v.Thickness, v.Color, v.Length, v.Width, v.PieceArea, v.SprayingArea, v.Qty, v.PackageQty, v.WeldLine, v.BendingTimes, v.Remark})
	}
	if _, err = tx.Exec(ctx, "DELETE FROM batch_item WHERE batch_id=$1", id); err != nil {
		return nil, err
	}
	if len(rows) > 0 {
		_, err = tx.CopyFrom(ctx, pgx.Identifier{"batch_item"}, []string{"batch_id", "seq", "piece_no", "drawing_no", "pattern", "thickness", "color", "length", "width", "piece_area", "spraying_area", "qty", "package_qty", "weld_line", "bending_times", "remark"}, pgx.CopyFromRows(rows))
		if err != nil {
			return nil, mapErr(err)
		}
	}
	_, err = tx.Exec(ctx, `UPDATE batch SET item_count=(SELECT count(*) FROM batch_item WHERE batch_id=$1), item_qty=(SELECT coalesce(sum(qty),0) FROM batch_item WHERE batch_id=$1), item_area=(SELECT coalesce(sum(piece_area*qty),0) FROM batch_item WHERE batch_id=$1), updated_at=now() WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.ListBatchItems(ctx, sc, id)
}
