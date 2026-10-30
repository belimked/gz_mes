package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/db"
	"github.com/belimked/gz_mes/backend/internal/server"
	"github.com/belimked/gz_mes/backend/internal/store"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		if os.Getenv("CI") != "" {
			fmt.Fprintln(os.Stderr, "TEST_DATABASE_URL is required in CI")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "skipping integration tests: TEST_DATABASE_URL not set")
		os.Exit(0)
	}
	ctx := context.Background()
	var err error
	pool, err = db.Open(ctx, url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// Verify migrations apply from scratch and roll back cleanly.
	for _, step := range []func(context.Context, *pgxpool.Pool) error{db.Migrate, db.Reset, db.Migrate} {
		if err := step(ctx, pool); err != nil {
			fmt.Fprintln(os.Stderr, "migrate:", err)
			os.Exit(1)
		}
	}
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

type env struct {
	t     *testing.T
	srv   *httptest.Server
	st    *store.Store
	plant int64 // own factory
	outA  int64 // outsource factory A
	outB  int64 // outsource factory B
}

func newEnv(t *testing.T) *env {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, "TRUNCATE project, user_role, role_menu, role, user_factory, app_user, factory RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatal(err)
	}
	st := store.New(pool)
	e := &env{t: t, st: st}
	e.plant = must(st.CreateFactory(ctx, "F01", "本厂", "own"))
	e.outA = must(st.CreateFactory(ctx, "W01", "外协A", "outsource"))
	e.outB = must(st.CreateFactory(ctx, "W02", "外协B", "outsource"))
	if _, err := st.EnsureAdmin(ctx, "admin", "admin-pass"); err != nil {
		t.Fatal(err)
	}
	h, err := server.New(st, auth.NewTokens("test-secret-test-secret-test-secret!", time.Hour), "test").Handler()
	if err != nil {
		t.Fatal(err)
	}
	e.srv = httptest.NewServer(h)
	t.Cleanup(e.srv.Close)
	return e
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// outsourceUser creates a non-admin user scoped to one factory with sale menus.
func (e *env) outsourceUser(username string, factoryID int64) {
	e.t.Helper()
	ctx := context.Background()
	uid, err := e.st.CreateUser(ctx, username, username, "pw-"+username, false)
	if err != nil {
		e.t.Fatal(err)
	}
	if err := e.st.GrantFactory(ctx, uid, factoryID); err != nil {
		e.t.Fatal(err)
	}
	if err := e.st.GrantMenus(ctx, uid, "sale-viewer", 201, 203); err != nil {
		e.t.Fatal(err)
	}
}

func (e *env) do(method, path, token string, body any, out any) int {
	e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			e.t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, e.srv.URL+"/api"+path, &buf)
	if err != nil {
		e.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			e.t.Fatalf("%s %s: decode: %v", method, path, err)
		}
	}
	return resp.StatusCode
}

func (e *env) login(username, password string) string {
	e.t.Helper()
	var r api.LoginResponse
	if code := e.do("POST", "/auth/login", "", api.LoginRequest{Username: username, Password: password}, &r); code != 200 {
		e.t.Fatalf("login %s: status %d", username, code)
	}
	return r.Token
}

func (e *env) createProject(token string, factoryID int64, code string) (int, api.Project) {
	e.t.Helper()
	var p api.Project
	status := e.do("POST", "/projects", token, api.ProjectCreate{FactoryId: factoryID, Code: code, FullName: "工程 " + code}, &p)
	return status, p
}

func TestHealthIsPublic(t *testing.T) {
	e := newEnv(t)
	var h api.Health
	if code := e.do("GET", "/health", "", nil, &h); code != 200 || h.Status != "ok" {
		t.Fatalf("health: %d %+v", code, h)
	}
}

func TestLogin(t *testing.T) {
	e := newEnv(t)
	var errBody api.Error
	if code := e.do("POST", "/auth/login", "", api.LoginRequest{Username: "admin", Password: "wrong"}, &errBody); code != 401 {
		t.Fatalf("wrong password: want 401, got %d", code)
	}
	if code := e.do("POST", "/auth/login", "", api.LoginRequest{Username: "nobody", Password: "x"}, nil); code != 401 {
		t.Fatalf("unknown user: want 401, got %d", code)
	}
	// Missing required field is rejected by OpenAPI validation.
	if code := e.do("POST", "/auth/login", "", map[string]string{"username": "admin"}, nil); code != 400 {
		t.Fatalf("missing password: want 400, got %d", code)
	}
	tok := e.login("admin", "admin-pass")
	var me api.MeResponse
	if code := e.do("GET", "/auth/me", tok, nil, &me); code != 200 || !me.User.IsAdmin || len(me.Menus) == 0 {
		t.Fatalf("me: %d %+v", code, me)
	}
}

func TestProtectedRoutesNeedToken(t *testing.T) {
	e := newEnv(t)
	for _, path := range []string{"/auth/me", "/factories", "/projects"} {
		if code := e.do("GET", path, "", nil, nil); code != 401 {
			t.Errorf("%s without token: want 401, got %d", path, code)
		}
		if code := e.do("GET", path, "garbage", nil, nil); code != 401 {
			t.Errorf("%s with bad token: want 401, got %d", path, code)
		}
	}
}

func TestMenusFollowRoles(t *testing.T) {
	e := newEnv(t)
	e.outsourceUser("wa", e.outA)
	var me api.MeResponse
	e.do("GET", "/auth/me", e.login("wa", "pw-wa"), nil, &me)
	if len(me.Menus) != 1 || me.Menus[0].Title != "销售管理" || me.Menus[0].Children == nil || len(*me.Menus[0].Children) != 2 {
		t.Fatalf("want only 销售管理 with 2 children, got %+v", me.Menus)
	}
}

func TestFactoryIsolation(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	e.outsourceUser("wa", e.outA)
	e.outsourceUser("wb", e.outB)
	wa, wb := e.login("wa", "pw-wa"), e.login("wb", "pw-wb")

	if code, _ := e.createProject(admin, e.plant, "P-PLANT"); code != 201 {
		t.Fatalf("admin create: %d", code)
	}
	if code, _ := e.createProject(wa, e.outA, "P-A"); code != 201 {
		t.Fatalf("wa create in own factory: %d", code)
	}
	if code, _ := e.createProject(wa, e.outB, "P-A-IN-B"); code != 403 {
		t.Fatalf("wa create in other factory: want 403, got %d", code)
	}

	var fs []api.Factory
	e.do("GET", "/factories", wa, nil, &fs)
	if len(fs) != 1 || fs[0].Id != e.outA {
		t.Fatalf("wa factories: %+v", fs)
	}

	var page api.ProjectPage
	e.do("GET", "/projects", wb, nil, &page)
	if page.Total != 0 {
		t.Fatalf("wb should see no projects, got %d", page.Total)
	}
	e.do("GET", "/projects", wa, nil, &page)
	if page.Total != 1 || page.Items[0].Code != "P-A" {
		t.Fatalf("wa should see only P-A, got %+v", page.Items)
	}
	e.do("GET", "/projects", admin, nil, &page)
	if page.Total != 2 {
		t.Fatalf("admin should see 2, got %d", page.Total)
	}
}

func TestProjectLifecycle(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	e.outsourceUser("wb", e.outB)
	wb := e.login("wb", "pw-wb")

	code, p := e.createProject(admin, e.plant, "P-001")
	if code != 201 || p.Confirmed || p.Kind != api.Offline || p.CreatedBy != "管理员" {
		t.Fatalf("create: %d %+v", code, p)
	}
	if code, _ := e.createProject(admin, e.plant, "P-001"); code != 409 {
		t.Fatalf("duplicate code: want 409, got %d", code)
	}
	// Same code in a different factory is allowed.
	if code, _ := e.createProject(admin, e.outA, "P-001"); code != 201 {
		t.Fatalf("same code other factory: want 201, got %d", code)
	}
	// Blank after trimming is rejected.
	if code := e.do("POST", "/projects", admin, api.ProjectCreate{FactoryId: e.plant, Code: "  ", FullName: "x"}, nil); code != 400 {
		t.Fatalf("blank code: want 400, got %d", code)
	}

	path := fmt.Sprintf("/projects/%d/confirm", p.Id)
	if code := e.do("POST", path, wb, nil, nil); code != 404 {
		t.Fatalf("confirm outside scope: want 404, got %d", code)
	}
	var confirmed api.Project
	if code := e.do("POST", path, admin, nil, &confirmed); code != 200 || !confirmed.Confirmed {
		t.Fatalf("confirm: %d %+v", code, confirmed)
	}
	if code := e.do("POST", path, admin, nil, nil); code != 409 {
		t.Fatalf("confirm twice: want 409, got %d", code)
	}
	if code := e.do("POST", "/projects/999999/confirm", admin, nil, nil); code != 404 {
		t.Fatalf("confirm missing: want 404, got %d", code)
	}
}

func TestListProjectsSearchAndPaging(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	for i := 1; i <= 5; i++ {
		e.createProject(admin, e.plant, fmt.Sprintf("SZ-%02d", i))
	}
	e.createProject(admin, e.plant, "GZ_100%")

	var page api.ProjectPage
	e.do("GET", "/projects?keyword=SZ&page=2&size=2", admin, nil, &page)
	if page.Total != 5 || len(page.Items) != 2 {
		t.Fatalf("paging: total=%d items=%d", page.Total, len(page.Items))
	}
	// LIKE wildcards in the keyword are matched literally.
	e.do("GET", "/projects?keyword=_100%25", admin, nil, &page)
	if page.Total != 1 {
		t.Fatalf("literal wildcard search: want 1, got %d", page.Total)
	}
	if code := e.do("GET", "/projects?size=1000", admin, nil, nil); code != 400 {
		t.Fatalf("size over max: want 400, got %d", code)
	}
}
