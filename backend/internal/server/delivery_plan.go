package server

import (
	"context"
	"errors"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/store"
	"github.com/oapi-codegen/runtime/types"
)

func planAPI(v *store.DeliveryPlan) api.DeliveryPlan {
	x := api.DeliveryPlan{Id: v.ID, PlanNo: v.PlanNo, FactoryId: v.FactoryID, LogisticsCompany: v.LogisticsCompany, Contact: v.Contact, Phone: v.Phone, ShippingAddress: v.ShippingAddress, ReceivingAddress: v.ReceivingAddress, CarModel: v.CarModel, PlateNo: v.PlateNo, Driver: v.Driver, DriverPhone: v.DriverPhone, Cost: v.Cost, ExtraCost: v.ExtraCost, EstimatedDepartureDate: types.Date{Time: v.EstimatedDepartureDate}, ConfirmedBy: v.ConfirmedBy, ConfirmedAt: v.ConfirmedAt, Remark: v.Remark, CreatedAt: v.CreatedAt, NoteCount: v.NoteCount, Qty: v.Qty, Area: v.Area}
	if v.ConfirmedDepartureDate != nil {
		x.ConfirmedDepartureDate = &types.Date{Time: *v.ConfirmedDepartureDate}
	}
	return x
}
func planDetail(v *store.DeliveryPlan, notes []store.DeliveryNote) api.DeliveryPlanDetail {
	x := planAPI(v)
	d := api.DeliveryPlanDetail{Id: x.Id, PlanNo: x.PlanNo, FactoryId: x.FactoryId, LogisticsCompany: x.LogisticsCompany, Contact: x.Contact, Phone: x.Phone, ShippingAddress: x.ShippingAddress, ReceivingAddress: x.ReceivingAddress, CarModel: x.CarModel, PlateNo: x.PlateNo, Driver: x.Driver, DriverPhone: x.DriverPhone, Cost: x.Cost, ExtraCost: x.ExtraCost, EstimatedDepartureDate: x.EstimatedDepartureDate, ConfirmedDepartureDate: x.ConfirmedDepartureDate, ConfirmedBy: x.ConfirmedBy, ConfirmedAt: x.ConfirmedAt, Remark: x.Remark, CreatedAt: x.CreatedAt, NoteCount: x.NoteCount, Qty: x.Qty, Area: x.Area, Notes: make([]api.DeliveryNote, 0, len(notes))}
	for i := range notes {
		d.Notes = append(d.Notes, toAPINote(&notes[i]))
	}
	return d
}
func planInput(b *api.DeliveryPlanInput) store.DeliveryPlanInput {
	return store.DeliveryPlanInput{FactoryID: b.FactoryId, LogisticsCompany: optionalString(b.LogisticsCompany), Contact: optionalString(b.Contact), Phone: optionalString(b.Phone), ShippingAddress: optionalString(b.ShippingAddress), ReceivingAddress: optionalString(b.ReceivingAddress), CarModel: optionalString(b.CarModel), PlateNo: optionalString(b.PlateNo), Driver: optionalString(b.Driver), DriverPhone: optionalString(b.DriverPhone), Cost: optionalFloat(b.Cost), ExtraCost: optionalFloat(b.ExtraCost), EstimatedDepartureDate: b.EstimatedDepartureDate.Time, Remark: optionalString(b.Remark), NoteIDs: b.NoteIds}
}
func (s *Server) ListDeliveryPlans(ctx context.Context, req api.ListDeliveryPlansRequestObject) (api.ListDeliveryPlansResponseObject, error) {
	q := req.Params
	page, size := pageSize(q.Page, q.Size)
	items, sum, err := s.store.ListDeliveryPlans(ctx, auth.FromContext(ctx).Scope, store.DeliveryPlanFilter{CreatedFrom: dateValue(q.CreatedFrom), CreatedTo: dateValue(q.CreatedTo), EstimatedFrom: dateValue(q.EstimatedFrom), EstimatedTo: dateValue(q.EstimatedTo), Confirmed: q.Confirmed, Keyword: optionalString(q.Keyword)}, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListDeliveryPlans200JSONResponse{Items: make([]api.DeliveryPlan, 0, len(items)), Total: int64(sum.Plans)}
	out.Summary.Plans, out.Summary.Confirmed, out.Summary.Cost, out.Summary.ExtraCost, out.Summary.Qty, out.Summary.Area = sum.Plans, sum.Confirmed, sum.Cost, sum.ExtraCost, sum.Qty, sum.Area
	for i := range items {
		out.Items = append(out.Items, planAPI(&items[i]))
	}
	return out, nil
}
func (s *Server) CreateDeliveryPlan(ctx context.Context, req api.CreateDeliveryPlanRequestObject) (api.CreateDeliveryPlanResponseObject, error) {
	if req.Body == nil {
		return api.CreateDeliveryPlan400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "计划不能为空")}, nil
	}
	p := auth.FromContext(ctx)
	v, notes, err := s.store.SaveDeliveryPlan(ctx, p.Scope, 0, planInput(req.Body), p.UserID)
	if errors.Is(err, store.ErrBadRef) || errors.Is(err, store.ErrNotFound) {
		return api.CreateDeliveryPlan400JSONResponse{ErrorJSONResponse: apiErr("invalid_reference", docMessage(err))}, nil
	}
	if err != nil {
		return nil, err
	}
	return api.CreateDeliveryPlan201JSONResponse(planDetail(v, notes)), nil
}
func (s *Server) GetDeliveryPlan(ctx context.Context, req api.GetDeliveryPlanRequestObject) (api.GetDeliveryPlanResponseObject, error) {
	v, n, err := s.store.GetDeliveryPlan(ctx, auth.FromContext(ctx).Scope, req.Id)
	if errors.Is(err, store.ErrNotFound) {
		return api.GetDeliveryPlan404JSONResponse{ErrorJSONResponse: apiErr("not_found", "计划不存在")}, nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetDeliveryPlan200JSONResponse(planDetail(v, n)), nil
}
func (s *Server) UpdateDeliveryPlan(ctx context.Context, req api.UpdateDeliveryPlanRequestObject) (api.UpdateDeliveryPlanResponseObject, error) {
	if req.Body == nil {
		return api.UpdateDeliveryPlan400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "计划不能为空")}, nil
	}
	p := auth.FromContext(ctx)
	v, n, err := s.store.SaveDeliveryPlan(ctx, p.Scope, req.Id, planInput(req.Body), p.UserID)
	if errors.Is(err, store.ErrBadRef) {
		return api.UpdateDeliveryPlan400JSONResponse{ErrorJSONResponse: apiErr("invalid_reference", docMessage(err))}, nil
	}
	if errors.Is(err, store.ErrNotFound) {
		return api.UpdateDeliveryPlan404JSONResponse(apiErr("not_found", "计划不存在")), nil
	}
	if errors.Is(err, store.ErrConflict) {
		return api.UpdateDeliveryPlan409JSONResponse(apiErr("already_confirmed", "计划已确认")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.UpdateDeliveryPlan200JSONResponse(planDetail(v, n)), nil
}
func (s *Server) ConfirmDeliveryPlan(ctx context.Context, req api.ConfirmDeliveryPlanRequestObject) (api.ConfirmDeliveryPlanResponseObject, error) {
	if req.Body == nil || req.Body.DepartureDate.IsZero() {
		return api.ConfirmDeliveryPlan400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "发车日期不能为空")}, nil
	}
	p := auth.FromContext(ctx)
	v, n, err := s.store.ConfirmDeliveryPlan(ctx, p.Scope, req.Id, p.UserID, req.Body.DepartureDate.Time)
	if errors.Is(err, store.ErrBadRef) {
		return api.ConfirmDeliveryPlan400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "至少关联一张发货单")}, nil
	}
	if errors.Is(err, store.ErrNotFound) {
		return api.ConfirmDeliveryPlan404JSONResponse(apiErr("not_found", "计划不存在")), nil
	}
	if errors.Is(err, store.ErrConflict) {
		return api.ConfirmDeliveryPlan409JSONResponse(apiErr("already_confirmed", "计划已确认")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.ConfirmDeliveryPlan200JSONResponse(planDetail(v, n)), nil
}
func (s *Server) DeleteDeliveryPlan(ctx context.Context, req api.DeleteDeliveryPlanRequestObject) (api.DeleteDeliveryPlanResponseObject, error) {
	err := s.store.DeleteDeliveryPlan(ctx, auth.FromContext(ctx).Scope, req.Id)
	if errors.Is(err, store.ErrNotFound) {
		return api.DeleteDeliveryPlan404JSONResponse{ErrorJSONResponse: apiErr("not_found", "计划不存在")}, nil
	}
	if errors.Is(err, store.ErrConflict) {
		return api.DeleteDeliveryPlan409JSONResponse(apiErr("already_confirmed", "计划已确认")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.DeleteDeliveryPlan204Response{}, nil
}
