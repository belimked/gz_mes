package server

// The generated BatchSchedule uses pointers for nullable fields, so JSON null and
// omission are indistinguishable. Nil dates and outsourceFactoryId mean unchanged;
// clearing them is not supported in this version.

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

func dateValue(d *types.Date) *time.Time {
	if d == nil {
		return nil
	}
	t := d.Time
	return &t
}
func apiDate(t *time.Time) *types.Date {
	if t == nil {
		return nil
	}
	return &types.Date{Time: *t}
}
func optionalFloat(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}
func optionalInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
func (s *Server) ListJobs(ctx context.Context, _ api.ListJobsRequestObject) (api.ListJobsResponseObject, error) {
	jobs, err := s.store.Jobs(ctx)
	if err != nil {
		return nil, err
	}
	out := make(api.ListJobs200JSONResponse, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, toAPIJob(j))
	}
	return out, nil
}
func toAPIBatch(b *store.Batch) api.Batch {
	return api.Batch{Id: b.ID, FactoryId: b.FactoryID, ContractId: b.ContractID, ContractNo: b.ContractNo, CustomerId: b.CustomerID, CustomerName: b.CustomerName, ProjectId: b.ProjectID, ProjectName: b.ProjectName, LotNo: b.LotNo, ClientOrderNo: b.ClientOrderNo, Thickness: b.Thickness, Color: b.Color, ColorNo: b.ColorNo, Texture: b.Texture, SprayingType: b.SprayingType, OrderQty: b.OrderQty, OrderArea: b.OrderArea, DeliveryAddress: b.DeliveryAddress, SpecialRequest: b.SpecialRequest, RequiredDate: apiDate(b.RequiredDate), Urgency: api.Urgency(b.Urgency), Status: api.BatchStatus(b.Status), ItemCount: b.ItemCount, ItemQty: b.ItemQty, ItemArea: b.ItemArea, PlanTechDate: apiDate(b.PlanTechDate), PlanDrawingDate: apiDate(b.PlanDrawingDate), PlanCutDate: apiDate(b.PlanCutDate), PlanSheetStart: apiDate(b.PlanSheetStart), PlanSheetDelivery: apiDate(b.PlanSheetDelivery), PlanSprayStart: apiDate(b.PlanSprayStart), PlanSprayDelivery: apiDate(b.PlanSprayDelivery), OutsourceFactoryId: b.OutsourceFactoryID, OutsourceFactoryName: b.OutsourceFactoryName, Workshop: b.Workshop, Remark: b.Remark, CreatedBy: b.CreatedBy, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt}
}
func (s *Server) ListBatches(ctx context.Context, req api.ListBatchesRequestObject) (api.ListBatchesResponseObject, error) {
	page, size := 1, 20
	if req.Params.Page != nil {
		page = *req.Params.Page
	}
	if req.Params.Size != nil {
		size = *req.Params.Size
	}
	f := store.BatchFilter{ContractID: req.Params.ContractId, ProjectID: req.Params.ProjectId, CustomerID: req.Params.CustomerId}
	if req.Params.Keyword != nil {
		f.Keyword = *req.Params.Keyword
	}
	if req.Params.Status != nil {
		f.Status = string(*req.Params.Status)
	}
	if req.Params.Urgency != nil {
		f.Urgency = string(*req.Params.Urgency)
	}
	if req.Params.BeginDate != nil {
		f.Begin = dateValue(req.Params.BeginDate)
	}
	if req.Params.EndDate != nil {
		f.End = dateValue(req.Params.EndDate)
	}
	if f.Begin != nil && f.End != nil && f.Begin.After(*f.End) {
		return api.ListBatches400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "日期范围无效")}, nil
	}
	items, total, err := s.store.ListBatches(ctx, auth.FromContext(ctx).Scope, f, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListBatches200JSONResponse{Items: make([]api.Batch, 0, len(items)), Total: total}
	for i := range items {
		out.Items = append(out.Items, toAPIBatch(&items[i]))
	}
	return out, nil
}
func (s *Server) CreateBatch(ctx context.Context, req api.CreateBatchRequestObject) (api.CreateBatchResponseObject, error) {
	p := auth.FromContext(ctx)
	b := req.Body
	in := store.BatchCreate{ContractID: b.ContractId, LotNo: strings.TrimSpace(b.LotNo), ClientOrderNo: optionalString(b.ClientOrderNo), Thickness: optionalFloat(b.Thickness), Color: optionalString(b.Color), ColorNo: optionalString(b.ColorNo), Texture: optionalString(b.Texture), SprayingType: optionalString(b.SprayingType), OrderQty: optionalInt(b.OrderQty), OrderArea: optionalFloat(b.OrderArea), DeliveryAddress: optionalString(b.DeliveryAddress), SpecialRequest: optionalString(b.SpecialRequest), RequiredDate: dateValue(b.RequiredDate), Urgency: "normal", Remark: optionalString(b.Remark), CreatedBy: p.UserID}
	if b.Urgency != nil {
		in.Urgency = string(*b.Urgency)
	}
	if in.LotNo == "" {
		return api.CreateBatch400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "批次号不能为空")}, nil
	}
	item, err := s.store.CreateBatch(ctx, p.Scope, in)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.CreateBatch404JSONResponse(apiErr("not_found", "合同不存在")), nil
	case errors.Is(err, store.ErrConflict):
		return api.CreateBatch409JSONResponse(apiErr("duplicate_code", "批次号已存在")), nil
	case err != nil:
		return nil, err
	}
	return api.CreateBatch201JSONResponse(toAPIBatch(item)), nil
}
func (s *Server) GetBatch(ctx context.Context, req api.GetBatchRequestObject) (api.GetBatchResponseObject, error) {
	b, err := s.store.GetBatch(ctx, auth.FromContext(ctx).Scope, req.Id)
	if errors.Is(err, store.ErrNotFound) {
		return api.GetBatch404JSONResponse(apiErr("not_found", "批次不存在")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetBatch200JSONResponse(toAPIBatch(b)), nil
}
func scheduleDate(v *types.Date) *store.Optional[*time.Time] {
	if v == nil {
		return nil
	}
	return &store.Optional[*time.Time]{Value: dateValue(v)}
}
func (s *Server) ScheduleBatch(ctx context.Context, req api.ScheduleBatchRequestObject) (api.ScheduleBatchResponseObject, error) {
	b := req.Body
	in := store.BatchSchedule{PlanTechDate: scheduleDate(b.PlanTechDate), PlanDrawingDate: scheduleDate(b.PlanDrawingDate), PlanCutDate: scheduleDate(b.PlanCutDate), PlanSheetStart: scheduleDate(b.PlanSheetStart), PlanSheetDelivery: scheduleDate(b.PlanSheetDelivery), PlanSprayStart: scheduleDate(b.PlanSprayStart), PlanSprayDelivery: scheduleDate(b.PlanSprayDelivery)}
	if b.OutsourceFactoryId != nil {
		in.OutsourceFactoryID = &store.Optional[*int64]{Value: b.OutsourceFactoryId}
	}
	if b.Workshop != nil {
		in.Workshop = &store.Optional[string]{Value: *b.Workshop}
	}
	if b.Remark != nil {
		in.Remark = &store.Optional[string]{Value: *b.Remark}
	}
	if b.Urgency != nil {
		in.Urgency = &store.Optional[string]{Value: string(*b.Urgency)}
	}
	item, err := s.store.ScheduleBatch(ctx, auth.FromContext(ctx).Scope, req.Id, in)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.ScheduleBatch404JSONResponse(apiErr("not_found", "批次不存在")), nil
	case errors.Is(err, store.ErrConflict):
		return api.ScheduleBatch409JSONResponse(apiErr("batch_done", "批次已完工，不能修改")), nil
	case errors.Is(err, store.ErrBadRef):
		return api.ScheduleBatch400JSONResponse{ErrorJSONResponse: apiErr("invalid_reference", "外协厂不存在或不是外协类型")}, nil
	case err != nil:
		return nil, err
	}
	return api.ScheduleBatch200JSONResponse(toAPIBatch(item)), nil
}
func toAPIBatchItem(i store.BatchItem) api.BatchItem {
	return api.BatchItem{Id: i.ID, BatchId: i.BatchID, Seq: i.Seq, PieceNo: i.PieceNo, DrawingNo: i.DrawingNo, Pattern: i.Pattern, Thickness: i.Thickness, Color: i.Color, Length: i.Length, Width: i.Width, PieceArea: i.PieceArea, SprayingArea: i.SprayingArea, Qty: i.Qty, PackageQty: i.PackageQty, WeldLine: i.WeldLine, BendingTimes: i.BendingTimes, Remark: i.Remark}
}
func toAPIItems(items []store.BatchItem) []api.BatchItem {
	out := make([]api.BatchItem, 0, len(items))
	for _, i := range items {
		out = append(out, toAPIBatchItem(i))
	}
	return out
}
func (s *Server) ListBatchItems(ctx context.Context, req api.ListBatchItemsRequestObject) (api.ListBatchItemsResponseObject, error) {
	items, err := s.store.ListBatchItems(ctx, auth.FromContext(ctx).Scope, req.Id)
	if errors.Is(err, store.ErrNotFound) {
		return api.ListBatchItems404JSONResponse(apiErr("not_found", "批次不存在")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.ListBatchItems200JSONResponse(toAPIItems(items)), nil
}
func (s *Server) ReplaceBatchItems(ctx context.Context, req api.ReplaceBatchItemsRequestObject) (api.ReplaceBatchItemsResponseObject, error) {
	in := make([]store.BatchItemInput, 0, len(*req.Body))
	for _, v := range *req.Body {
		in = append(in, store.BatchItemInput{PieceNo: v.PieceNo, DrawingNo: optionalString(v.DrawingNo), Pattern: optionalString(v.Pattern), Thickness: optionalFloat(v.Thickness), Color: optionalString(v.Color), Length: optionalFloat(v.Length), Width: optionalFloat(v.Width), PieceArea: optionalFloat(v.PieceArea), SprayingArea: optionalFloat(v.SprayingArea), Qty: v.Qty, PackageQty: optionalInt(v.PackageQty), WeldLine: optionalFloat(v.WeldLine), BendingTimes: optionalInt(v.BendingTimes), Remark: optionalString(v.Remark)})
	}
	items, err := s.store.ReplaceBatchItems(ctx, auth.FromContext(ctx).Scope, req.Id, in)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.ReplaceBatchItems404JSONResponse(apiErr("not_found", "批次不存在")), nil
	case errors.Is(err, store.ErrConflict):
		return api.ReplaceBatchItems409JSONResponse(apiErr("batch_done", "批次已完工，不能修改")), nil
	case errors.Is(err, store.ErrHasLabels):
		return api.ReplaceBatchItems409JSONResponse(apiErr("has_labels", "已生成标签，不能再替换构件清单")), nil
	case errors.Is(err, store.ErrDuplicateItem):
		return api.ReplaceBatchItems400JSONResponse{ErrorJSONResponse: apiErr("duplicate_piece_no", "构件编号重复: "+strings.TrimPrefix(err.Error(), store.ErrDuplicateItem.Error()+": "))}, nil
	case err != nil:
		return nil, err
	}
	return api.ReplaceBatchItems200JSONResponse(toAPIItems(items)), nil
}
