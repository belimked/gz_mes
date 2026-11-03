package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNotScanned      = errors.New("not scanned")
	ErrLabelInHandover = errors.New("label in handover")
	ErrLabelDelivered  = errors.New("label delivered")
	ErrNoteAccounted   = errors.New("note accounted")
)

func nextDocNo(ctx context.Context, tx pgx.Tx, prefix, kind string) (string, error) {
	var day time.Time
	var n int
	err := tx.QueryRow(ctx, `INSERT INTO doc_seq(kind,day,n) VALUES($1,CURRENT_DATE,1) ON CONFLICT(kind,day) DO UPDATE SET n=doc_seq.n+1 RETURNING day,n`, kind).Scan(&day, &n)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s%04d", prefix, day.Format("20060102"), n), nil
}

type documentLabel struct {
	ID      int64
	BarCode string
	Area    float64
}

func uniqueCodes(codes []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(codes))
	for _, v := range codes {
		v = strings.TrimSpace(v)
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
func (s *Store) documentLabels(ctx context.Context, tx pgx.Tx, batchID int64, codes []string) ([]documentLabel, float64, error) {
	codes = uniqueCodes(codes)
	if len(codes) == 0 {
		return nil, 0, ErrBadRef
	}
	rows, err := tx.Query(ctx, `SELECT l.id,l.bar_code,bi.piece_area::float8 FROM label l JOIN batch_item bi ON bi.id=l.batch_item_id WHERE l.bar_code=ANY($1) AND l.batch_id=$2 AND NOT l.cancelled ORDER BY l.id`, codes, batchID)
	if err != nil {
		return nil, 0, err
	}
	labels, err := pgx.CollectRows(rows, pgx.RowToStructByPos[documentLabel])
	if err != nil {
		return nil, 0, err
	}
	if len(labels) != len(codes) {
		return nil, 0, fmt.Errorf("%w: 条码不存在/不属于该批次/已作废", ErrBadRef)
	}
	var area float64
	for _, l := range labels {
		area += l.Area
	}
	return labels, area, nil
}
func (s *Store) documentScans(ctx context.Context, tx pgx.Tx, labels []documentLabel, jobID int64, message string) error {
	for _, l := range labels {
		var ok bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM scan_record WHERE label_id=$1 AND job_id=$2)`, l.ID, jobID).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: %s%s", ErrNotScanned, message, l.BarCode)
		}
	}
	return nil
}
func (s *Store) docLabels(ctx context.Context, table, col string, id int64) ([]Label, error) {
	rows, err := s.db.Query(ctx, labelSelect+fmt.Sprintf(` JOIN %s dl ON dl.label_id=l.id WHERE dl.%s=$1 ORDER BY l.id`, table, col), id)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Label])
}

// updateBatchProgress is called while the batch row is locked by the caller.
func updateBatchProgress(ctx context.Context, tx pgx.Tx, batchID, jobID int64) error {
	_, err := tx.Exec(ctx, `UPDATE batch SET status=CASE WHEN (SELECT seq FROM job WHERE id=$2)=(SELECT max(seq) FROM job) AND (SELECT count(*) FROM label WHERE batch_id=$1 AND NOT cancelled)=(SELECT count(DISTINCT sr.label_id) FROM scan_record sr JOIN job j ON j.id=sr.job_id JOIN label l ON l.id=sr.label_id WHERE sr.batch_id=$1 AND NOT l.cancelled AND j.seq=(SELECT max(seq) FROM job)) THEN 'done' WHEN status IN ('pending','scheduled') THEN 'in_production' ELSE status END,updated_at=now() WHERE id=$1`, batchID, jobID)
	return err
}

type Handover struct {
	ID                 int64
	HandoverNo         string
	FactoryID, BatchID int64
	LotNo, ProjectName string
	JobID              int64
	JobCode, JobName   string
	Qty                int
	Area               float64
	Remark, PackedBy   string
	PackedAt           time.Time
	Received           bool
	ReceivedBy         *string
	ReceivedAt         *time.Time
}

const handoverFrom = ` FROM handover h JOIN batch b ON b.id=h.batch_id JOIN contract c ON c.id=b.contract_id JOIN project p ON p.id=c.project_id JOIN job j ON j.id=h.job_id JOIN app_user u ON u.id=h.packed_by LEFT JOIN app_user ru ON ru.id=h.received_by`
const handoverSelect = `SELECT h.id,h.handover_no,h.factory_id,h.batch_id,b.lot_no,p.full_name,h.job_id,j.code,j.name,h.qty,h.area::float8,h.remark,u.display_name,h.packed_at,h.received,ru.display_name,h.received_at` + handoverFrom

func (s *Store) GetHandover(ctx context.Context, sc auth.Scope, id int64) (*Handover, []Label, error) {
	rows, err := s.db.Query(ctx, handoverSelect+` WHERE h.id=$1 AND ($2 OR h.factory_id=ANY($3))`, id, sc.All, sc.FactoryIDs)
	if err != nil {
		return nil, nil, err
	}
	v, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Handover])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	labels, err := s.docLabels(ctx, "handover_label", "handover_id", id)
	return &v, labels, err
}

type HandoverFilter struct {
	BatchID, JobID *int64
	Received       *bool
	Keyword        string
}

func (s *Store) ListHandovers(ctx context.Context, sc auth.Scope, f HandoverFilter, page, size int) ([]Handover, int64, error) {
	where := ` WHERE ($1 OR h.factory_id=ANY($2))`
	args := []any{sc.All, sc.FactoryIDs}
	add := func(q string, v any) { args = append(args, v); where += fmt.Sprintf(q, len(args)) }
	if f.BatchID != nil {
		add(` AND h.batch_id=$%d`, *f.BatchID)
	}
	if f.JobID != nil {
		add(` AND h.job_id=$%d`, *f.JobID)
	}
	if f.Received != nil {
		add(` AND h.received=$%d`, *f.Received)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		add(` AND (h.handover_no ILIKE $%[1]d OR b.lot_no ILIKE $%[1]d OR p.full_name ILIKE $%[1]d)`, "%"+escapeLike(kw)+"%")
	}
	var total int64
	if err := s.db.QueryRow(ctx, `SELECT count(*)`+handoverFrom+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, handoverSelect+where+fmt.Sprintf(` ORDER BY h.packed_at DESC,h.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Handover])
	return items, total, err
}
func (s *Store) CreateHandover(ctx context.Context, sc auth.Scope, batchID int64, jobCode string, codes []string, remark string, userID int64) (*Handover, []Label, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var factoryID int64
	err = tx.QueryRow(ctx, `SELECT factory_id FROM batch WHERE id=$1 AND ($2 OR factory_id=ANY($3)) FOR UPDATE`, batchID, sc.All, sc.FactoryIDs).Scan(&factoryID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	var jobID int64
	err = tx.QueryRow(ctx, `SELECT id FROM job WHERE code=$1`, jobCode).Scan(&jobID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrBadRef
	}
	if err != nil {
		return nil, nil, err
	}
	labels, area, err := s.documentLabels(ctx, tx, batchID, codes)
	if err != nil {
		return nil, nil, err
	}
	if err = s.documentScans(ctx, tx, labels, jobID, "有标签尚未在该工序扫描: "); err != nil {
		return nil, nil, err
	}
	no, err := nextDocNo(ctx, tx, "JJ", "handover")
	if err != nil {
		return nil, nil, err
	}
	var id int64
	err = tx.QueryRow(ctx, `INSERT INTO handover(handover_no,factory_id,batch_id,job_id,qty,area,remark,packed_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, no, factoryID, batchID, jobID, len(labels), area, remark, userID).Scan(&id)
	if err != nil {
		return nil, nil, err
	}
	for _, l := range labels {
		tag, e := tx.Exec(ctx, `INSERT INTO handover_label(handover_id,label_id,job_id) VALUES($1,$2,$3) ON CONFLICT (label_id,job_id) DO NOTHING`, id, l.ID, jobID)
		if e != nil {
			return nil, nil, e
		}
		if tag.RowsAffected() == 0 {
			return nil, nil, fmt.Errorf("%w: 标签已在其他交接单: %s", ErrLabelInHandover, l.BarCode)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return s.GetHandover(ctx, sc, id)
}
func (s *Store) ReceiveHandover(ctx context.Context, sc auth.Scope, id, userID int64) (*Handover, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var received bool
	err = tx.QueryRow(ctx, `SELECT received FROM handover WHERE id=$1 AND ($2 OR factory_id=ANY($3)) FOR UPDATE`, id, sc.All, sc.FactoryIDs).Scan(&received)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if received {
		return nil, ErrConflict
	}
	if _, err = tx.Exec(ctx, `UPDATE handover SET received=true,received_by=$2,received_at=now() WHERE id=$1`, id, userID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	v, _, err := s.GetHandover(ctx, sc, id)
	return v, err
}

type DeliveryNote struct {
	ID                                             int64
	NoteNo                                         string
	FactoryID, BatchID                             int64
	LotNo, ProjectName                             string
	CustomerID                                     int64
	CustomerName, ContractNo, DeliveryBatch        string
	Qty                                            int
	Area                                           float64
	LogisticsCompany, PlateNo, Driver, DriverPhone string
	DeliveredAt                                    time.Time
	Remark                                         string
	Accounted                                      bool
	AccountID                                      *int64
	PlanID                                         *int64
	PlanNo                                         *string
	CreatedBy                                      string
	CreatedAt                                      time.Time
}

const noteFrom = ` FROM delivery_note d JOIN batch b ON b.id=d.batch_id JOIN contract c ON c.id=b.contract_id JOIN project p ON p.id=c.project_id JOIN customer cu ON cu.id=d.customer_id JOIN app_user u ON u.id=d.created_by LEFT JOIN delivery_plan dp ON dp.id=d.plan_id`
const noteSelect = `SELECT d.id,d.note_no,d.factory_id,d.batch_id,b.lot_no,p.full_name,d.customer_id,cu.name,c.contract_no,d.delivery_batch,d.qty,d.area::float8,d.logistics_company,d.plate_no,d.driver,d.driver_phone,d.delivered_at,d.remark,(d.account_id IS NOT NULL),d.account_id,d.plan_id,dp.plan_no,u.display_name,d.created_at` + noteFrom

func (s *Store) GetDeliveryNote(ctx context.Context, sc auth.Scope, id int64) (*DeliveryNote, []Label, error) {
	rows, err := s.db.Query(ctx, noteSelect+` WHERE d.id=$1 AND ($2 OR d.factory_id=ANY($3))`, id, sc.All, sc.FactoryIDs)
	if err != nil {
		return nil, nil, err
	}
	v, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[DeliveryNote])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	labels, err := s.docLabels(ctx, "delivery_note_label", "note_id", id)
	return &v, labels, err
}

type DeliveryNoteFilter struct {
	BatchID, CustomerID, FactoryID *int64
	Unplanned                      *bool
	Keyword                        string
	Accounted                      *bool
	Begin, End                     *time.Time
}

func (s *Store) ListDeliveryNotes(ctx context.Context, sc auth.Scope, f DeliveryNoteFilter, page, size int) ([]DeliveryNote, int64, error) {
	where := ` WHERE ($1 OR d.factory_id=ANY($2))`
	args := []any{sc.All, sc.FactoryIDs}
	add := func(q string, v any) { args = append(args, v); where += fmt.Sprintf(q, len(args)) }
	if f.BatchID != nil {
		add(` AND d.batch_id=$%d`, *f.BatchID)
	}
	if f.CustomerID != nil {
		add(` AND d.customer_id=$%d`, *f.CustomerID)
	}
	if f.Accounted != nil {
		add(` AND (d.account_id IS NOT NULL)=$%d`, *f.Accounted)
	}
	if f.FactoryID != nil {
		add(` AND d.factory_id=$%d`, *f.FactoryID)
	}
	if f.Unplanned != nil {
		add(` AND (d.plan_id IS NULL)=$%d`, *f.Unplanned)
	}
	if f.Begin != nil {
		add(` AND d.delivered_at >= $%d`, *f.Begin)
	}
	if f.End != nil {
		add(` AND d.delivered_at <= $%d`, *f.End)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		add(` AND (d.note_no ILIKE $%[1]d OR b.lot_no ILIKE $%[1]d OR p.full_name ILIKE $%[1]d OR d.plate_no ILIKE $%[1]d)`, "%"+escapeLike(kw)+"%")
	}
	var total int64
	if err := s.db.QueryRow(ctx, `SELECT count(*)`+noteFrom+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, noteSelect+where+fmt.Sprintf(` ORDER BY d.created_at DESC,d.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[DeliveryNote])
	return items, total, err
}

type DeliveryNoteInput struct {
	BatchID                                                               int64
	BarCodes                                                              []string
	DeliveryBatch, LogisticsCompany, PlateNo, Driver, DriverPhone, Remark string
	DeliveredAt                                                           time.Time
}

func (s *Store) CreateDeliveryNote(ctx context.Context, sc auth.Scope, in DeliveryNoteInput, userID int64) (*DeliveryNote, []Label, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var factoryID, customerID int64
	err = tx.QueryRow(ctx, `SELECT b.factory_id,c.customer_id FROM batch b JOIN contract c ON c.id=b.contract_id WHERE b.id=$1 AND ($2 OR b.factory_id=ANY($3)) FOR UPDATE OF b`, in.BatchID, sc.All, sc.FactoryIDs).Scan(&factoryID, &customerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	labels, area, err := s.documentLabels(ctx, tx, in.BatchID, in.BarCodes)
	if err != nil {
		return nil, nil, err
	}
	var job55, job60 int64
	if err = tx.QueryRow(ctx, `SELECT id FROM job WHERE code='55'`).Scan(&job55); err != nil {
		return nil, nil, err
	}
	if err = tx.QueryRow(ctx, `SELECT id FROM job WHERE code='60'`).Scan(&job60); err != nil {
		return nil, nil, err
	}
	if err = s.documentScans(ctx, tx, labels, job55, "有标签尚未完成成品装架: "); err != nil {
		return nil, nil, err
	}
	no, err := nextDocNo(ctx, tx, "FH", "delivery")
	if err != nil {
		return nil, nil, err
	}
	var id int64
	err = tx.QueryRow(ctx, `INSERT INTO delivery_note(note_no,factory_id,batch_id,customer_id,delivery_batch,qty,area,logistics_company,plate_no,driver,driver_phone,delivered_at,remark,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) RETURNING id`, no, factoryID, in.BatchID, customerID, in.DeliveryBatch, len(labels), area, in.LogisticsCompany, in.PlateNo, in.Driver, in.DriverPhone, in.DeliveredAt, in.Remark, userID).Scan(&id)
	if err != nil {
		return nil, nil, err
	}
	for _, l := range labels {
		tag, e := tx.Exec(ctx, `INSERT INTO delivery_note_label(note_id,label_id) VALUES($1,$2) ON CONFLICT (label_id) DO NOTHING`, id, l.ID)
		if e != nil {
			return nil, nil, e
		}
		if tag.RowsAffected() == 0 {
			return nil, nil, fmt.Errorf("%w: 标签已发货: %s", ErrLabelDelivered, l.BarCode)
		}
		if _, e = tx.Exec(ctx, `INSERT INTO scan_record(label_id,batch_id,job_id,scanned_by,device) VALUES($1,$2,$3,$4,'delivery') ON CONFLICT(label_id,job_id) DO NOTHING`, l.ID, in.BatchID, job60, userID); e != nil {
			return nil, nil, e
		}
	}
	if err = updateBatchProgress(ctx, tx, in.BatchID, job60); err != nil {
		return nil, nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return s.GetDeliveryNote(ctx, sc, id)
}

type DeliverAccount struct {
	ID                                int64
	AccountNo                         string
	FactoryID, CustomerID             int64
	CustomerName                      string
	NoteCount, TotalQty               int
	TotalArea, UnitPrice, ExtraAmount float64
	ExtraRemark                       string
	Amount                            float64
	Status, Remark, CreatedBy         string
	CreatedAt                         time.Time
	ConfirmedBy                       *string
	ConfirmedAt                       *time.Time
}

const accountFrom = ` FROM deliver_account a JOIN customer cu ON cu.id=a.customer_id JOIN app_user u ON u.id=a.created_by LEFT JOIN app_user conf ON conf.id=a.confirmed_by`
const accountSelect = `SELECT a.id,a.account_no,a.factory_id,a.customer_id,cu.name,a.note_count,a.total_qty,a.total_area::float8,a.unit_price::float8,a.extra_amount::float8,a.extra_remark,a.amount::float8,a.status,a.remark,u.display_name,a.created_at,conf.display_name,a.confirmed_at` + accountFrom

func (s *Store) GetDeliverAccount(ctx context.Context, sc auth.Scope, id int64) (*DeliverAccount, []DeliveryNote, error) {
	rows, err := s.db.Query(ctx, accountSelect+` WHERE a.id=$1 AND ($2 OR a.factory_id=ANY($3))`, id, sc.All, sc.FactoryIDs)
	if err != nil {
		return nil, nil, err
	}
	v, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[DeliverAccount])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	rows, err = s.db.Query(ctx, noteSelect+` WHERE d.account_id=$1 ORDER BY d.id`, id)
	if err != nil {
		return nil, nil, err
	}
	notes, err := pgx.CollectRows(rows, pgx.RowToStructByPos[DeliveryNote])
	return &v, notes, err
}

type DeliverAccountFilter struct {
	CustomerID      *int64
	Status, Keyword string
}

func (s *Store) ListDeliverAccounts(ctx context.Context, sc auth.Scope, f DeliverAccountFilter, page, size int) ([]DeliverAccount, int64, error) {
	where := ` WHERE ($1 OR a.factory_id=ANY($2))`
	args := []any{sc.All, sc.FactoryIDs}
	add := func(q string, v any) { args = append(args, v); where += fmt.Sprintf(q, len(args)) }
	if f.CustomerID != nil {
		add(` AND a.customer_id=$%d`, *f.CustomerID)
	}
	if f.Status != "" {
		add(` AND a.status=$%d`, f.Status)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		add(` AND (a.account_no ILIKE $%[1]d OR cu.name ILIKE $%[1]d)`, "%"+escapeLike(kw)+"%")
	}
	var total int64
	if err := s.db.QueryRow(ctx, `SELECT count(*)`+accountFrom+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, accountSelect+where+fmt.Sprintf(` ORDER BY a.created_at DESC,a.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[DeliverAccount])
	return items, total, err
}

type DeliverAccountInput struct {
	CustomerID             int64
	NoteIDs                []int64
	UnitPrice, ExtraAmount float64
	ExtraRemark, Remark    string
}

func (s *Store) CreateDeliverAccount(ctx context.Context, sc auth.Scope, in DeliverAccountInput, userID int64) (*DeliverAccount, []DeliveryNote, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var factoryID int64
	err = tx.QueryRow(ctx, `SELECT factory_id FROM customer WHERE id=$1 AND ($2 OR factory_id=ANY($3))`, in.CustomerID, sc.All, sc.FactoryIDs).Scan(&factoryID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrBadRef
	}
	if err != nil {
		return nil, nil, err
	}
	seen := map[int64]bool{}
	ids := make([]int64, 0, len(in.NoteIDs))
	for _, id := range in.NoteIDs {
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	if len(ids) == 0 {
		return nil, nil, ErrBadRef
	}
	rows, err := tx.Query(ctx, `SELECT id,note_no,customer_id,account_id,qty,area::float8 FROM delivery_note WHERE id=ANY($1) AND ($2 OR factory_id=ANY($3)) ORDER BY id FOR UPDATE`, ids, sc.All, sc.FactoryIDs)
	if err != nil {
		return nil, nil, err
	}
	type note struct {
		ID         int64
		No         string
		CustomerID int64
		AccountID  *int64
		Qty        int
		Area       float64
	}
	notes, err := pgx.CollectRows(rows, pgx.RowToStructByPos[note])
	if err != nil {
		return nil, nil, err
	}
	if len(notes) != len(ids) {
		return nil, nil, fmt.Errorf("%w: 发货单不存在", ErrBadRef)
	}
	var qty int
	var area float64
	for _, n := range notes {
		if n.CustomerID != in.CustomerID {
			return nil, nil, fmt.Errorf("%w: 发货单不属于该客户", ErrBadRef)
		}
		if n.AccountID != nil {
			return nil, nil, fmt.Errorf("%w: 发货单已对账: %s", ErrNoteAccounted, n.No)
		}
		qty += n.Qty
		area += n.Area
	}
	no, err := nextDocNo(ctx, tx, "DZ", "account")
	if err != nil {
		return nil, nil, err
	}
	amount := math.Round((area*in.UnitPrice+in.ExtraAmount)*100) / 100
	var id int64
	err = tx.QueryRow(ctx, `INSERT INTO deliver_account(account_no,factory_id,customer_id,note_count,total_qty,total_area,unit_price,extra_amount,extra_remark,amount,remark,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`, no, factoryID, in.CustomerID, len(notes), qty, area, in.UnitPrice, in.ExtraAmount, in.ExtraRemark, amount, in.Remark, userID).Scan(&id)
	if err != nil {
		return nil, nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE delivery_note SET account_id=$1 WHERE id=ANY($2)`, id, ids); err != nil {
		return nil, nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return s.GetDeliverAccount(ctx, sc, id)
}
func (s *Store) ConfirmDeliverAccount(ctx context.Context, sc auth.Scope, id, userID int64) (*DeliverAccount, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM deliver_account WHERE id=$1 AND ($2 OR factory_id=ANY($3)) FOR UPDATE`, id, sc.All, sc.FactoryIDs).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status == "confirmed" {
		return nil, ErrConflict
	}
	if _, err = tx.Exec(ctx, `UPDATE deliver_account SET status='confirmed',confirmed_by=$2,confirmed_at=now() WHERE id=$1`, id, userID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	v, _, err := s.GetDeliverAccount(ctx, sc, id)
	return v, err
}
