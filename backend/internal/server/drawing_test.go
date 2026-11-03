package server_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/belimked/gz_mes/backend/internal/api"
)

func TestDrawingFlow(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "admin-pass")
	_, _, contract := e.batchContract(admin, "drawing")
	var b api.Batch
	if c := e.do("POST", "/batches", admin, api.BatchCreate{ContractId: contract.Id, LotNo: "DRAW-1"}, &b); c != 201 {
		t.Fatal(c)
	}
	base := fmt.Sprintf("/batches/%d/drawings", b.Id)
	upload := func(name, kind string, content []byte) (int, api.Drawing) {
		t.Helper()
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		_ = mw.WriteField("kind", kind)
		part, _ := mw.CreateFormFile("file", name)
		_, _ = part.Write(content)
		_ = mw.Close()
		req, _ := http.NewRequest("POST", e.srv.URL+"/api"+base, &buf)
		req.Header.Set("Authorization", "Bearer "+admin)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var d api.Drawing
		if resp.StatusCode == 201 {
			if err = json.NewDecoder(resp.Body).Decode(&d); err != nil {
				t.Fatal(err)
			}
		}
		_ = resp.Body.Close()
		return resp.StatusCode, d
	}
	pdf := []byte("%PDF-1.4\n drawing test")
	r, d := upload("展开图纸.PDF", "unfold", pdf)
	if r != 201 {
		t.Fatalf("upload: %d", r)
	}
	sum := sha256.Sum256(pdf)
	if d.Sha256 != hex.EncodeToString(sum[:]) || d.ContentType != "application/pdf" {
		t.Fatalf("metadata: %+v", d)
	}
	var ds []api.Drawing
	if c := e.do("GET", base, admin, nil, &ds); c != 200 || len(ds) != 1 || ds[0].UploaderName == "" {
		t.Fatalf("list: %d %+v", c, ds)
	}
	var page api.DrawingBatchPage
	if c := e.do("GET", "/drawing-batches?uploadState=partial", admin, nil, &page); c != 200 || page.Total != 1 || page.Items[0].UnfoldCount != 1 {
		t.Fatalf("partial: %d %+v", c, page)
	}
	if c := e.do("GET", "/drawing-batches?uploadState=complete", admin, nil, &page); c != 200 || page.Total != 0 {
		t.Fatalf("complete: %d %+v", c, page)
	}
	if c := e.do("GET", "/drawing-batches?uploadState=none", admin, nil, &page); c != 200 || page.Total != 0 {
		t.Fatalf("none: %d %+v", c, page)
	}
	r, _ = upload("forming.pdf", "forming", pdf)
	if r != 201 {
		t.Fatal(r)
	}
	if c := e.do("GET", "/drawing-batches?uploadState=complete", admin, nil, &page); c != 200 || page.Total != 1 || page.Items[0].FormingCount != 1 {
		t.Fatalf("complete: %d %+v", c, page)
	}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/drawings/%d/file", e.srv.URL, d.Id), nil)
	req.Header.Set("Authorization", "Bearer "+admin)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != 200 || !bytes.Equal(got, pdf) || resp.Header.Get("Content-Type") != "application/pdf" || !strings.Contains(resp.Header.Get("Content-Disposition"), url.PathEscape(d.OriginalName)) {
		t.Fatalf("download: %d %q", resp.StatusCode, resp.Header.Get("Content-Disposition"))
	}
	r, _ = upload("bad.exe", "nesting", pdf)
	if r != 400 {
		t.Fatalf("extension: %d", r)
	}
	r, _ = upload("huge.pdf", "nesting", bytes.Repeat([]byte("x"), 50*1024*1024+1))
	if r != 413 {
		t.Fatalf("oversize: %d", r)
	}
	if c := e.do("GET", "/batches/999999/drawings", admin, nil, nil); c != 404 {
		t.Fatal(c)
	}
	e.outsourceUser("drawing-other", e.outA)
	other := e.login("drawing-other", "pw-drawing-other")
	if c := e.do("GET", base, other, nil, nil); c != 404 {
		t.Fatalf("factory: %d", c)
	}
	if c := e.do("GET", "/drawing-batches", other, nil, &page); c != 200 || page.Total != 0 {
		t.Fatalf("factory summary: %d %+v", c, page)
	}
	if c := e.do("DELETE", fmt.Sprintf("/drawings/%d", d.Id), other, nil, nil); c != 404 {
		t.Fatalf("factory delete: %d", c)
	}
	if c := e.do("GET", fmt.Sprintf("/drawings/%d/file", d.Id), other, nil, nil); c != 404 {
		t.Fatalf("file factory: %d", c)
	}
	if c := e.do("DELETE", fmt.Sprintf("/drawings/%d", d.Id), admin, nil, nil); c != 204 {
		t.Fatal(c)
	}
	if c := e.do("GET", fmt.Sprintf("/drawings/%d/file", d.Id), admin, nil, nil); c != 404 {
		t.Fatalf("deleted: %d", c)
	}
}
