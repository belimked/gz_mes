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
	"github.com/oapi-codegen/runtime/types"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/db"
	"github.com/belimked/gz_mes/backend/internal/server"
	"github.com/belimked/gz_mes/backend/internal/storage"
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
	_, err := pool.Exec(ctx, "TRUNCATE audit_log, delivery_plan, plan, batch_valuation, deliver_account, delivery_note_label, delivery_note, handover_label, handover, doc_seq, scan_record, label, batch_item, batch, contract, customer, project, user_role, role_menu, role, user_factory, app_user, factory RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatal(err)
	}
	st := store.New(pool)
	e := &env{t: t, st: st}
	e.plant = must(st.CreateFactory(ctx, "F01", "本厂", "own")).ID
	e.outA = must(st.CreateFactory(ctx, "W01", "外协A", "outsource")).ID
	e.outB = must(st.CreateFactory(ctx, "W02", "外协B", "outsource")).ID
	if _, err := st.EnsureAdmin(ctx, "admin", "admin-pass"); err != nil {
		t.Fatal(err)
	}
	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	h, err := server.NewWithFiles(st, auth.NewTokens("test-secret-test-secret-test-secret!", time.Hour), "test", files, 50*1024*1024).Handler()
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

func TestCreateProjectUnknownFactory(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	var er api.Error
	if code := e.do("POST", "/projects", admin, api.ProjectCreate{FactoryId: 999999, Code: "X", FullName: "X"}, &er); code != 400 || er.Code != "invalid_factory" {
		t.Fatalf("unknown factory: want 400 invalid_factory, got %d %q", code, er.Code)
	}
}

func (e *env) createCustomer(token string, factoryID int64, name string) (int, api.Customer) {
	e.t.Helper()
	var c api.Customer
	status := e.do("POST", "/customers", token, api.CustomerCreate{FactoryId: factoryID, Name: name}, &c)
	return status, c
}

func TestCustomerLifecycle(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	e.outsourceUser("wa", e.outA)
	wa := e.login("wa", "pw-wa")
	status, c := e.createCustomer(admin, e.plant, "客户 A")
	if status != 201 || c.Confirmed || c.CreatedBy != "管理员" {
		t.Fatalf("create: %d %+v", status, c)
	}
	var er api.Error
	if code := e.do("POST", "/customers", admin, api.CustomerCreate{FactoryId: e.plant, Name: "客户 A"}, &er); code != 409 || er.Code != "duplicate_code" {
		t.Fatalf("duplicate: %d %+v", code, er)
	}
	if code := e.do("POST", "/customers", wa, api.CustomerCreate{FactoryId: e.plant, Name: "wrong"}, &er); code != 403 || er.Code != "forbidden_factory" {
		t.Fatalf("forbidden: %d %+v", code, er)
	}
	if code, _ := e.createCustomer(admin, e.outA, "客户 A"); code != 201 {
		t.Fatalf("same name other factory: %d", code)
	}
	path := fmt.Sprintf("/customers/%d/confirm", c.Id)
	if code := e.do("POST", path, wa, nil, &er); code != 404 || er.Code != "not_found" {
		t.Fatalf("confirm outside scope: %d %+v", code, er)
	}
	var confirmed api.Customer
	if code := e.do("POST", path, admin, nil, &confirmed); code != 200 || !confirmed.Confirmed {
		t.Fatalf("confirm: %d %+v", code, confirmed)
	}
	if code := e.do("POST", path, admin, nil, &er); code != 409 || er.Code != "already_confirmed" {
		t.Fatalf("repeat confirm: %d %+v", code, er)
	}
	var page api.CustomerPage
	if code := e.do("GET", "/customers?keyword=%E5%AE%A2%E6%88%B7", wa, nil, &page); code != 200 || page.Total != 1 || page.Items[0].FactoryId != e.outA {
		t.Fatalf("scoped list: %d %+v", code, page)
	}
}

func TestContractLifecycle(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	e.outsourceUser("wa", e.outA)
	wa := e.login("wa", "pw-wa")
	_, c := e.createCustomer(admin, e.plant, "客户 X")
	_, foreign := e.createCustomer(admin, e.outA, "客户 Y")
	_, p := e.createProject(admin, e.plant, "P-X")
	body := api.ContractCreate{FactoryId: e.plant, ContractNo: "C_001%", CustomerId: c.Id, ProjectId: p.Id}
	var contract api.Contract
	if code := e.do("POST", "/contracts", admin, body, &contract); code != 201 || contract.CustomerName != c.Name || contract.ProjectName != p.FullName || contract.CreatedBy != "管理员" {
		t.Fatalf("create: %d %+v", code, contract)
	}
	var er api.Error
	if code := e.do("POST", "/contracts", admin, body, &er); code != 409 || er.Code != "duplicate_code" {
		t.Fatalf("duplicate: %d %+v", code, er)
	}
	body.ContractNo = "C-foreign"
	body.CustomerId = foreign.Id
	if code := e.do("POST", "/contracts", admin, body, &er); code != 400 || er.Code != "invalid_reference" {
		t.Fatalf("foreign customer: %d %+v", code, er)
	}
	path := fmt.Sprintf("/contracts/%d", contract.Id)
	if code := e.do("GET", path, wa, nil, &er); code != 404 || er.Code != "not_found" {
		t.Fatalf("outside scope: %d %+v", code, er)
	}
	var page api.ContractPage
	date := contract.CreatedAt.Format("2006-01-02")
	for _, query := range []string{"keyword=_001%25", fmt.Sprintf("customerId=%d", c.Id), "beginDate=" + date + "&endDate=" + date} {
		if code := e.do("GET", "/contracts?"+query, admin, nil, &page); code != 200 || page.Total != 1 || len(page.Items) != 1 {
			t.Fatalf("filter %s: %d %+v", query, code, page)
		}
	}
	if code := e.do("GET", "/contracts?customerId="+fmt.Sprint(foreign.Id), admin, nil, &page); code != 200 || page.Total != 0 {
		t.Fatalf("customer filter: %d %+v", code, page)
	}
	if code := e.do("GET", "/contracts", wa, nil, &page); code != 200 || page.Total != 0 {
		t.Fatalf("scoped list: %d %+v", code, page)
	}
	for _, kind := range []api.ValuationAuditKind{"customer", "outsource"} {
		var audited api.Contract
		if code := e.do("POST", path+"/audit-valuation", admin, api.ValuationAudit{Kind: kind}, &audited); code != 200 {
			t.Fatalf("audit %s: %d", kind, code)
		}
		if code := e.do("POST", path+"/audit-valuation", admin, api.ValuationAudit{Kind: kind}, &er); code != 409 || er.Code != "already_audited" {
			t.Fatalf("repeat audit %s: %d %+v", kind, code, er)
		}
	}
	if code := e.do("GET", path, admin, nil, &contract); code != 200 || !contract.CustomerValuationAudited || !contract.OutsourceValuationAudited {
		t.Fatalf("get: %d %+v", code, contract)
	}
}

func TestJobsSeeded(t *testing.T) {
	e := newEnv(t)
	var jobs []api.Job
	if code := e.do("GET", "/jobs", e.login("admin", "admin-pass"), nil, &jobs); code != 200 || len(jobs) != 11 {
		t.Fatalf("jobs: %d %+v", code, jobs)
	}
	for i, j := range jobs {
		if j.Seq != i+1 {
			t.Fatalf("job %d: %+v", i, j)
		}
	}
	if jobs[2].Name != "钣金下线" {
		t.Fatalf("third job: %+v", jobs[2])
	}
}

func (e *env) batchContract(token, suffix string) (api.Customer, api.Project, api.Contract) {
	e.t.Helper()
	_, c := e.createCustomer(token, e.plant, "客户 "+suffix)
	_, p := e.createProject(token, e.plant, "P-"+suffix)
	var contract api.Contract
	if code := e.do("POST", "/contracts", token, api.ContractCreate{FactoryId: e.plant, ContractNo: "C-" + suffix, CustomerId: c.Id, ProjectId: p.Id}, &contract); code != 201 {
		e.t.Fatalf("contract: %d", code)
	}
	return c, p, contract
}
func TestBatchLifecycle(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	c, p, contract := e.batchContract(admin, "batch")
	body := api.BatchCreate{ContractId: contract.Id, LotNo: "201"}
	var batch api.Batch
	if code := e.do("POST", "/batches", admin, body, &batch); code != 201 || batch.FactoryId != e.plant || batch.CustomerName != c.Name || batch.ProjectName != p.FullName || batch.Status != "pending" {
		t.Fatalf("create: %d %+v", code, batch)
	}
	var er api.Error
	if code := e.do("POST", "/batches", admin, body, &er); code != 409 || er.Code != "duplicate_code" {
		t.Fatalf("duplicate: %d %+v", code, er)
	}
	if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: 999999, LotNo: "other"}, &er); code != 404 || er.Code != "not_found" {
		t.Fatalf("missing contract: %d %+v", code, er)
	}
	e.outsourceUser("wa", e.outA)
	wa := e.login("wa", "pw-wa")
	path := fmt.Sprintf("/batches/%d", batch.Id)
	if code := e.do("GET", path, wa, nil, &er); code != 404 {
		t.Fatalf("scope: %d", code)
	}
	if code := e.do("PUT", path+"/items", wa, []api.BatchItemInput{}, &er); code != 404 {
		t.Fatalf("items scope: %d", code)
	}
	if code := e.do("POST", "/batches", wa, api.BatchCreate{ContractId: contract.Id, LotNo: "wa"}, &er); code != 404 {
		t.Fatalf("contract scope: %d", code)
	}
	start := api.BatchSchedule{PlanSheetStart: &apiDateForTest}
	if code := e.do("PUT", path+"/schedule", admin, start, &batch); code != 200 || batch.Status != "scheduled" || batch.PlanSheetStart == nil {
		t.Fatalf("schedule: %d %+v", code, batch)
	}
	if code := e.do("PUT", path+"/schedule", admin, api.BatchSchedule{OutsourceFactoryId: &e.plant}, &er); code != 400 || er.Code != "invalid_reference" {
		t.Fatalf("own plant: %d %+v", code, er)
	}
	if code := e.do("PUT", path+"/schedule", admin, api.BatchSchedule{OutsourceFactoryId: &e.outA}, &batch); code != 200 || batch.OutsourceFactoryName == nil || *batch.OutsourceFactoryName != "外协A" {
		t.Fatalf("outsource: %d %+v", code, batch)
	}
	input := []api.BatchItemInput{{PieceNo: "A", Qty: 2, PieceArea: floatPtr(1.25)}, {PieceNo: "B", Qty: 3, PieceArea: floatPtr(0.5)}, {PieceNo: "C", Qty: 1, PieceArea: floatPtr(2)}}
	var items []api.BatchItem
	if code := e.do("PUT", path+"/items", admin, input, &items); code != 200 || len(items) != 3 {
		t.Fatalf("replace: %d %+v", code, items)
	}
	if code := e.do("GET", path, admin, nil, &batch); code != 200 || batch.ItemCount != 3 || batch.ItemQty != 6 || batch.ItemArea != 6 {
		t.Fatalf("totals: %d %+v", code, batch)
	}
	if code := e.do("GET", path+"/items", admin, nil, &items); code != 200 || len(items) != 3 || items[0].Seq != 1 || items[1].PieceNo != "B" || items[2].Seq != 3 {
		t.Fatalf("order: %d %+v", code, items)
	}
	input = append(input, api.BatchItemInput{PieceNo: "B", Qty: 1})
	if code := e.do("PUT", path+"/items", admin, input, &er); code != 400 || er.Code != "duplicate_piece_no" || er.Message != "构件编号重复: B" {
		t.Fatalf("duplicate item: %d %+v", code, er)
	}
	var generated api.LabelGenerateResult
	if code := e.do("POST", path+"/labels/generate", admin, nil, &generated); code != 200 || generated.Total != 6 {
		t.Fatalf("generate: %d %+v", code, generated)
	}
	if code := e.do("PUT", path+"/items", admin, input[:3], &er); code != 409 || er.Code != "has_labels" || er.Message != "已生成标签，不能再替换构件清单" {
		t.Fatalf("labelled items: %d %+v", code, er)
	}
	if _, err := pool.Exec(context.Background(), "UPDATE batch SET status='done' WHERE id=$1", batch.Id); err != nil {
		t.Fatal(err)
	}
	if code := e.do("PUT", path+"/items", admin, input[:3], &er); code != 409 || er.Code != "batch_done" {
		t.Fatalf("done items: %d %+v", code, er)
	}
	if code := e.do("PUT", path+"/schedule", admin, start, &er); code != 409 || er.Code != "batch_done" {
		t.Fatalf("done schedule: %d %+v", code, er)
	}
}

var apiDateForTest = types.Date{Time: time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)}

func floatPtr(v float64) *float64 { return &v }

func TestListBatchesFilters(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	c, p, contract := e.batchContract(admin, "filters")
	date := types.Date{Time: time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)}
	urgent := api.Urgency("urgent")
	var b api.Batch
	for _, v := range []api.BatchCreate{{ContractId: contract.Id, LotNo: "LOT_001%", RequiredDate: &date, Urgency: &urgent}, {ContractId: contract.Id, LotNo: "LOT-002"}} {
		if code := e.do("POST", "/batches", admin, v, &b); code != 201 {
			t.Fatalf("create %s: %d", v.LotNo, code)
		}
	}
	path := fmt.Sprintf("/batches/%d/schedule", b.Id)
	if code := e.do("PUT", path, admin, api.BatchSchedule{PlanSheetStart: &date}, &b); code != 200 || b.Status != "scheduled" {
		t.Fatalf("schedule second batch: %d %+v", code, b)
	}
	var scheduled api.BatchPage
	if code := e.do("GET", "/batches?status=scheduled", admin, nil, &scheduled); code != 200 || scheduled.Total != 1 || scheduled.Items[0].LotNo != "LOT-002" {
		t.Fatalf("scheduled filter: %d %+v", code, scheduled)
	}
	for _, query := range []string{"keyword=_001%25", "status=pending&urgency=urgent", fmt.Sprintf("contractId=%d", contract.Id) + "&urgency=urgent", fmt.Sprintf("projectId=%d&customerId=%d", p.Id, c.Id) + "&urgency=urgent", "beginDate=2026-03-12&endDate=2026-03-12"} {
		var page api.BatchPage
		if code := e.do("GET", "/batches?"+query, admin, nil, &page); code != 200 || page.Total != 1 || page.Items[0].LotNo != "LOT_001%" {
			t.Fatalf("filter %s: %d %+v", query, code, page)
		}
	}
	var er api.Error
	if code := e.do("GET", "/batches?beginDate=2026-03-13&endDate=2026-03-12", admin, nil, &er); code != 400 {
		t.Fatalf("invalid dates: %d", code)
	}
}

func TestEnsureDefaultFactory(t *testing.T) {
	e := newEnv(t)
	if created, err := e.st.EnsureDefaultFactory(context.Background()); err != nil || created {
		t.Fatalf("factories exist: created=%v err=%v", created, err)
	}
	if _, err := pool.Exec(context.Background(), "TRUNCATE factory CASCADE"); err != nil {
		t.Fatal(err)
	}
	if created, err := e.st.EnsureDefaultFactory(context.Background()); err != nil || !created {
		t.Fatalf("empty: created=%v err=%v", created, err)
	}
}

// scanReq builds a scan request; QC jobs (25 / 45) default to a pass result.
func scanReq(bar, job string) api.ScanRequest {
	r := api.ScanRequest{BarCode: bar, JobCode: job}
	if job == "25" || job == "45" {
		pass := api.Pass
		r.Result = &pass
	}
	return r
}
