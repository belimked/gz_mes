package server

import (
	"context"
	"errors"
	"math"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/store"
)

func toAPIValuation(v *store.BatchValuation) api.BatchValuation {
	return api.BatchValuation{BatchId: v.BatchID, LotNo: v.LotNo, ContractId: v.ContractID, ContractNo: v.ContractNo, CustomerId: v.CustomerID, CustomerName: v.CustomerName, ProjectName: v.ProjectName, Thickness: v.Thickness, Color: v.Color, OrderQty: v.OrderQty, OrderArea: v.OrderArea, ItemArea: v.ItemArea, DeliveredQty: v.DeliveredQty, DeliveredArea: v.DeliveredArea, Status: api.BatchStatus(v.Status), Basis: api.ValuationBasis(v.Basis), BasisArea: v.BasisArea, UnitPrice: v.UnitPrice, IngotPrice: v.IngotPrice, Amount: v.Amount, Remark: v.Remark, Audited: v.Audited, AuditedBy: v.AuditedBy, AuditedAt: v.AuditedAt, UpdatedBy: v.UpdatedBy, UpdatedAt: v.UpdatedAt, CreatedAt: v.CreatedAt}
}
func (s *Server) ListBatchValuations(ctx context.Context, req api.ListBatchValuationsRequestObject) (api.ListBatchValuationsResponseObject, error) {
	f := store.BatchValuationFilter{Keyword: optionalString(req.Params.Keyword), ContractID: req.Params.ContractId, CustomerID: req.Params.CustomerId, Audited: req.Params.Audited, Begin: dateValue(req.Params.BeginDate), End: dateValue(req.Params.EndDate)}
	if f.Begin != nil && f.End != nil && f.Begin.After(*f.End) {
		return api.ListBatchValuations400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "日期范围无效")}, nil
	}
	page, size := pageSize(req.Params.Page, req.Params.Size)
	items, total, amount, err := s.store.ListBatchValuations(ctx, auth.FromContext(ctx).Scope, f, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListBatchValuations200JSONResponse{Items: make([]api.BatchValuation, 0, len(items)), Total: total, TotalAmount: amount}
	for i := range items {
		out.Items = append(out.Items, toAPIValuation(&items[i]))
	}
	return out, nil
}
func (s *Server) SetBatchValuation(ctx context.Context, req api.SetBatchValuationRequestObject) (api.SetBatchValuationResponseObject, error) {
	b := req.Body
	if b == nil || (b.Basis != "order_area" && b.Basis != "item_area" && b.Basis != "delivered_area") || math.IsNaN(b.UnitPrice) || math.IsInf(b.UnitPrice, 0) || b.UnitPrice < 0 || math.IsNaN(optionalFloat(b.IngotPrice)) || math.IsInf(optionalFloat(b.IngotPrice), 0) || optionalFloat(b.IngotPrice) < 0 {
		return api.SetBatchValuation400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "计价依据和价格无效")}, nil
	}
	p := auth.FromContext(ctx)
	v, err := s.store.SetBatchValuation(ctx, p.Scope, req.Id, store.BatchValuationInput{Basis: string(b.Basis), UnitPrice: b.UnitPrice, IngotPrice: optionalFloat(b.IngotPrice), Remark: optionalString(b.Remark)}, p.UserID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.SetBatchValuation404JSONResponse(apiErr("not_found", "批次不存在")), nil
	case errors.Is(err, store.ErrConflict):
		return api.SetBatchValuation409JSONResponse(apiErr("already_audited", "计价已审核")), nil
	case err != nil:
		return nil, err
	}
	return api.SetBatchValuation200JSONResponse(toAPIValuation(v)), nil
}
func (s *Server) AuditBatchValuation(ctx context.Context, req api.AuditBatchValuationRequestObject) (api.AuditBatchValuationResponseObject, error) {
	p := auth.FromContext(ctx)
	v, err := s.store.AuditBatchValuation(ctx, p.Scope, req.Id, p.UserID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.AuditBatchValuation404JSONResponse(apiErr("not_found", "该批次尚未计价")), nil
	case errors.Is(err, store.ErrConflict):
		return api.AuditBatchValuation409JSONResponse(apiErr("already_audited", "计价已审核")), nil
	case err != nil:
		return nil, err
	}
	return api.AuditBatchValuation200JSONResponse(toAPIValuation(v)), nil
}
