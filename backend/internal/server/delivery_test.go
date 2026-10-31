package server_test

import (
	"fmt"
	"math"
	"regexp"
	"testing"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/oapi-codegen/runtime/types"
)

func TestHandoverAndDeliveryFlow(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	customer, _, contract := e.batchContract(admin, "delivery")
	var b api.Batch
	if code := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: "DELIVERY-1"}, &b); code != 201 {
		t.Fatalf("batch: %d", code)
	}
	path := fmt.Sprintf("/batches/%d", b.Id)
	var items []api.BatchItem
	if code := e.do("PUT", path+"/items", admin, []api.BatchItemInput{{PieceNo: "A", Qty: 2, PieceArea: floatPtr(1.5)}, {PieceNo: "B", Qty: 1, PieceArea: floatPtr(2)}}, &items); code != 200 {
		t.Fatalf("items: %d", code)
	}
	var gen api.LabelGenerateResult
	if code := e.do("POST", path+"/labels/generate", admin, nil, &gen); code != 200 || gen.Total != 3 {
		t.Fatalf("generate: %d %+v", code, gen)
	}
	var page api.LabelPage
	if code := e.do("GET", fmt.Sprintf("/labels?batchId=%d", b.Id), admin, nil, &page); code != 200 || len(page.Items) != 3 {
		t.Fatalf("labels: %d %+v", code, page)
	}
	codes := make([]string, 0, 3)
	for _, v := range page.Items {
		codes = append(codes, v.BarCode)
	}
	var er api.Error
	handover := api.HandoverCreate{BatchId: b.Id, JobCode: "10", BarCodes: codes}
	if code := e.do("POST", "/handovers", admin, handover, &er); code != 409 || er.Code != "not_scanned" {
		t.Fatalf("not scanned: %d %+v", code, er)
	}
	for _, bar := range codes {
		var r api.ScanResult
		if code := e.do("POST", "/scans", admin, scanReq(bar, "10"), &r); code != 201 {
			t.Fatalf("scan 10: %d", code)
		}
	}
	var h api.HandoverDetail
	if code := e.do("POST", "/handovers", admin, handover, &h); code != 201 || h.Qty != 3 || len(h.Labels) != 3 || !regexp.MustCompile(`^JJ\d{8}\d{4}$`).MatchString(h.HandoverNo) {
		t.Fatalf("handover: %d %+v", code, h)
	}
	if code := e.do("POST", "/handovers", admin, handover, &er); code != 409 || er.Code != "label_in_handover" {
		t.Fatalf("duplicate handover: %d %+v", code, er)
	}
	hp := fmt.Sprintf("/handovers/%d/receive", h.Id)
	var received api.Handover
	if code := e.do("POST", hp, admin, nil, &received); code != 200 || !received.Received {
		t.Fatalf("receive: %d %+v", code, received)
	}
	if code := e.do("POST", hp, admin, nil, &er); code != 409 || er.Code != "already_received" {
		t.Fatalf("receive again: %d %+v", code, er)
	}
	var hs api.HandoverPage
	if code := e.do("GET", "/handovers?received=true", admin, nil, &hs); code != 200 || hs.Total != 1 {
		t.Fatalf("list handovers: %d %+v", code, hs)
	}
	note := api.DeliveryNoteCreate{BatchId: b.Id, BarCodes: codes, DeliveredAt: types.Date{Time: time.Now()}}
	if code := e.do("POST", "/delivery-notes", admin, note, &er); code != 409 || er.Code != "not_scanned" {
		t.Fatalf("not loaded: %d %+v", code, er)
	}
	for _, bar := range codes {
		for _, job := range []string{"15", "20", "25", "30", "35", "40", "45", "50", "55"} {
			var r api.ScanResult
			if code := e.do("POST", "/scans", admin, scanReq(bar, job), &r); code != 201 {
				t.Fatalf("scan %s: %d", job, code)
			}
		}
	}
	var n api.DeliveryNoteDetail
	if code := e.do("POST", "/delivery-notes", admin, note, &n); code != 201 || n.Qty != 3 || len(n.Labels) != 3 || !regexp.MustCompile(`^FH\d{8}\d{4}$`).MatchString(n.NoteNo) {
		t.Fatalf("note: %d %+v", code, n)
	}
	var progress api.BatchProgress
	if code := e.do("GET", path+"/progress", admin, nil, &progress); code != 200 || progress.Status != "done" {
		t.Fatalf("progress: %d %+v", code, progress)
	}
	if code := e.do("POST", "/delivery-notes", admin, note, &er); code != 409 || er.Code != "label_delivered" {
		t.Fatalf("duplicate note: %d %+v", code, er)
	}
	var ns api.DeliveryNotePage
	if code := e.do("GET", "/delivery-notes?accounted=false", admin, nil, &ns); code != 200 || ns.Total != 1 {
		t.Fatalf("notes: %d %+v", code, ns)
	}
	account := api.DeliverAccountCreate{CustomerId: customer.Id, DeliveryNoteIds: []int64{n.Id}, UnitPrice: 100, ExtraAmount: floatPtr(50)}
	var a api.DeliverAccountDetail
	if code := e.do("POST", "/deliver-accounts", admin, account, &a); code != 201 || a.Status != "draft" || math.Abs(a.Amount-math.Round((n.Area*100+50)*100)/100) > 0.001 || !regexp.MustCompile(`^DZ\d{8}\d{4}$`).MatchString(a.AccountNo) {
		t.Fatalf("account: %d %+v", code, a)
	}
	if code := e.do("POST", "/deliver-accounts", admin, account, &er); code != 409 || er.Code != "note_accounted" {
		t.Fatalf("account again: %d %+v", code, er)
	}
	ap := fmt.Sprintf("/deliver-accounts/%d", a.Id)
	var confirmed api.DeliverAccount
	if code := e.do("POST", ap+"/confirm", admin, nil, &confirmed); code != 200 || confirmed.Status != "confirmed" {
		t.Fatalf("confirm: %d %+v", code, confirmed)
	}
	if code := e.do("POST", ap+"/confirm", admin, nil, &er); code != 409 || er.Code != "already_confirmed" {
		t.Fatalf("confirm again: %d %+v", code, er)
	}
	var accounts api.DeliverAccountPage
	if code := e.do("GET", "/deliver-accounts?status=confirmed", admin, nil, &accounts); code != 200 || accounts.Total != 1 {
		t.Fatalf("accounts: %d %+v", code, accounts)
	}
	if code := e.do("GET", ap, admin, nil, &a); code != 200 || len(a.Notes) != 1 || !a.Notes[0].Accounted {
		t.Fatalf("account detail: %d %+v", code, a)
	}
	e.outsourceUser("delivery-wa", e.outA)
	wa := e.login("delivery-wa", "pw-delivery-wa")
	if code := e.do("GET", fmt.Sprintf("/delivery-notes/%d", n.Id), wa, nil, &er); code != 404 || er.Code != "not_found" {
		t.Fatalf("scope: %d %+v", code, er)
	}
}
