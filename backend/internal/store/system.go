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

// AllMenus returns every menu as a flat list in display order.
func (s *Store) AllMenus(ctx context.Context) ([]*Menu, error) {
	rows, err := s.db.Query(ctx, "SELECT id, parent_id, title, path, icon FROM menu ORDER BY sort, id")
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Menu, error) {
		m := &Menu{}
		err := row.Scan(&m.ID, &m.ParentID, &m.Title, &m.Path, &m.Icon)
		return m, err
	})
}

type UserDetail struct {
	ID          int64
	Username    string
	DisplayName string
	IsAdmin     bool
	Enabled     bool
	FactoryIDs  []int64
	MenuIDs     []int64
	CreatedAt   time.Time
}

type UserFilter struct {
	Keyword string
	Enabled *bool
}

type UserCreate struct {
	Username    string
	DisplayName string
	Password    string
	IsAdmin     bool
	FactoryIDs  []int64
	MenuIDs     []int64
}

type UserUpdate struct {
	DisplayName string
	Enabled     bool
	IsAdmin     bool
	FactoryIDs  []int64
	MenuIDs     []int64
}

const detailSelect = `SELECT u.id, u.username, u.display_name, u.is_admin, u.enabled,
 COALESCE((SELECT array_agg(uf.factory_id ORDER BY uf.factory_id) FROM user_factory uf WHERE uf.user_id = u.id), ARRAY[]::bigint[]),
 COALESCE((SELECT array_agg(rm.menu_id ORDER BY rm.menu_id) FROM user_role ur JOIN role r ON r.id = ur.role_id JOIN role_menu rm ON rm.role_id = r.id WHERE ur.user_id = u.id AND r.code = 'user:' || u.username), ARRAY[]::bigint[]),
 u.created_at FROM app_user u`

func scanDetail(row pgx.Row) (*UserDetail, error) {
	d := &UserDetail{}
	err := row.Scan(&d.ID, &d.Username, &d.DisplayName, &d.IsAdmin, &d.Enabled, &d.FactoryIDs, &d.MenuIDs, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return d, err
}

func (s *Store) GetUser(ctx context.Context, id int64) (*UserDetail, error) {
	return scanDetail(s.db.QueryRow(ctx, detailSelect+" WHERE u.id = $1", id))
}

func (s *Store) ListUsers(ctx context.Context, f UserFilter, page, size int) ([]UserDetail, int64, error) {
	where := " WHERE ($1 = '' OR u.username ILIKE $2 OR u.display_name ILIKE $2) AND ($3::boolean IS NULL OR u.enabled = $3)"
	args := []any{strings.TrimSpace(f.Keyword), "%" + escapeLike(strings.TrimSpace(f.Keyword)) + "%", f.Enabled}
	var total int64
	if err := s.db.QueryRow(ctx, "SELECT count(*) FROM app_user u"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Query(ctx, detailSelect+where+" ORDER BY u.id DESC LIMIT $4 OFFSET $5", append(args, size, (page-1)*size)...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (UserDetail, error) {
		d, err := scanDetail(row)
		if err != nil {
			return UserDetail{}, err
		}
		return *d, nil
	})
	return items, total, err
}

// replaceUserGrants validates references and replaces both sets in a transaction.
func replaceUserGrants(ctx context.Context, tx pgx.Tx, id int64, username string, factories, menus []int64) error {
	for _, item := range []struct {
		table string
		ids   []int64
	}{{"factory", factories}, {"menu", menus}} {
		var count int
		if err := tx.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %s WHERE id = ANY($1)", item.table), item.ids).Scan(&count); err != nil {
			return err
		}
		distinct := make(map[int64]bool, len(item.ids))
		for _, v := range item.ids {
			distinct[v] = true
		}
		if count != len(distinct) {
			return ErrBadRef
		}
	}
	if _, err := tx.Exec(ctx, "DELETE FROM user_factory WHERE user_id = $1", id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "INSERT INTO user_factory (user_id, factory_id) SELECT $1, unnest($2::bigint[]) ON CONFLICT DO NOTHING", id, factories); err != nil {
		return mapErr(err)
	}
	var roleID int64
	if err := tx.QueryRow(ctx, "INSERT INTO role (code, name) VALUES ($1, $1) ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name RETURNING id", "user:"+username).Scan(&roleID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "DELETE FROM user_role WHERE user_id = $1", id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "INSERT INTO user_role (user_id, role_id) VALUES ($1, $2)", id, roleID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "DELETE FROM role_menu WHERE role_id = $1", roleID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, "INSERT INTO role_menu (role_id, menu_id) SELECT $1, unnest($2::bigint[]) ON CONFLICT DO NOTHING", roleID, menus)
	return mapErr(err)
}

func (s *Store) CreateUserFull(ctx context.Context, in UserCreate) (*UserDetail, error) {
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var id int64
	err = tx.QueryRow(ctx, "INSERT INTO app_user (username, display_name, password_hash, is_admin) VALUES ($1,$2,$3,$4) RETURNING id", in.Username, in.DisplayName, hash, in.IsAdmin).Scan(&id)
	if err != nil {
		return nil, mapErr(err)
	}
	if err := replaceUserGrants(ctx, tx, id, in.Username, in.FactoryIDs, in.MenuIDs); err != nil {
		return nil, err
	}
	d, err := scanDetail(tx.QueryRow(ctx, detailSelect+" WHERE u.id = $1", id))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Store) UpdateUser(ctx context.Context, id int64, in UserUpdate) (*UserDetail, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var username string
	err = tx.QueryRow(ctx, "SELECT username FROM app_user WHERE id = $1 FOR UPDATE", id).Scan(&username)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, "UPDATE app_user SET display_name=$2, enabled=$3, is_admin=$4, updated_at=now() WHERE id=$1", id, in.DisplayName, in.Enabled, in.IsAdmin); err != nil {
		return nil, err
	}
	if err := replaceUserGrants(ctx, tx, id, username, in.FactoryIDs, in.MenuIDs); err != nil {
		return nil, err
	}
	d, err := scanDetail(tx.QueryRow(ctx, detailSelect+" WHERE u.id = $1", id))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Store) ResetPassword(ctx context.Context, id int64, password string) error {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx, "UPDATE app_user SET password_hash=$2, updated_at=now() WHERE id=$1", id, hash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
