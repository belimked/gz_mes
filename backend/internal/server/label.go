package server

import (
	"context"
	"errors"
	"strings"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/store"
)

func toAPILabel(l *store.Label) api.Label {
	return api.Label{Id: l.ID, BarCode: l.BarCode, FactoryId: l.FactoryID, BatchId: l.BatchID, LotNo: l.LotNo, ProjectName: l.ProjectName, BatchItemId: l.BatchItemID, PieceNo: l.PieceNo, PieceIndex: l.PieceIndex, Qty: l.Qty, DrawingNo: l.DrawingNo, Pattern: l.Pattern, Thickness: l.Thickness, Color: l.Color, PieceArea: l.PieceArea, Cancelled: l.Cancelled, LastJobId: l.LastJobID, LastJobName: l.LastJobName, LastScannedAt: l.LastScannedAt, CreatedAt: l.CreatedAt}
}
func toAPIScan(v store.Scan) api.Scan {
	return api.Scan{Id: v.ID, LabelId: v.LabelID, BarCode: v.BarCode, BatchId: v.BatchID, LotNo: v.LotNo, ProjectName: v.ProjectName, PieceNo: v.PieceNo, PieceIndex: v.PieceIndex, PieceArea: v.PieceArea, JobId: v.JobID, JobName: v.JobName, ScannedBy: v.ScannedBy, StaffNo: v.StaffNo, ScannedAt: v.ScannedAt, Device: v.Device}
}
func toAPIJob(j store.Job) api.Job {
	return api.Job{Id: j.ID, Code: j.Code, Name: j.Name, Seq: j.Seq, Stage: api.JobStage(j.Stage), IsQc: j.IsQc}
}
func toAPIProgress(v *store.Progress) api.BatchProgress {
	p := api.BatchProgress{BatchId: v.BatchID, Status: api.BatchStatus(v.Status), TotalQty: v.TotalQty, TotalArea: v.TotalArea, LabelQty: v.LabelQty, Jobs: make([]api.JobProgress, 0, len(v.Jobs))}
	for _, j := range v.Jobs {
		p.Jobs = append(p.Jobs, api.JobProgress{JobId: j.JobID, JobCode: j.JobCode, JobName: j.JobName, Seq: j.Seq, ScannedQty: j.ScannedQty, ScannedArea: j.ScannedArea})
	}
	return p
}
func pageSize(page, size *int) (int, int) {
	p, n := 1, 20
	if page != nil {
		p = *page
	}
	if size != nil {
		n = *size
	}
	return p, n
}
func (s *Server) GenerateLabels(ctx context.Context, req api.GenerateLabelsRequestObject) (api.GenerateLabelsResponseObject, error) {
	p := auth.FromContext(ctx)
	created, total, err := s.store.GenerateLabels(ctx, p.Scope, req.Id, p.UserID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.GenerateLabels404JSONResponse(apiErr("not_found", "批次不存在")), nil
	case errors.Is(err, store.ErrNoItems):
		return api.GenerateLabels400JSONResponse{ErrorJSONResponse: apiErr("no_items", "批次尚无构件清单")}, nil
	case err != nil:
		return nil, err
	}
	return api.GenerateLabels200JSONResponse{Created: created, Total: total}, nil
}
func (s *Server) GetBatchProgress(ctx context.Context, req api.GetBatchProgressRequestObject) (api.GetBatchProgressResponseObject, error) {
	p, err := s.store.BatchProgress(ctx, auth.FromContext(ctx).Scope, req.Id)
	if errors.Is(err, store.ErrNotFound) {
		return api.GetBatchProgress404JSONResponse(apiErr("not_found", "批次不存在")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetBatchProgress200JSONResponse(toAPIProgress(p)), nil
}
func (s *Server) ListLabels(ctx context.Context, req api.ListLabelsRequestObject) (api.ListLabelsResponseObject, error) {
	f := store.LabelFilter{BatchID: req.Params.BatchId, Cancelled: req.Params.Cancelled}
	if req.Params.Keyword != nil {
		f.Keyword = *req.Params.Keyword
	}
	page, size := pageSize(req.Params.Page, req.Params.Size)
	items, total, err := s.store.ListLabels(ctx, auth.FromContext(ctx).Scope, f, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListLabels200JSONResponse{Items: make([]api.Label, 0, len(items)), Total: total}
	for i := range items {
		out.Items = append(out.Items, toAPILabel(&items[i]))
	}
	return out, nil
}
func (s *Server) GetLabel(ctx context.Context, req api.GetLabelRequestObject) (api.GetLabelResponseObject, error) {
	l, scans, err := s.store.GetLabel(ctx, auth.FromContext(ctx).Scope, req.BarCode)
	if errors.Is(err, store.ErrNotFound) {
		return api.GetLabel404JSONResponse(apiErr("not_found", "标签不存在")), nil
	}
	if err != nil {
		return nil, err
	}
	out := api.LabelDetail{Id: l.ID, BarCode: l.BarCode, FactoryId: l.FactoryID, BatchId: l.BatchID, LotNo: l.LotNo, ProjectName: l.ProjectName, BatchItemId: l.BatchItemID, PieceNo: l.PieceNo, PieceIndex: l.PieceIndex, Qty: l.Qty, DrawingNo: l.DrawingNo, Pattern: l.Pattern, Thickness: l.Thickness, Color: l.Color, PieceArea: l.PieceArea, Cancelled: l.Cancelled, LastJobId: l.LastJobID, LastJobName: l.LastJobName, LastScannedAt: l.LastScannedAt, CreatedAt: l.CreatedAt, Scans: make([]api.Scan, 0, len(scans))}
	for _, v := range scans {
		out.Scans = append(out.Scans, toAPIScan(v))
	}
	return api.GetLabel200JSONResponse(out), nil
}
func (s *Server) CancelLabel(ctx context.Context, req api.CancelLabelRequestObject) (api.CancelLabelResponseObject, error) {
	l, err := s.store.CancelLabel(ctx, auth.FromContext(ctx).Scope, req.BarCode)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.CancelLabel404JSONResponse(apiErr("not_found", "标签不存在")), nil
	case errors.Is(err, store.ErrHasScans):
		return api.CancelLabel409JSONResponse(apiErr("has_scans", "标签已有扫描记录，不能作废")), nil
	case errors.Is(err, store.ErrConflict):
		return api.CancelLabel409JSONResponse(apiErr("already_cancelled", "标签已作废")), nil
	case err != nil:
		return nil, err
	}
	return api.CancelLabel200JSONResponse(toAPILabel(l)), nil
}
func (s *Server) ListScans(ctx context.Context, req api.ListScansRequestObject) (api.ListScansResponseObject, error) {
	f := store.ScanFilter{BatchID: req.Params.BatchId, JobID: req.Params.JobId, Begin: dateValue(req.Params.BeginDate), Keyword: optionalString(req.Params.Keyword), Staff: optionalString(req.Params.Staff), Newest: true}
	if req.Params.EndDate != nil {
		end := req.Params.EndDate.AddDate(0, 0, 1)
		f.End = &end
	}
	if f.Begin != nil && f.End != nil && !f.Begin.Before(*f.End) {
		return api.ListScans400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "日期范围无效")}, nil
	}
	page, size := pageSize(req.Params.Page, req.Params.Size)
	items, total, err := s.store.ListScans(ctx, auth.FromContext(ctx).Scope, f, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListScans200JSONResponse{Items: make([]api.Scan, 0, len(items)), Total: total}
	for _, v := range items {
		out.Items = append(out.Items, toAPIScan(v))
	}
	return out, nil
}
func (s *Server) ScanLabel(ctx context.Context, req api.ScanLabelRequestObject) (api.ScanLabelResponseObject, error) {
	p := auth.FromContext(ctx)
	b := req.Body
	in := store.ScanInput{BarCode: strings.TrimSpace(b.BarCode), JobCode: strings.TrimSpace(b.JobCode), Device: optionalString(b.Device), UserID: p.UserID, DefectJobID: b.DefectJobId, DefectCategory: optionalString(b.DefectCategory), DefectReason: optionalString(b.DefectReason)}
	if b.Result != nil {
		in.Result = string(*b.Result)
	}
	if in.Result != "fail" {
		in.DefectJobID, in.DefectCategory, in.DefectReason = nil, "", ""
	}
	scan, qc, label, progress, err := s.store.Scan(ctx, p.Scope, in)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.ScanLabel404JSONResponse(apiErr("not_found", "标签不存在")), nil
	case errors.Is(err, store.ErrBadRef):
		return api.ScanLabel400JSONResponse{ErrorJSONResponse: apiErr("invalid_job", "工序代码不存在")}, nil
	case errors.Is(err, store.ErrQcResultRequired):
		return api.ScanLabel400JSONResponse{ErrorJSONResponse: apiErr("qc_result_required", "质检岗位必须选择合格 / 不合格")}, nil
	case errors.Is(err, store.ErrLabelCancelled):
		return api.ScanLabel409JSONResponse(apiErr("label_cancelled", "标签已作废")), nil
	case errors.Is(err, store.ErrAlreadyScanned):
		return api.ScanLabel409JSONResponse(apiErr("already_scanned", "该标签在此工序已扫描")), nil
	case errors.Is(err, store.ErrOutOfOrder):
		return api.ScanLabel409JSONResponse(apiErr("out_of_order", "请先完成上一道工序: "+strings.TrimPrefix(err.Error(), store.ErrOutOfOrder.Error()+": "))), nil
	case err != nil:
		return nil, err
	}
	out := api.ScanLabel201JSONResponse{Label: toAPILabel(label), Progress: toAPIProgress(progress)}
	if scan != nil {
		v := toAPIScan(*scan)
		out.Scan = &v
	}
	if qc != nil {
		v := toAPIQc(*qc)
		out.Qc = &v
	}
	return out, nil
}
