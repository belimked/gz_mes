package server

import (
	"context"
	"errors"
	"strings"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/store"
	"github.com/oapi-codegen/runtime/types"
)

func toAPIPlan(p *store.Plan) api.Plan {
	out := api.Plan{Id: p.ID, Kind: api.PlanKind(p.Kind), FactoryId: p.FactoryID, BatchId: p.BatchID, LotNo: p.LotNo, ProjectName: p.ProjectName, CustomerName: p.CustomerName, Thickness: p.Thickness, Color: p.Color, OrderQty: p.OrderQty, OrderArea: p.OrderArea, ItemQty: p.ItemQty, ItemArea: p.ItemArea, LabelQty: p.LabelQty,
		StartDate: types.Date{Time: p.StartDate}, EndDate: types.Date{Time: p.EndDate}, Workshop: p.Workshop, JobId: p.JobID, JobName: p.JobName, PaintLine: p.PaintLine, Shift: p.Shift, Urgency: api.Urgency(p.Urgency), AuditState: api.AuditState(p.AuditState), AuditedBy: p.AuditedBy, AuditedAt: p.AuditedAt,
		StopState: api.StopState(p.StopState), Remark: p.Remark, TargetJobId: p.TargetJobID, TargetJobName: p.TargetJobName, DoneQty: p.DoneQty, DoneArea: p.DoneArea, Progress: p.Progress, LastScanAt: p.LastScanAt, Overdue: p.Overdue, CreatedBy: p.CreatedBy, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
	sprayType, stopReason := p.SprayType, p.StopReason
	out.SprayType, out.StopReason = &sprayType, &stopReason
	return out
}

func optionalTrimmed(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	return &v
}

func invalidPlanMessage(err error) string {
	return strings.TrimPrefix(err.Error(), store.ErrInvalidPlan.Error()+": ")
}

func (s *Server) ListPlans(ctx context.Context, req api.ListPlansRequestObject) (api.ListPlansResponseObject, error) {
	q := req.Params
	f := store.PlanFilter{Kind: string(q.Kind), From: dateValue(q.From), To: dateValue(q.To), Workshop: optionalString(q.Workshop), PaintLine: optionalString(q.PaintLine), Keyword: optionalString(q.Keyword), JobID: q.JobId, Completed: q.Completed}
	if f.From != nil && f.To != nil && f.From.After(*f.To) {
		return api.ListPlans400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "日期范围无效")}, nil
	}
	if q.AuditState != nil {
		f.AuditState = string(*q.AuditState)
	}
	if q.StopState != nil {
		f.StopState = string(*q.StopState)
	}
	page, size := pageSize(q.Page, q.Size)
	items, sum, err := s.store.ListPlans(ctx, auth.FromContext(ctx).Scope, f, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListPlans200JSONResponse{Items: make([]api.Plan, 0, len(items)), Total: sum.Plans, Page: page, Size: size}
	out.Summary.Plans, out.Summary.Qty, out.Summary.Area, out.Summary.DoneQty, out.Summary.DoneArea, out.Summary.Overdue = sum.Plans, sum.Qty, sum.Area, sum.DoneQty, sum.DoneArea, sum.Overdue
	for i := range items {
		out.Items = append(out.Items, toAPIPlan(&items[i]))
	}
	return out, nil
}

func (s *Server) CreatePlan(ctx context.Context, req api.CreatePlanRequestObject) (api.CreatePlanResponseObject, error) {
	p := auth.FromContext(ctx)
	b := req.Body
	in := store.PlanCreate{Kind: string(b.Kind), BatchID: b.BatchId, StartDate: b.StartDate.Time, EndDate: b.EndDate.Time, Workshop: optionalString(b.Workshop), JobID: b.JobId, PaintLine: optionalString(b.PaintLine), Shift: optionalString(b.Shift), SprayType: optionalString(b.SprayType), Urgency: "normal", Remark: optionalString(b.Remark), CreatedBy: p.UserID}
	if b.Urgency != nil {
		in.Urgency = string(*b.Urgency)
	}
	item, err := s.store.CreatePlan(ctx, p.Scope, in)
	switch {
	case errors.Is(err, store.ErrInvalidPlan):
		return api.CreatePlan400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", invalidPlanMessage(err))}, nil
	case errors.Is(err, store.ErrNotFound):
		return api.CreatePlan404JSONResponse(apiErr("not_found", "批次不存在")), nil
	case errors.Is(err, store.ErrForbidden):
		return api.CreatePlan403JSONResponse(apiErr("forbidden_factory", "无权在该厂区创建数据")), nil
	case errors.Is(err, store.ErrDuplicatePlan):
		return api.CreatePlan409JSONResponse(apiErr("duplicate_plan", "该批次在此日期范围内已有计划")), nil
	case err != nil:
		return nil, err
	}
	return api.CreatePlan201JSONResponse(toAPIPlan(item)), nil
}

func (s *Server) GetPlan(ctx context.Context, req api.GetPlanRequestObject) (api.GetPlanResponseObject, error) {
	item, err := s.store.GetPlan(ctx, auth.FromContext(ctx).Scope, req.Id)
	if errors.Is(err, store.ErrNotFound) {
		return api.GetPlan404JSONResponse{ErrorJSONResponse: apiErr("not_found", "计划不存在")}, nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetPlan200JSONResponse(toAPIPlan(item)), nil
}

func (s *Server) UpdatePlan(ctx context.Context, req api.UpdatePlanRequestObject) (api.UpdatePlanResponseObject, error) {
	b := req.Body
	in := store.PlanUpdate{StartDate: b.StartDate.Time, EndDate: b.EndDate.Time, Workshop: optionalTrimmed(b.Workshop), PaintLine: optionalTrimmed(b.PaintLine), Shift: optionalTrimmed(b.Shift), SprayType: optionalTrimmed(b.SprayType), JobID: b.JobId, Remark: optionalTrimmed(b.Remark)}
	if b.Urgency != nil {
		u := string(*b.Urgency)
		in.Urgency = &u
	}
	item, err := s.store.UpdatePlan(ctx, auth.FromContext(ctx).Scope, req.Id, in)
	switch {
	case errors.Is(err, store.ErrInvalidPlan):
		return api.UpdatePlan400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", invalidPlanMessage(err))}, nil
	case errors.Is(err, store.ErrNotFound):
		return api.UpdatePlan404JSONResponse(apiErr("not_found", "计划不存在")), nil
	case errors.Is(err, store.ErrAlreadyAudited):
		return api.UpdatePlan409JSONResponse(apiErr("already_audited", "计划已审核，不能修改")), nil
	case errors.Is(err, store.ErrDuplicatePlan):
		return api.UpdatePlan409JSONResponse(apiErr("duplicate_plan", "该批次在此日期范围内已有计划")), nil
	case err != nil:
		return nil, err
	}
	return api.UpdatePlan200JSONResponse(toAPIPlan(item)), nil
}

func (s *Server) DeletePlan(ctx context.Context, req api.DeletePlanRequestObject) (api.DeletePlanResponseObject, error) {
	err := s.store.DeletePlan(ctx, auth.FromContext(ctx).Scope, req.Id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.DeletePlan404JSONResponse{ErrorJSONResponse: apiErr("not_found", "计划不存在")}, nil
	case errors.Is(err, store.ErrAlreadyAudited):
		return api.DeletePlan409JSONResponse(apiErr("already_audited", "计划已审核，不能删除")), nil
	case err != nil:
		return nil, err
	}
	return api.DeletePlan204Response{}, nil
}

func (s *Server) AuditPlan(ctx context.Context, req api.AuditPlanRequestObject) (api.AuditPlanResponseObject, error) {
	p := auth.FromContext(ctx)
	item, err := s.store.AuditPlan(ctx, p.Scope, req.Id, req.Body.Action == api.Audit, p.UserID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.AuditPlan404JSONResponse{ErrorJSONResponse: apiErr("not_found", "计划不存在")}, nil
	case errors.Is(err, store.ErrAlreadyAudited):
		return api.AuditPlan409JSONResponse(apiErr("already_audited", "计划已审核")), nil
	case errors.Is(err, store.ErrNotAudited):
		return api.AuditPlan409JSONResponse(apiErr("not_audited", "计划未审核")), nil
	case err != nil:
		return nil, err
	}
	return api.AuditPlan200JSONResponse(toAPIPlan(item)), nil
}

func (s *Server) SetPlanStopState(ctx context.Context, req api.SetPlanStopStateRequestObject) (api.SetPlanStopStateResponseObject, error) {
	item, err := s.store.SetPlanStopState(ctx, auth.FromContext(ctx).Scope, req.Id, string(req.Body.Action), optionalString(req.Body.Reason))
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.SetPlanStopState404JSONResponse{ErrorJSONResponse: apiErr("not_found", "计划不存在")}, nil
	case errors.Is(err, store.ErrInvalidTransition):
		return api.SetPlanStopState409JSONResponse(apiErr("invalid_transition", "当前状态不允许该操作")), nil
	case err != nil:
		return nil, err
	}
	return api.SetPlanStopState200JSONResponse(toAPIPlan(item)), nil
}
