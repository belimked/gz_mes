package server

import (
	"context"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/store"
)

func toProgress(p store.OutsourceProgress) api.OutsourceProgress {
	return api.OutsourceProgress{ScannedQty: p.ScannedQty, ScannedArea: p.ScannedArea, UnscannedQty: p.UnscannedQty, UnscannedArea: p.UnscannedArea, Percent: p.Percent}
}
func (s *Server) GetOutsourceOrders(ctx context.Context, req api.GetOutsourceOrdersRequestObject) (api.GetOutsourceOrdersResponseObject, error) {
	q := req.Params
	from, to := dateValue(q.From), dateValue(q.To)
	if from != nil && to != nil && from.After(*to) {
		return api.GetOutsourceOrders400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "日期范围无效")}, nil
	}
	page, size := pageSize(q.Page, q.Size)
	items, sum, err := s.store.OutsourceOrders(ctx, auth.FromContext(ctx).Scope, store.OutsourceFilter{From: from, To: to, FactoryID: q.OutsourceFactoryId, Keyword: optionalString(q.Keyword)}, page, size)
	if err != nil {
		return nil, err
	}
	out := api.GetOutsourceOrders200JSONResponse{Items: make([]api.OutsourceOrder, 0, len(items)), Total: int64(sum.Batches)}
	out.Summary.Batches, out.Summary.PlannedQty, out.Summary.PlannedArea = sum.Batches, sum.PlannedQty, sum.PlannedArea
	out.Summary.CuttingScannedQty, out.Summary.ShippingScannedQty = sum.CuttingScannedQty, sum.ShippingScannedQty
	for _, v := range items {
		out.Items = append(out.Items, api.OutsourceOrder{BatchId: v.BatchID, LotNo: v.LotNo, ProjectName: v.ProjectName, CustomerName: v.CustomerName, ClientOrderNo: v.ClientOrderNo, CreatedAt: v.CreatedAt, Thickness: v.Thickness, Color: v.Color, SprayingType: v.SprayingType, OutsourceFactoryId: v.OutsourceFactoryID, OutsourceFactoryName: v.OutsourceFactoryName, Status: v.Status, ItemQty: v.ItemQty, ItemArea: v.ItemArea, Cutting: toProgress(v.Cutting()), Shipping: toProgress(v.Shipping())})
	}
	return out, nil
}
