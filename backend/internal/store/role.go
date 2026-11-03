package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type Role struct {
	ID                 int64
	Code, Name, Remark string
	IsStop             bool
	MenuIDs            []int64
	UserCount          int
	CreatedAt          time.Time
}
type RoleInput struct {
	Code, Name, Remark string
	IsStop             bool
	MenuIDs            []int64
}

const roleSelect = `SELECT r.id,r.code,r.name,r.remark,r.is_stop,COALESCE((SELECT array_agg(menu_id ORDER BY menu_id) FROM role_menu WHERE role_id=r.id),ARRAY[]::bigint[]), (SELECT count(*)::int FROM user_role WHERE role_id=r.id),r.created_at FROM role r WHERE NOT r.is_personal`

func scanRole(row pgx.Row) (*Role, error) {
	r := &Role{}
	err := row.Scan(&r.ID, &r.Code, &r.Name, &r.Remark, &r.IsStop, &r.MenuIDs, &r.UserCount, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return r, err
}
func (s *Store) Roles(ctx context.Context) ([]Role, error) {
	rows, err := s.db.Query(ctx, roleSelect+` ORDER BY r.id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Role, error) {
		r, e := scanRole(row)
		if e != nil {
			return Role{}, e
		}
		return *r, nil
	})
}
func (s *Store) SaveRole(ctx context.Context, id *int64, in RoleInput) (*Role, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var roleID int64
	if id == nil {
		err = tx.QueryRow(ctx, `INSERT INTO role(code,name,remark,is_stop) VALUES($1,$2,$3,$4) RETURNING id`, in.Code, in.Name, in.Remark, in.IsStop).Scan(&roleID)
	} else {
		err = tx.QueryRow(ctx, `UPDATE role SET code=$2,name=$3,remark=$4,is_stop=$5 WHERE id=$1 AND NOT is_personal RETURNING id`, *id, in.Code, in.Name, in.Remark, in.IsStop).Scan(&roleID)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, mapErr(err)
	}
	var invalid int
	err = tx.QueryRow(ctx, `SELECT count(*) FROM unnest($1::bigint[]) AS v(id) WHERE NOT EXISTS (SELECT 1 FROM menu WHERE id=v.id)`, in.MenuIDs).Scan(&invalid)
	if err != nil {
		return nil, err
	}
	if invalid > 0 {
		return nil, ErrBadRef
	}
	if _, err = tx.Exec(ctx, `DELETE FROM role_menu WHERE role_id=$1`, roleID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO role_menu(role_id,menu_id) SELECT $1,unnest($2::bigint[]) ON CONFLICT DO NOTHING`, roleID, in.MenuIDs); err != nil {
		return nil, mapErr(err)
	}
	r, err := scanRole(tx.QueryRow(ctx, roleSelect+` AND r.id=$1`, roleID))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r, nil
}

type RoleOccupiedError struct{ Count int }

func (e *RoleOccupiedError) Error() string {
	return fmt.Sprintf("角色仍分配给 %d 个用户，请先移除或改为停用", e.Count)
}
func (s *Store) DeleteRole(ctx context.Context, id int64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var count int
	err = tx.QueryRow(ctx, `SELECT (SELECT count(*)::int FROM user_role WHERE role_id=r.id) FROM role r WHERE id=$1 AND NOT is_personal FOR UPDATE`, id).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if count > 0 {
		return &RoleOccupiedError{Count: count}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM role_menu WHERE role_id=$1`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM role WHERE id=$1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
