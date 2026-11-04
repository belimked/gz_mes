package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/oapi-codegen/runtime/types"
)

func boardOptionalDate(t *time.Time) *types.Date {
	if t == nil {
		return nil
	}
	d := types.Date{Time: *t}
	return &d
}

func (s *Server) WorkOrderBoard(ctx context.Context, req api.WorkOrderBoardRequestObject) (api.WorkOrderBoardResponseObject, error) {
	q := req.Params
	if (q.Page != nil && *q.Page < 1) || (q.Size != nil && (*q.Size < 1 || *q.Size > 50)) {
		return api.WorkOrderBoard400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "页码或每页条数无效")}, nil
	}
	page, size := 1, 10
	if q.Page != nil {
		page = *q.Page
	}
	if q.Size != nil {
		size = *q.Size
	}
	status := ""
	if q.Status != nil {
		status = string(*q.Status)
		if status != "pending" && status != "scheduled" && status != "in_production" {
			return api.WorkOrderBoard400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "批次状态无效")}, nil
		}
	}
	today := boardDate(nil)
	jobs, rows, total, err := s.store.WorkOrders(ctx, auth.FromContext(ctx).Scope, status, optionalString(q.Keyword), page, size, today)
	if err != nil {
		return nil, err
	}
	out := api.WorkOrderBoard200JSONResponse{Items: []api.WorkOrderRow{}, Jobs: []api.Job{}, Total: total}
	for _, j := range jobs {
		out.Jobs = append(out.Jobs, api.Job{Id: j.ID, Code: j.Code, Name: j.Name, Seq: j.Seq, Stage: api.JobStage(j.Stage), IsQc: j.IsQc})
	}
	for _, r := range rows {
		state := api.WorkOrderRowPrintState("partial")
		if r.LabelQty == 0 {
			state = "none"
		} else if r.LabelQty >= r.ItemQty {
			state = "all"
		}
		row := api.WorkOrderRow{BatchId: r.BatchID, LotNo: r.LotNo, ProjectName: r.ProjectName, CustomerName: r.CustomerName, Workshop: r.Workshop, Urgency: api.WorkOrderRowUrgency(r.Urgency), Status: api.BatchStatus(r.Status), OrderDate: types.Date{Time: r.OrderDate}, RequiredDate: boardOptionalDate(r.RequiredDate), PlanCutDate: boardOptionalDate(r.PlanCutDate), ItemQty: r.ItemQty, ItemArea: float32(r.ItemArea), LabelQty: r.LabelQty, Overdue: r.RequiredDate != nil && r.RequiredDate.Before(today), PrintState: state, Jobs: []api.WorkOrderJob{}}
		for i, c := range r.Jobs {
			row.Jobs = append(row.Jobs, api.WorkOrderJob{JobId: c.JobID, Code: jobs[i].Code, Name: jobs[i].Name, Qty: c.Qty, Pct: c.Pct})
		}
		out.Items = append(out.Items, row)
	}
	return out, nil
}
func (s *Server) CockpitBoard(ctx context.Context, _ api.CockpitBoardRequestObject) (api.CockpitBoardResponseObject, error) {
	data, err := s.store.Cockpit(ctx, auth.FromContext(ctx).Scope, boardDate(nil))
	if err != nil {
		return nil, err
	}
	// JSON normalization maps the store's aggregate to the generated wire types.
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var out api.CockpitBoard200JSONResponse
	if err = json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
