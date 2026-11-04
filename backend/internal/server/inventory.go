package server

import (
	"context"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/store"
	"github.com/oapi-codegen/runtime/types"
)

func (s *Server) ListInventory(ctx context.Context, req api.ListInventoryRequestObject) (api.ListInventoryResponseObject, error) {
	q := req.Params
	if (q.Kind != "semi" && q.Kind != "finished") || (q.From != nil && q.To != nil && q.From.After(q.To.Time)) {
		return api.ListInventory400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "参数或日期范围无效")}, nil
	}
	has := true
	if q.HasStock != nil {
		has = *q.HasStock
	}
	page, size := pageSize(q.Page, q.Size)
	items, total, sum, err := s.store.Inventory(ctx, auth.FromContext(ctx).Scope, store.InventoryFilter{Kind: q.Kind, From: dateValue(q.From), To: dateValue(q.To), Keyword: optionalString(q.Keyword), BatchID: q.BatchId, HasStock: has}, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListInventory200JSONResponse{Items: make([]api.InventoryRow, 0, len(items)), Total: total}
	out.Totals.StockQty, out.Totals.StockArea, out.Totals.InQty, out.Totals.OutQty = sum.StockQty, sum.StockArea, sum.InQty, sum.OutQty
	for _, r := range items {
		out.Items = append(out.Items, api.InventoryRow{BatchItemId: r.BatchItemID, BatchId: r.BatchID, LotNo: r.LotNo, ProjectName: r.ProjectName, ClientOrderNo: r.ClientOrderNo, OrderDate: types.Date{Time: r.OrderDate}, Thickness: r.Thickness, Color: r.Color, PieceNo: r.PieceNo, Seq: r.Seq, DrawingNo: r.DrawingNo, PlanQty: r.PlanQty, PieceArea: r.PieceArea, PlanArea: r.PlanArea, InQty: r.InQty, OutQty: r.OutQty, StockQty: r.StockQty, StockArea: r.StockArea})
	}
	return out, nil
}

func (s *Server) ProjectSummary(ctx context.Context, req api.ProjectSummaryRequestObject) (api.ProjectSummaryResponseObject, error) {
	q := req.Params
	status := optionalString(q.Status)
	if status == "" {
		status = "all"
	}
	if (status != "all" && status != "done" && status != "not_done") || (q.From != nil && q.To != nil && q.From.After(q.To.Time)) {
		return api.ProjectSummary400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "参数或日期范围无效")}, nil
	}
	page, size := pageSize(q.Page, q.Size)
	jobs, rows, total, sum, err := s.store.ProjectSummary(ctx, auth.FromContext(ctx).Scope, store.ProjectSummaryFilter{From: dateValue(q.From), To: dateValue(q.To), Keyword: optionalString(q.Keyword), Status: status}, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ProjectSummary200JSONResponse{Jobs: make([]api.Job, 0, len(jobs)), Items: make([]api.ProjectSummaryRow, 0, len(rows)), Total: total}
	for _, j := range jobs {
		out.Jobs = append(out.Jobs, toAPIJob(j))
	}
	out.Summary.BatchCount, out.Summary.PlanQty, out.Summary.PlanArea, out.Summary.LabelQty, out.Summary.Cells = sum.BatchCount, sum.PlanQty, sum.PlanArea, sum.LabelQty, toAPICells(sum.Cells)
	for _, r := range rows {
		out.Items = append(out.Items, api.ProjectSummaryRow{ProjectId: r.ProjectID, ProjectCode: r.ProjectCode, ProjectName: r.ProjectName, CustomerName: r.CustomerName, BatchCount: r.BatchCount, PlanQty: r.PlanQty, PlanArea: r.PlanArea, LabelQty: r.LabelQty, Status: api.BatchStatus(r.Status), Cells: toAPICells(r.Cells)})
	}
	return out, nil
}
