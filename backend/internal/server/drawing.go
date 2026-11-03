package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/store"
)

func drawingAPI(d *store.Drawing) api.Drawing {
	return api.Drawing{Id: d.ID, BatchId: d.BatchID, Kind: api.DrawingKind(d.Kind), DrawingNo: d.DrawingNo, Remark: d.Remark, OriginalName: d.OriginalName, ContentType: d.ContentType, SizeBytes: d.SizeBytes, Sha256: d.SHA256, StorageKey: d.StorageKey, UploadedBy: d.UploadedBy, UploaderName: d.UploaderName, UploadedAt: d.UploadedAt}
}
func (s *Server) ListDrawingBatches(ctx context.Context, r api.ListDrawingBatchesRequestObject) (api.ListDrawingBatchesResponseObject, error) {
	page, size := pageSize(r.Params.Page, r.Params.Size)
	if page < 1 || size < 1 || size > 200 {
		return api.ListDrawingBatches400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "分页无效")}, nil
	}
	var begin, end *time.Time
	if r.Params.BeginDate != nil {
		begin = &r.Params.BeginDate.Time
	}
	if r.Params.EndDate != nil {
		end = &r.Params.EndDate.Time
	}
	if begin != nil && end != nil && begin.After(*end) {
		return api.ListDrawingBatches400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "日期无效")}, nil
	}
	state := optionalString((*string)(r.Params.UploadState))
	items, total, err := s.store.DrawingBatches(ctx, auth.FromContext(ctx).Scope, store.DrawingFilter{Keyword: optionalString(r.Params.Keyword), State: state, Begin: begin, End: end}, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListDrawingBatches200JSONResponse{Items: make([]api.DrawingBatch, 0, len(items)), Total: total}
	for _, v := range items {
		out.Items = append(out.Items, api.DrawingBatch{Id: v.ID, LotNo: v.LotNo, ProjectName: v.ProjectName, CustomerName: v.CustomerName, CreatedAt: v.CreatedAt, UnfoldCount: v.UnfoldCount, FormingCount: v.FormingCount, NestingCount: v.NestingCount, LastUploadedAt: v.LastUploadedAt})
	}
	return out, nil
}
func (s *Server) ListDrawings(ctx context.Context, r api.ListDrawingsRequestObject) (api.ListDrawingsResponseObject, error) {
	ds, e := s.store.Drawings(ctx, auth.FromContext(ctx).Scope, r.Id)
	if errors.Is(e, store.ErrNotFound) {
		return api.ListDrawings404JSONResponse{ErrorJSONResponse: apiErr("not_found", "批次不存在")}, nil
	}
	if e != nil {
		return nil, e
	}
	out := make(api.ListDrawings200JSONResponse, 0, len(ds))
	for i := range ds {
		out = append(out, drawingAPI(&ds[i]))
	}
	return out, nil
}
func (s *Server) DeleteDrawing(ctx context.Context, r api.DeleteDrawingRequestObject) (api.DeleteDrawingResponseObject, error) {
	key, e := s.store.RemoveDrawing(ctx, auth.FromContext(ctx).Scope, r.Id)
	if errors.Is(e, store.ErrNotFound) {
		return api.DeleteDrawing404JSONResponse{ErrorJSONResponse: apiErr("not_found", "图纸不存在")}, nil
	}
	if e != nil {
		return nil, e
	}
	if e = s.files.Delete(key); e != nil {
		slog.ErrorContext(ctx, "remove drawing file", "key", key, "err", e)
	}
	return api.DeleteDrawing204Response{}, nil
}

// Upload and binary download are dispatched as raw HTTP after authentication.
// The generated multipart.Reader cannot install MaxBytesReader before decoding;
// its binary response also forces application/octet-stream instead of the actual MIME.
func (s *Server) UploadDrawing(context.Context, api.UploadDrawingRequestObject) (api.UploadDrawingResponseObject, error) {
	return nil, errors.New("raw upload route required")
}
func (s *Server) GetDrawingFile(context.Context, api.GetDrawingFileRequestObject) (api.GetDrawingFileResponseObject, error) {
	return nil, errors.New("raw download route required")
}
func rawID(r *http.Request, prefix, suffix string) (int64, error) {
	v := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, prefix), suffix)
	if strings.Contains(v, "/") {
		return 0, errors.New("invalid id")
	}
	id, e := strconv.ParseInt(v, 10, 64)
	if id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, e
}
func (s *Server) rawUpload(w http.ResponseWriter, r *http.Request) {
	id, e := rawID(r, "/batches/", "/drawings")
	if e != nil {
		writeErr(w, 404, "not_found", "批次不存在")
		return
	}
	ok, e := s.store.BatchInScope(r.Context(), auth.FromContext(r.Context()).Scope, id)
	if e != nil {
		writeErr(w, 500, "internal", "查询失败")
		return
	}
	if !ok {
		writeErr(w, 404, "not_found", "批次不存在")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.maxUpload+65536)
	mr, e := r.MultipartReader()
	if e != nil {
		writeErr(w, 400, "invalid_request", "需要 multipart/form-data")
		return
	}
	d := store.Drawing{BatchID: id, UploadedBy: auth.FromContext(r.Context()).UserID}
	var key string
	var gotFile bool
	fail := func(err error) {
		if key != "" {
			_ = s.files.Delete(key)
		}
		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			writeErr(w, 413, "too_large", "文件超过上传限制")
		} else {
			writeErr(w, 400, "invalid_request", "无效的上传文件")
		}
	}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			fail(err)
			return
		}
		name := part.FormName()
		switch name {
		case "file":
			if gotFile {
				_ = part.Close()
				fail(errors.New("duplicate file"))
				return
			}
			gotFile = true
			d.OriginalName = path.Base(strings.ReplaceAll(part.FileName(), "\\", "/"))
			ext := strings.ToLower(path.Ext(d.OriginalName))
			mimeType := map[string]string{".pdf": "application/pdf", ".dwg": "application/acad", ".dxf": "application/dxf", ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg"}[ext]
			if mimeType == "" {
				_ = part.Close()
				fail(errors.New("extension"))
				return
			}
			d.ContentType = mimeType
			key, d.SHA256, d.SizeBytes, err = s.files.Put(part)
			if err != nil {
				_ = part.Close()
				fail(err)
				return
			}
			d.StorageKey = key
		case "kind", "drawingNo", "remark":
			b, err := io.ReadAll(io.LimitReader(part, 8193))
			if err != nil || len(b) > 8192 {
				_ = part.Close()
				fail(errors.New("field too large"))
				return
			}
			switch name {
			case "kind":
				d.Kind = string(b)
			case "drawingNo":
				d.DrawingNo = string(b)
			case "remark":
				d.Remark = string(b)
			}
		}
		_ = part.Close()
	}
	if !gotFile || d.SizeBytes == 0 || d.SizeBytes > s.maxUpload || (d.Kind != "unfold" && d.Kind != "forming" && d.Kind != "nesting") {
		if key != "" {
			_ = s.files.Delete(key)
		}
		if d.SizeBytes > s.maxUpload {
			writeErr(w, 413, "too_large", "文件超过上传限制")
		} else {
			writeErr(w, 400, "invalid_request", "文件或类型无效")
		}
		return
	}
	saved, e := s.store.AddDrawing(r.Context(), auth.FromContext(r.Context()).Scope, d)
	if e != nil {
		_ = s.files.Delete(key)
		if errors.Is(e, store.ErrNotFound) {
			writeErr(w, 404, "not_found", "批次不存在")
		} else {
			slog.ErrorContext(r.Context(), "save drawing", "err", e)
			writeErr(w, 500, "internal", "保存失败")
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	_ = json.NewEncoder(w).Encode(drawingAPI(saved))
}
func (s *Server) rawFile(w http.ResponseWriter, r *http.Request) {
	id, e := rawID(r, "/drawings/", "/file")
	if e != nil {
		writeErr(w, 404, "not_found", "图纸不存在")
		return
	}
	d, e := s.store.Drawing(r.Context(), auth.FromContext(r.Context()).Scope, id)
	if errors.Is(e, store.ErrNotFound) {
		writeErr(w, 404, "not_found", "图纸不存在")
		return
	}
	if e != nil {
		writeErr(w, 500, "internal", "查询失败")
		return
	}
	f, e := s.files.Open(d.StorageKey)
	if e != nil {
		slog.ErrorContext(r.Context(), "open drawing", "err", e)
		writeErr(w, 500, "internal", "文件不存在")
		return
	}
	defer f.Close()
	disposition := "inline"
	if r.URL.Query().Get("download") == "1" {
		disposition = "attachment"
	}
	w.Header().Set("Content-Type", d.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename*=UTF-8''%s", disposition, url.PathEscape(d.OriginalName)))
	w.Header().Set("Content-Length", strconv.FormatInt(d.SizeBytes, 10))
	_, _ = io.Copy(w, f)
}
