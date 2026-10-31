package server_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/belimked/gz_mes/backend/internal/api"
)

func TestSystemAdminOnly(t *testing.T) {
	e := newEnv(t)
	e.outsourceUser("waixie", e.outA)
	token := e.login("waixie", "pw-waixie")
	for _, test := range []struct {
		method, path string
		body         any
	}{{"GET", "/users", nil}, {"POST", "/factories", api.FactoryCreate{Code: "NEW", Name: "new", Kind: "own"}}} {
		var er api.Error
		if code := e.do(test.method, test.path, token, test.body, &er); code != 403 || er.Code != "forbidden" {
			t.Fatalf("%s %s: %d %+v", test.method, test.path, code, er)
		}
	}
}

func TestFactoryCrud(t *testing.T) {
	e := newEnv(t)
	token := e.login("admin", "admin-pass")
	body := api.FactoryCreate{Code: "NEW", Name: "新厂", Kind: "own"}
	var f api.Factory
	if code := e.do("POST", "/factories", token, body, &f); code != 201 || f.Code != "NEW" || f.IsStop {
		t.Fatalf("create: %d %+v", code, f)
	}
	var er api.Error
	if code := e.do("POST", "/factories", token, body, &er); code != 409 || er.Code != "duplicate_code" {
		t.Fatalf("duplicate: %d %+v", code, er)
	}
	if code := e.do("PUT", fmt.Sprintf("/factories/%d", f.Id), token, api.FactoryUpdate{Name: "改名", IsStop: true}, &f); code != 200 || f.Name != "改名" || !f.IsStop {
		t.Fatalf("update: %d %+v", code, f)
	}
	var fs []api.Factory
	if code := e.do("GET", "/factories", token, nil, &fs); code != 200 {
		t.Fatal(code)
	}
	found := false
	for _, item := range fs {
		if item.Id == f.Id && item.IsStop {
			found = true
		}
	}
	if !found {
		t.Fatalf("stopped factory missing: %+v", fs)
	}
	if code := e.do("PUT", "/factories/999999", token, api.FactoryUpdate{Name: "x", IsStop: false}, &er); code != 404 {
		t.Fatalf("missing factory: %d %+v", code, er)
	}
}

func TestUserLifecycle(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	body := api.UserCreate{Username: "worker", DisplayName: "工人", Password: "old-pass"}
	var er api.Error
	if code := e.do("POST", "/users", admin, body, &er); code != 400 {
		t.Fatalf("no factory: %d %+v", code, er)
	}
	factories, menus := []int64{e.plant}, []int64{201, 203}
	body.FactoryIds, body.MenuIds = &factories, &menus
	var u api.UserDetail
	if code := e.do("POST", "/users", admin, body, &u); code != 201 || len(u.FactoryIds) != 1 || len(u.MenuIds) != 2 {
		t.Fatalf("create: %d %+v", code, u)
	}
	worker := e.login("worker", "old-pass")
	var me api.MeResponse
	if code := e.do("GET", "/auth/me", worker, nil, &me); code != 200 || len(me.Menus) != 1 || me.Menus[0].Children == nil || len(*me.Menus[0].Children) != len(menus) {
		t.Fatalf("menus: %d %+v", code, me)
	}
	path := fmt.Sprintf("/users/%d", u.Id)
	update := api.UserUpdate{DisplayName: "工人", Enabled: false, IsAdmin: false, FactoryIds: factories, MenuIds: []int64{}}
	if code := e.do("PUT", path, admin, update, &u); code != 200 || u.Enabled || len(u.MenuIds) != 0 {
		t.Fatalf("disable: %d %+v", code, u)
	}
	if code := e.do("POST", "/auth/login", "", api.LoginRequest{Username: "worker", Password: "old-pass"}, &er); code != 401 || er.Code != "invalid_credentials" {
		t.Fatalf("disabled login: %d %+v", code, er)
	}
	if code := e.do("GET", "/auth/me", worker, nil, &er); code != 401 {
		t.Fatalf("old token: %d", code)
	}
	var adminUser api.UserPage
	e.do("GET", "/users?keyword=admin", admin, nil, &adminUser)
	self := fmt.Sprintf("/users/%d", adminUser.Items[0].Id)
	if code := e.do("PUT", self, admin, api.UserUpdate{DisplayName: "admin", Enabled: false, IsAdmin: true, FactoryIds: []int64{}, MenuIds: []int64{}}, &er); code != 409 || er.Code != "self_lock" {
		t.Fatalf("self lock: %d %+v", code, er)
	}
	if code := e.do("PUT", path+"/password", admin, api.PasswordReset{Password: "new-pass"}, nil); code != 204 {
		t.Fatalf("reset: %d", code)
	}
	update.Enabled = true
	if code := e.do("PUT", path, admin, update, &u); code != 200 {
		t.Fatalf("enable: %d", code)
	}
	e.login("worker", "new-pass")
	var page api.UserPage
	if code := e.do("GET", "/users?keyword=worker", admin, nil, &page); code != 200 || page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("filter: %d %+v", code, page)
	}
	var all []api.MenuItem
	if code := e.do("GET", "/menus", admin, nil, &all); code != 200 {
		t.Fatal(code)
	}
	var seeded int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM menu").Scan(&seeded); err != nil {
		t.Fatal(err)
	}
	if got := len(all); got != seeded {
		t.Fatalf("menu seed count: got %d, want %d", got, seeded)
	}
}
