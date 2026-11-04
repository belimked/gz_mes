package server_test

import (
	"testing"

	"github.com/belimked/gz_mes/backend/internal/api"
)

func TestAuditLogs(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	if code := e.do("POST", "/projects", admin, api.ProjectCreate{FactoryId: e.plant, Code: "AUD", FullName: "审计项目"}, nil); code != 201 {
		t.Fatal(code)
	}
	var page api.AuditLogPage
	if code := e.do("GET", "/audit-logs?method=POST", admin, nil, &page); code != 200 {
		t.Fatal(code)
	}
	found := false
	for _, a := range page.Items {
		if a.Route == "/projects" && a.Status == 201 && a.Username == "admin" {
			found = true
		}
	}
	if !found {
		t.Fatalf("mutation absent: %+v", page.Items)
	}
	before := page.Total
	e.do("POST", "/scans", admin, api.ScanRequest{BarCode: "missing", JobCode: "10"}, nil)
	e.do("GET", "/projects", admin, nil, nil)
	e.do("GET", "/audit-logs?method=POST", admin, nil, &page)
	if page.Total != before {
		t.Fatal("GET audited")
	}
	e.outsourceUser("guest", e.outA)
	guest := e.login("guest", "pw-guest")
	if code := e.do("GET", "/audit-logs", guest, nil, nil); code != 403 {
		t.Fatal(code)
	}
	if code := e.do("POST", "/auth/login", "", api.LoginRequest{Username: "unknown", Password: "bad"}, nil); code != 401 {
		t.Fatal(code)
	}
	if code := e.do("GET", "/audit-logs?result=fail&keyword=unknown", admin, nil, &page); code != 200 || page.Total != 1 || page.Items[0].Status != 401 || page.Items[0].UserId != nil {
		t.Fatalf("login failure: %d %+v", code, page)
	}
}
