package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

type Drawing struct {
	ID, BatchID                                        int64
	Kind, DrawingNo, Remark, OriginalName, ContentType string
	SizeBytes                                          int64
	SHA256, StorageKey                                 string
	UploadedBy                                         int64
	UploaderName                                       string
	UploadedAt                                         time.Time
}
type DrawingBatch struct {
	ID                                      int64
	LotNo, ProjectName, CustomerName        string
	CreatedAt                               time.Time
	UnfoldCount, FormingCount, NestingCount int
	LastUploadedAt                          *time.Time
}
type DrawingFilter struct {
	Keyword, State string
	Begin, End     *time.Time
}

const drawingSelect = `SELECT d.id,d.batch_id,d.kind,d.drawing_no,d.remark,d.original_name,d.content_type,d.size_bytes,d.sha256,d.storage_key,d.uploaded_by,u.display_name,d.uploaded_at FROM drawing d JOIN app_user u ON u.id=d.uploaded_by`

func (s *Store) BatchInScope(ctx context.Context, sc auth.Scope, id int64) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM batch WHERE id=$1 AND ($2 OR factory_id=ANY($3)))`, id, sc.All, sc.FactoryIDs).Scan(&ok)
	return ok, err
}
func (s *Store) Drawings(ctx context.Context, sc auth.Scope, batchID int64) ([]Drawing, error) {
	ok, e := s.BatchInScope(ctx, sc, batchID)
	if e != nil {
		return nil, e
	}
	if !ok {
		return nil, ErrNotFound
	}
	rows, e := s.db.Query(ctx, drawingSelect+` WHERE d.batch_id=$1 ORDER BY d.uploaded_at DESC,d.id DESC`, batchID)
	if e != nil {
		return nil, e
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Drawing])
}
func (s *Store) Drawing(ctx context.Context, sc auth.Scope, id int64) (*Drawing, error) {
	rows, e := s.db.Query(ctx, drawingSelect+` JOIN batch b ON b.id=d.batch_id WHERE d.id=$1 AND ($2 OR b.factory_id=ANY($3))`, id, sc.All, sc.FactoryIDs)
	if e != nil {
		return nil, e
	}
	v, e := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Drawing])
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &v, e
}
func (s *Store) AddDrawing(ctx context.Context, sc auth.Scope, d Drawing) (*Drawing, error) {
	rows, e := s.db.Query(ctx, `INSERT INTO drawing(batch_id,kind,drawing_no,remark,original_name,content_type,size_bytes,sha256,storage_key,uploaded_by) SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10 WHERE EXISTS(SELECT 1 FROM batch WHERE id=$1 AND ($11 OR factory_id=ANY($12))) RETURNING id`, d.BatchID, d.Kind, d.DrawingNo, d.Remark, d.OriginalName, d.ContentType, d.SizeBytes, d.SHA256, d.StorageKey, d.UploadedBy, sc.All, sc.FactoryIDs)
	if e != nil {
		return nil, e
	}
	id, e := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int64])
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if e != nil {
		return nil, e
	}
	return s.Drawing(ctx, sc, id)
}
func (s *Store) RemoveDrawing(ctx context.Context, sc auth.Scope, id int64) (string, error) {
	var key string
	e := s.db.QueryRow(ctx, `DELETE FROM drawing d USING batch b WHERE d.id=$1 AND b.id=d.batch_id AND ($2 OR b.factory_id=ANY($3)) RETURNING d.storage_key`, id, sc.All, sc.FactoryIDs).Scan(&key)
	if errors.Is(e, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return key, e
}
func (s *Store) DrawingBatches(ctx context.Context, sc auth.Scope, f DrawingFilter, page, size int) ([]DrawingBatch, int64, error) {
	args := []any{sc.All, sc.FactoryIDs, "%" + escapeLike(strings.TrimSpace(f.Keyword)) + "%", f.Begin, f.End, f.State}
	q := ` FROM batch b JOIN contract c ON c.id=b.contract_id JOIN project p ON p.id=c.project_id JOIN customer cu ON cu.id=c.customer_id LEFT JOIN LATERAL (SELECT count(*) FILTER(WHERE kind='unfold')::int AS unfold_count,count(*) FILTER(WHERE kind='forming')::int AS forming_count,count(*) FILTER(WHERE kind='nesting')::int AS nesting_count,max(uploaded_at) AS last_uploaded_at FROM drawing WHERE batch_id=b.id) d ON true WHERE ($1 OR b.factory_id=ANY($2)) AND (b.lot_no ILIKE $3 OR p.full_name ILIKE $3 OR c.contract_no ILIKE $3) AND ($4::date IS NULL OR b.created_at >= $4::date) AND ($5::date IS NULL OR b.created_at < $5::date + interval '1 day') AND (NULLIF($6::text,'') IS NULL OR ($6='none' AND d.unfold_count+d.forming_count+d.nesting_count=0) OR ($6='complete' AND d.unfold_count>0 AND d.forming_count>0) OR ($6='partial' AND d.unfold_count+d.forming_count+d.nesting_count>0 AND (d.unfold_count=0 OR d.forming_count=0)))`
	var total int64
	e := s.db.QueryRow(ctx, `SELECT count(*)`+q, args...).Scan(&total)
	if e != nil {
		return nil, 0, e
	}
	rows, e := s.db.Query(ctx, `SELECT b.id,b.lot_no,p.full_name,cu.name,b.created_at,d.unfold_count,d.forming_count,d.nesting_count,d.last_uploaded_at`+q+` ORDER BY b.created_at DESC,b.id DESC LIMIT $7 OFFSET $8`, append(args, size, (page-1)*size)...)
	if e != nil {
		return nil, 0, e
	}
	items, e := pgx.CollectRows(rows, pgx.RowToStructByPos[DrawingBatch])
	return items, total, e
}
