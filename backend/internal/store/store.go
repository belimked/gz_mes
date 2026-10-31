// Package store is the PostgreSQL data access layer.
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/belimked/gz_mes/backend/internal/auth"
)

// Sentinel errors mapped to HTTP status codes by the server layer.
var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
	// ErrBadRef means a referenced row (e.g. factory) does not exist.
	ErrBadRef = errors.New("bad reference")
)

// Store wraps a pgx pool.
type Store struct{ db *pgxpool.Pool }

// New returns a Store.
func New(db *pgxpool.Pool) *Store { return &Store{db: db} }

// User is an application user row.
type User struct {
	ID           int64
	Username     string
	DisplayName  string
	StaffNo      string
	PasswordHash string
	IsAdmin      bool
	Enabled      bool
}

const userCols = "id, username, display_name, password_hash, is_admin, enabled, staff_no"

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.IsAdmin, &u.Enabled, &u.StaffNo)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

// UserByUsername looks up a user by login name.
func (s *Store) UserByUsername(ctx context.Context, username string) (*User, error) {
	return scanUser(s.db.QueryRow(ctx, "SELECT "+userCols+" FROM app_user WHERE username = $1", username))
}

// Principal loads an enabled user together with their factory scope.
func (s *Store) Principal(ctx context.Context, userID int64) (*auth.Principal, error) {
	u, err := scanUser(s.db.QueryRow(ctx, "SELECT "+userCols+" FROM app_user WHERE id = $1 AND enabled", userID))
	if err != nil {
		return nil, err
	}
	p := &auth.Principal{UserID: u.ID, Username: u.Username, DisplayName: u.DisplayName, StaffNo: u.StaffNo, IsAdmin: u.IsAdmin}
	if u.IsAdmin {
		p.Scope = auth.Scope{All: true}
		return p, nil
	}
	rows, err := s.db.Query(ctx, "SELECT factory_id FROM user_factory WHERE user_id = $1 ORDER BY factory_id", userID)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return nil, err
	}
	p.Scope = auth.Scope{FactoryIDs: ids}
	return p, nil
}

// CreateUser inserts a user and returns its id.
func (s *Store) CreateUser(ctx context.Context, username, displayName, password string, isAdmin bool) (int64, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.db.QueryRow(ctx,
		"INSERT INTO app_user (username, display_name, password_hash, is_admin) VALUES ($1, $2, $3, $4) RETURNING id",
		username, displayName, hash, isAdmin).Scan(&id)
	return id, mapErr(err)
}

// EnsureAdmin creates the first admin when the user table is empty.
// It returns true if a user was created.
func (s *Store) EnsureAdmin(ctx context.Context, username, password string) (bool, error) {
	var n int
	if err := s.db.QueryRow(ctx, "SELECT count(*) FROM app_user").Scan(&n); err != nil {
		return false, err
	}
	if n > 0 {
		return false, nil
	}
	if password == "" {
		return false, errors.New("no users exist: set ADMIN_PASSWORD to bootstrap the first admin")
	}
	_, err := s.CreateUser(ctx, username, "管理员", password, true)
	return err == nil, err
}

// EnsureDefaultFactory creates an "own" factory when none exists, so a fresh
// install is usable right away. It returns true if one was created.
func (s *Store) EnsureDefaultFactory(ctx context.Context) (bool, error) {
	var n int
	if err := s.db.QueryRow(ctx, "SELECT count(*) FROM factory").Scan(&n); err != nil {
		return false, err
	}
	if n > 0 {
		return false, nil
	}
	_, err := s.CreateFactory(ctx, "F01", "本厂", "own")
	return err == nil, err
}

// Menu is a menu tree node.
type Menu struct {
	ID       int64
	ParentID *int64
	Title    string
	Path     string
	Icon     string
	Children []*Menu
}

// MenusFor returns the menu tree visible to p. Admins see everything;
// others see menus granted through roles plus their ancestors.
func (s *Store) MenusFor(ctx context.Context, p *auth.Principal) ([]*Menu, error) {
	q := "SELECT id, parent_id, title, path, icon FROM menu ORDER BY sort, id"
	args := []any{}
	if !p.IsAdmin {
		q = `WITH RECURSIVE granted AS (
			SELECT m.* FROM menu m
			JOIN role_menu rm ON rm.menu_id = m.id
			JOIN user_role ur ON ur.role_id = rm.role_id
			WHERE ur.user_id = $1
			UNION
			SELECT p.* FROM menu p JOIN granted g ON g.parent_id = p.id
		) SELECT DISTINCT id, parent_id, title, path, icon, sort FROM granted ORDER BY sort, id`
		args = append(args, p.UserID)
	}
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var flat []*Menu
	for rows.Next() {
		m := &Menu{}
		dest := []any{&m.ID, &m.ParentID, &m.Title, &m.Path, &m.Icon}
		if !p.IsAdmin {
			var sort int
			dest = append(dest, &sort)
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		flat = append(flat, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	byID := make(map[int64]*Menu, len(flat))
	for _, m := range flat {
		byID[m.ID] = m
	}
	var roots []*Menu
	for _, m := range flat {
		if m.ParentID != nil {
			if parent, ok := byID[*m.ParentID]; ok {
				parent.Children = append(parent.Children, m)
				continue
			}
		}
		roots = append(roots, m)
	}
	return roots, nil
}

// Factory is a plant or outsource factory.
type Factory struct {
	ID     int64
	Code   string
	Name   string
	Kind   string
	IsStop bool
}

// CreateFactory inserts a factory.
func (s *Store) CreateFactory(ctx context.Context, code, name, kind string) (*Factory, error) {
	f := &Factory{}
	err := s.db.QueryRow(ctx, "INSERT INTO factory (code, name, kind) VALUES ($1, $2, $3) RETURNING id, code, name, kind, is_stop", code, name, kind).Scan(&f.ID, &f.Code, &f.Name, &f.Kind, &f.IsStop)
	return f, mapErr(err)
}

// UpdateFactory changes the mutable factory fields.
func (s *Store) UpdateFactory(ctx context.Context, id int64, name string, isStop bool) (*Factory, error) {
	f := &Factory{}
	err := s.db.QueryRow(ctx, "UPDATE factory SET name = $2, is_stop = $3 WHERE id = $1 RETURNING id, code, name, kind, is_stop", id, name, isStop).Scan(&f.ID, &f.Code, &f.Name, &f.Kind, &f.IsStop)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return f, err
}

// Factories lists factories within scope.
func (s *Store) Factories(ctx context.Context, sc auth.Scope) ([]Factory, error) {
	rows, err := s.db.Query(ctx,
		"SELECT id, code, name, kind, is_stop FROM factory WHERE ($1 OR id = ANY($2)) ORDER BY id",
		sc.All, sc.FactoryIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Factory])
}

// GrantFactory adds factoryID to a user's scope.
func (s *Store) GrantFactory(ctx context.Context, userID, factoryID int64) error {
	_, err := s.db.Exec(ctx, "INSERT INTO user_factory (user_id, factory_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", userID, factoryID)
	return err
}

// GrantMenus creates (or reuses) a role with the given menus and assigns it to the user.
func (s *Store) GrantMenus(ctx context.Context, userID int64, roleCode string, menuIDs ...int64) error {
	var roleID int64
	err := s.db.QueryRow(ctx,
		"INSERT INTO role (code, name) VALUES ($1, $1) ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name RETURNING id",
		roleCode).Scan(&roleID)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(ctx,
		"INSERT INTO role_menu (role_id, menu_id) SELECT $1, unnest($2::bigint[]) ON CONFLICT DO NOTHING", roleID, menuIDs); err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, "INSERT INTO user_role (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", userID, roleID)
	return err
}

// Project row.
type Project struct {
	ID        int64
	FactoryID int64
	Code      string
	FullName  string
	ShortName string
	Address   string
	Kind      string
	Confirmed bool
	CreatedBy string
	CreatedAt time.Time
}

const projectSelect = `SELECT p.id, p.factory_id, p.code, p.full_name, p.short_name, p.address, p.kind, p.confirmed,
	u.display_name, p.created_at FROM project p JOIN app_user u ON u.id = p.created_by`

// ListProjects returns one page of projects within scope.
func (s *Store) ListProjects(ctx context.Context, sc auth.Scope, keyword string, page, size int) ([]Project, int64, error) {
	where := " WHERE ($1 OR p.factory_id = ANY($2))"
	args := []any{sc.All, sc.FactoryIDs}
	if kw := strings.TrimSpace(keyword); kw != "" {
		args = append(args, "%"+escapeLike(kw)+"%")
		where += fmt.Sprintf(" AND (p.code ILIKE $%[1]d OR p.full_name ILIKE $%[1]d OR p.short_name ILIKE $%[1]d)", len(args))
	}
	var total int64
	if err := s.db.QueryRow(ctx, "SELECT count(*) FROM project p"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, size, (page-1)*size)
	q := projectSelect + where + fmt.Sprintf(" ORDER BY p.created_at DESC, p.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Project])
	return items, total, err
}

// ProjectCreate holds new project fields.
type ProjectCreate struct {
	FactoryID int64
	Code      string
	FullName  string
	ShortName string
	Address   string
	Kind      string
	CreatedBy int64
}

// CreateProject inserts a project. Scope must be checked by the caller.
func (s *Store) CreateProject(ctx context.Context, in ProjectCreate) (*Project, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO project (factory_id, code, full_name, short_name, address, kind, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		in.FactoryID, in.Code, in.FullName, in.ShortName, in.Address, in.Kind, in.CreatedBy).Scan(&id)
	if err != nil {
		return nil, mapErr(err)
	}
	return s.getProject(ctx, auth.Scope{All: true}, id)
}

// ConfirmProject locks a project. Returns ErrNotFound outside scope and ErrConflict if already confirmed.
func (s *Store) ConfirmProject(ctx context.Context, sc auth.Scope, id int64) (*Project, error) {
	tag, err := s.db.Exec(ctx,
		"UPDATE project SET confirmed = true WHERE id = $1 AND NOT confirmed AND ($2 OR factory_id = ANY($3))",
		id, sc.All, sc.FactoryIDs)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		if _, err := s.getProject(ctx, sc, id); err != nil {
			return nil, err
		}
		return nil, ErrConflict
	}
	return s.getProject(ctx, sc, id)
}

func (s *Store) getProject(ctx context.Context, sc auth.Scope, id int64) (*Project, error) {
	rows, err := s.db.Query(ctx, projectSelect+" WHERE p.id = $1 AND ($2 OR p.factory_id = ANY($3))", id, sc.All, sc.FactoryIDs)
	if err != nil {
		return nil, err
	}
	p, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Project])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &p, err
}

func escapeLike(s string) string {
	return strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(s)
}

func mapErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrConflict
	}
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return ErrBadRef
	}
	return err
}
