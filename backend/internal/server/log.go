package server

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/store"
	"github.com/oapi-codegen/runtime/types"
)

func toAPILabelLog(v store.LabelLog) api.LabelLog {
	return api.LabelLog{Id: v.ID, LabelId: v.LabelID, OpType: api.LabelLogOpType(v.OpType), Reason: v.Reason, OperatorName: v.OperatorName, StaffNo: v.StaffNo, CreatedAt: v.CreatedAt, BarCode: v.BarCode, FactoryId: v.FactoryID, BatchId: v.BatchID, LotNo: v.LotNo, ProjectName: v.ProjectName, PieceNo: v.PieceNo, PieceIndex: v.PieceIndex, Qty: v.Qty, DrawingNo: v.DrawingNo, Pattern: v.Pattern, Thickness: float32(v.Thickness), Color: v.Color, PieceArea: float32(v.PieceArea)}
}
func logDate(v *types.Date) *time.Time { return dateValue(v) }
func (s *Server) ListLabelLogs(ctx context.Context, req api.ListLabelLogsRequestObject) (api.ListLabelLogsResponseObject, error) {
	q := req.Params
	page, size := pageSize(q.Page, q.Size)
	if page < 1 || size < 1 || size > 200 || q.From != nil && q.To != nil && q.From.After(q.To.Time) {
		return api.ListLabelLogs400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "分页或日期范围无效")}, nil
	}
	f := store.LabelLogFilter{From: logDate(q.From), To: logDate(q.To), BatchID: q.BatchId, Keyword: optionalString(q.Keyword), Operator: optionalString(q.Operator)}
	if q.OpType != nil {
		f.OpType = string(*q.OpType)
	}
	items, total, totals, err := s.store.ListLabelLogs(ctx, auth.FromContext(ctx).Scope, f, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListLabelLogs200JSONResponse{Items: make([]api.LabelLog, 0, len(items)), Total: total}
	out.Totals.GenerateQty = totals.GenerateQty
	out.Totals.CancelQty = totals.CancelQty
	out.Totals.ReprintQty = totals.ReprintQty
	out.Totals.Area = float32(totals.Area)
	for _, v := range items {
		out.Items = append(out.Items, toAPILabelLog(v))
	}
	return out, nil
}
func (s *Server) ReprintLabel(ctx context.Context, req api.ReprintLabelRequestObject) (api.ReprintLabelResponseObject, error) {
	if req.Body == nil || strings.TrimSpace(req.Body.Reason) == "" {
		return api.ReprintLabel400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "请填写重打原因")}, nil
	}
	p := auth.FromContext(ctx)
	l, err := s.store.ReprintLabel(ctx, p.Scope, req.BarCode, p.UserID, strings.TrimSpace(req.Body.Reason))
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.ReprintLabel404JSONResponse(apiErr("not_found", "标签不存在")), nil
	case errors.Is(err, store.ErrLabelCancelled):
		return api.ReprintLabel409JSONResponse(apiErr("label_cancelled", "标签已作废")), nil
	case err != nil:
		return nil, err
	}
	return api.ReprintLabel200JSONResponse(toAPILabel(l)), nil
}
func (s *Server) ListAuditLogs(ctx context.Context, req api.ListAuditLogsRequestObject) (api.ListAuditLogsResponseObject, error) {
	if !adminOnly(ctx) {
		return api.ListAuditLogs403JSONResponse(forbidden()), nil
	}
	q := req.Params
	page, size := pageSize(q.Page, q.Size)
	if page < 1 || size < 1 || size > 200 || q.From != nil && q.To != nil && q.From.After(q.To.Time) {
		return api.ListAuditLogs400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "分页或日期范围无效")}, nil
	}
	f := store.AuditFilter{From: logDate(q.From), To: logDate(q.To), Keyword: optionalString(q.Keyword)}
	if q.Method != nil {
		f.Method = string(*q.Method)
	}
	if q.Result != nil {
		f.Result = *q.Result
	}
	items, total, err := s.store.ListAuditLogs(ctx, f, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListAuditLogs200JSONResponse{Items: make([]api.AuditLog, 0, len(items)), Total: total}
	for _, v := range items {
		out.Items = append(out.Items, api.AuditLog{Id: v.ID, UserId: v.UserID, Username: v.Username, DisplayName: v.DisplayName, Method: v.Method, Route: v.Route, Path: v.Path, Status: v.Status, DurationMs: v.DurationMs, Ip: v.IP, CreatedAt: v.CreatedAt})
	}
	return out, nil
}
