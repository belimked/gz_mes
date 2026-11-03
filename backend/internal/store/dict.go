package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type DictItem struct {
	ID                 int64
	Kind, Name         string
	Sort               int
	StartTime, EndTime *string
	IsStop             bool
	Remark             string
	CreatedAt          time.Time
}

type DictInput struct {
	Kind, Name         string
	Sort               int
	StartTime, EndTime *string
	IsStop             bool
	Remark             string
}

const dictSelect = `SELECT id,kind,name,sort,to_char(start_time,'HH24:MI'),to_char(end_time,'HH24:MI'),is_stop,remark,created_at FROM dict_item`

func (s *Store) Dicts(ctx context.Context, kind string, stopped bool) ([]DictItem, error) {
	rows, err := s.db.Query(ctx, dictSelect+` WHERE ($1='' OR kind=$1) AND ($2 OR NOT is_stop) ORDER BY kind,sort,id`, kind, stopped)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[DictItem])
}

// CheckDictValue allows an unchanged legacy value even if it has since been stopped.
func CheckDictValue(ctx context.Context, tx pgx.Tx, kind, value, old string) error {
	if value == "" || value == old {
		return nil
	}
	var ok bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM dict_item WHERE kind=$1 AND name=$2 AND NOT is_stop)`, kind, value).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		label := map[string]string{"workshop": "车间", "paint_line": "喷涂线", "shift": "班次"}[kind]
		return invalidPlan(fmt.Sprintf("%s「%s」不在启用的基础字典中", label, value))
	}
	return nil
}

func (s *Store) SaveDict(ctx context.Context, id *int64, in DictInput) (*DictItem, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var itemID int64
	if id == nil {
		err = tx.QueryRow(ctx, `INSERT INTO dict_item(kind,name,sort,start_time,end_time,is_stop,remark) VALUES($1,$2,$3,$4::time,$5::time,$6,$7) RETURNING id`, in.Kind, in.Name, in.Sort, in.StartTime, in.EndTime, in.IsStop, in.Remark).Scan(&itemID)
	} else {
		var kind, old string
		err = tx.QueryRow(ctx, `SELECT kind,name FROM dict_item WHERE id=$1 FOR UPDATE`, *id).Scan(&kind, &old)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		if kind != in.Kind {
			return nil, ErrBadRef
		}
		err = tx.QueryRow(ctx, `UPDATE dict_item SET name=$2,sort=$3,start_time=$4::time,end_time=$5::time,is_stop=$6,remark=$7 WHERE id=$1 RETURNING id`, *id, in.Name, in.Sort, in.StartTime, in.EndTime, in.IsStop, in.Remark).Scan(&itemID)
		if err == nil && old != in.Name {
			switch kind {
			case "workshop":
				_, err = tx.Exec(ctx, `UPDATE batch SET workshop=$1 WHERE workshop=$2`, in.Name, old)
				if err == nil {
					_, err = tx.Exec(ctx, `UPDATE plan SET workshop=$1 WHERE workshop=$2`, in.Name, old)
				}
			case "paint_line":
				_, err = tx.Exec(ctx, `UPDATE plan SET paint_line=$1 WHERE paint_line=$2`, in.Name, old)
			case "shift":
				_, err = tx.Exec(ctx, `UPDATE plan SET shift=$1 WHERE shift=$2`, in.Name, old)
			}
		}
	}
	if err != nil {
		return nil, mapErr(err)
	}
	rows, err := tx.Query(ctx, dictSelect+` WHERE id=$1`, itemID)
	if err != nil {
		return nil, err
	}
	item, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[DictItem])
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &item, nil
}
