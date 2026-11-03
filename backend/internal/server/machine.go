package server

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/store"
)

func toAPIMachine(m store.Machine) api.Machine {
	return api.Machine{Id: m.ID, FactoryId: m.FactoryID, JobId: m.JobID, Code: m.Code, Name: m.Name, FactoryName: m.FactoryName, JobName: m.JobName, JobSeq: m.JobSeq, CapacityQty: m.CapacityQty, CapacityArea: float32(m.CapacityArea), IsStop: m.IsStop, Remark: &m.Remark, CreatedAt: m.CreatedAt}
}
func machineInput(b *api.MachineInput) store.MachineInput {
	return store.MachineInput{FactoryID: b.FactoryId, JobID: b.JobId, Code: strings.TrimSpace(b.Code), Name: strings.TrimSpace(b.Name), CapacityQty: b.CapacityQty, CapacityArea: float64(b.CapacityArea), IsStop: b.IsStop, Remark: optionalString(b.Remark)}
}
func invalidMachineInput(in store.MachineInput) bool {
	return in.FactoryID <= 0 || in.JobID <= 0 || in.Code == "" || in.Name == "" || in.CapacityQty < 0 || in.CapacityArea < 0
}
func (s *Server) ListMachines(ctx context.Context, req api.ListMachinesRequestObject) (api.ListMachinesResponseObject, error) {
	if req.Params.FactoryId != nil {
		ok, err := s.store.OwnFactoryInScope(ctx, auth.FromContext(ctx).Scope, *req.Params.FactoryId)
		if err != nil {
			return nil, err
		}
		if !ok {
			return api.ListMachines400JSONResponse{ErrorJSONResponse: apiErr("invalid_factory", "厂区必须是授权的自有厂区")}, nil
		}
	}
	stopped := req.Params.IncludeStopped != nil && *req.Params.IncludeStopped
	machines, err := s.store.Machines(ctx, auth.FromContext(ctx).Scope, req.Params.FactoryId, req.Params.JobId, stopped)
	if err != nil {
		return nil, err
	}
	out := make(api.ListMachines200JSONResponse, 0, len(machines))
	for _, m := range machines {
		out = append(out, toAPIMachine(m))
	}
	return out, nil
}
func (s *Server) CreateMachine(ctx context.Context, req api.CreateMachineRequestObject) (api.CreateMachineResponseObject, error) {
	in := machineInput(req.Body)
	if invalidMachineInput(in) {
		return api.CreateMachine400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "机台信息无效")}, nil
	}
	m, err := s.store.SaveMachine(ctx, auth.FromContext(ctx).Scope, nil, in)
	switch {
	case errors.Is(err, store.ErrBadRef):
		return api.CreateMachine400JSONResponse{ErrorJSONResponse: apiErr("invalid_reference", "只能在授权的自有厂区选择有效工序")}, nil
	case errors.Is(err, store.ErrConflict):
		return api.CreateMachine409JSONResponse(apiErr("duplicate_code", "机台编码已存在")), nil
	case err != nil:
		return nil, err
	}
	return api.CreateMachine201JSONResponse(toAPIMachine(*m)), nil
}
func (s *Server) UpdateMachine(ctx context.Context, req api.UpdateMachineRequestObject) (api.UpdateMachineResponseObject, error) {
	in := machineInput(req.Body)
	if invalidMachineInput(in) {
		return api.UpdateMachine400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "机台信息无效")}, nil
	}
	m, err := s.store.SaveMachine(ctx, auth.FromContext(ctx).Scope, &req.Id, in)
	switch {
	case errors.Is(err, store.ErrBadRef):
		return api.UpdateMachine400JSONResponse{ErrorJSONResponse: apiErr("invalid_reference", "只能在授权的自有厂区选择有效工序")}, nil
	case errors.Is(err, store.ErrNotFound):
		return api.UpdateMachine404JSONResponse(apiErr("not_found", "机台不存在")), nil
	case errors.Is(err, store.ErrConflict):
		return api.UpdateMachine409JSONResponse(apiErr("duplicate_code", "机台编码已存在")), nil
	case err != nil:
		return nil, err
	}
	return api.UpdateMachine200JSONResponse(toAPIMachine(*m)), nil
}
func (s *Server) GetMachineOutputReport(ctx context.Context, req api.GetMachineOutputReportRequestObject) (api.GetMachineOutputReportResponseObject, error) {
	from, to := time.Now().Truncate(24*time.Hour), time.Now().Truncate(24*time.Hour)
	if req.Params.From != nil {
		from = req.Params.From.Time
	}
	if req.Params.To != nil {
		to = req.Params.To.Time
	}
	page, size := pageSize(req.Params.Page, req.Params.Size)
	if from.After(to) || page < 1 || size < 1 || size > 200 {
		return api.GetMachineOutputReport400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "日期范围或分页参数无效")}, nil
	}
	mode := "summary"
	if req.Params.Mode != nil {
		mode = string(*req.Params.Mode)
	}
	rows, total, summary, err := s.store.MachineOutput(ctx, auth.FromContext(ctx).Scope, store.MachineOutputFilter{From: from, To: to, FactoryID: req.Params.FactoryId, MachineID: req.Params.MachineId, JobID: req.Params.JobId, Keyword: optionalString(req.Params.Keyword), Mode: mode}, page, size)
	if err != nil {
		return nil, err
	}
	out := api.GetMachineOutputReport200JSONResponse{Items: make([]api.MachineOutputRow, 0, len(rows)), Total: total}
	out.Summary.Machines = summary.Machines
	out.Summary.Qty = summary.Qty
	out.Summary.PieceArea = float32(summary.PieceArea)
	out.Summary.SprayingArea = float32(summary.SprayingArea)
	out.Summary.UnspecifiedQty = summary.UnspecifiedQty
	for _, r := range rows {
		v := api.MachineOutputRow{MachineId: r.MachineID, Code: r.Code, Name: r.Name, FactoryName: r.FactoryName, JobName: r.JobName, BatchId: r.BatchID, LotNo: r.LotNo, ProjectName: r.ProjectName, Qty: r.Qty, PieceArea: float32(r.PieceArea), SprayingArea: float32(r.SprayingArea), Days: r.Days, CapacityQty: r.CapacityQty}
		convert := func(n *float64) *float32 {
			if n == nil {
				return nil
			}
			v := float32(*n)
			return &v
		}
		v.CapacityArea = convert(r.CapacityArea)
		v.QtyLoad = convert(r.QtyLoad)
		v.AreaLoad = convert(r.AreaLoad)
		v.LoadFactor = convert(r.LoadFactor)
		out.Items = append(out.Items, v)
	}
	return out, nil
}
