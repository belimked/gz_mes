package server_test

import (
	"fmt"
	"testing"

	"github.com/belimked/gz_mes/backend/internal/api"
)

func TestRolesAndUserGrants(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	e.outsourceUser("guest", e.outA)
	guest := e.login("guest", "pw-guest")
	var er api.Error
	if code := e.do("GET", "/roles", guest, nil, &er); code != 403 {
		t.Fatal(code)
	}
	in := api.RoleInput{Code: "  auditor ", Name: " 审核员 ", IsStop: false, MenuIds: []int64{201}}
	var r api.Role
	if code := e.do("POST", "/roles", admin, in, &r); code != 201 || r.Code != "auditor" || r.Name != "审核员" {
		t.Fatalf("create %d %+v", code, r)
	}
	if code := e.do("POST", "/roles", admin, in, &er); code != 409 || er.Code != "duplicate_code" {
		t.Fatalf("duplicate %d %+v", code, er)
	}
	in.Code = "user:forbidden"
	if code := e.do("POST", "/roles", admin, in, &er); code != 400 {
		t.Fatal(code)
	}
	in.Code = "auditor"
	in.MenuIds = []int64{999999}
	if code := e.do("PUT", fmt.Sprintf("/roles/%d", r.Id), admin, in, &er); code != 400 {
		t.Fatal(code)
	}
	in.MenuIds = []int64{201}
	var page api.UserPage
	e.do("GET", "/users?keyword=guest", admin, nil, &page)
	u := page.Items[0]
	path := fmt.Sprintf("/users/%d", u.Id)
	update := api.UserUpdate{DisplayName: u.DisplayName, Enabled: true, IsAdmin: false, FactoryIds: u.FactoryIds, MenuIds: []int64{}, RoleIds: &[]int64{r.Id}}
	if code := e.do("PUT", path, admin, update, &u); code != 200 || len(u.RoleIds) != 1 {
		t.Fatalf("assign %d %+v", code, u)
	}
	var me api.MeResponse
	e.do("GET", "/auth/me", guest, nil, &me)
	found := false
	for _, root := range me.Menus {
		if root.Children != nil {
			for _, child := range *root.Children {
				if child.Id == 201 {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("role menu missing")
	}
	if code := e.do("DELETE", fmt.Sprintf("/roles/%d", r.Id), admin, nil, &er); code != 409 {
		t.Fatalf("occupied %d %+v", code, er)
	}
	update.RoleIds = nil
	if code := e.do("PUT", path, admin, update, &u); code != 200 || len(u.RoleIds) != 1 {
		t.Fatal("omitted roleIds lost", code, u)
	}
	in.IsStop = true
	if code := e.do("PUT", fmt.Sprintf("/roles/%d", r.Id), admin, in, &r); code != 200 {
		t.Fatal(code)
	}
	update.RoleIds = &[]int64{r.Id}
	if code := e.do("PUT", path, admin, update, &u); code != 200 || len(u.RoleIds) != 1 {
		t.Fatalf("existing stopped role must remain assignable: %d %+v", code, u)
	}
	newUser := api.UserCreate{Username: "fresh", DisplayName: "新人", Password: "new-pass", FactoryIds: &[]int64{e.plant}, RoleIds: &[]int64{r.Id}}
	if code := e.do("POST", "/users", admin, newUser, &er); code != 400 {
		t.Fatalf("new stopped role assignment: %d %+v", code, er)
	}
	e.do("GET", "/auth/me", guest, nil, &me)
	for _, root := range me.Menus {
		if root.Children != nil {
			for _, child := range *root.Children {
				if child.Id == 201 {
					t.Fatal("stopped role menu still visible")
				}
			}
		}
	}
	update.RoleIds = &[]int64{}
	if code := e.do("PUT", path, admin, update, &u); code != 200 {
		t.Fatal(code)
	}
	if code := e.do("DELETE", fmt.Sprintf("/roles/%d", r.Id), admin, nil, nil); code != 204 {
		t.Fatal(code)
	}
	var roles []api.Role
	e.do("GET", "/roles", admin, nil, &roles)
	for _, role := range roles {
		if len(role.Code) >= 5 && role.Code[:5] == "user:" {
			t.Fatal("personal role listed")
		}
	}
	var personal int64
	if err := pool.QueryRow(t.Context(), "SELECT id FROM role WHERE code='user:guest'").Scan(&personal); err != nil {
		t.Fatal(err)
	}
	if code := e.do("PUT", fmt.Sprintf("/roles/%d", personal), admin, in, &er); code != 404 {
		t.Fatal(code)
	}
}
