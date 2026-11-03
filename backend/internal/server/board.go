package server

import (
	"context"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/oapi-codegen/runtime/types"
)

func boardDate(d *types.Date) time.Time {
	if d != nil {
		return d.Time
	}
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}
func apiBoardDate(s string) types.Date {
	t, _ := time.Parse("2006-01-02", s)
	return types.Date{Time: t}
}
func (s *Server) GetSprayDailyBoard(ctx context.Context, req api.GetSprayDailyBoardRequestObject) (api.GetSprayDailyBoardResponseObject, error) {
	rows, err := s.store.DailyBoard(ctx, auth.FromContext(ctx).Scope, boardDate(req.Params.Date), "spray_daily", optionalString(req.Params.PaintLine), nil)
	if err != nil {
		return nil, err
	}
	out := api.GetSprayDailyBoard200JSONResponse{Rows: []api.SprayBoardRow{}}
	out.Summary.ByLine = []api.SprayLineSummary{}
	lines := map[string]int{}
	for _, r := range rows {
		out.Rows = append(out.Rows, api.SprayBoardRow{PlanId: r.PlanID, Seq: r.Seq, ProjectName: r.ProjectName, LotNo: r.LotNo, Color: r.Color, ColorNo: r.ColorNo, SprayingType: r.SprayingType, PaintLine: r.PaintLine, Shift: r.Shift, Urgency: r.Urgency, StartDate: apiBoardDate(r.StartDate), EndDate: apiBoardDate(r.EndDate), PlannedArea: float32(r.PlannedArea), PlannedQty: r.PlannedQty, SheetArea: float32(r.SheetArea), SprayedArea: float32(r.SprayedArea), DaySprayedArea: float32(r.DaySprayedArea), UnsprayedArea: float32(r.UnsprayedArea), Progress: float32(r.Progress), State: r.State, DueToday: r.DueToday})
		out.Summary.Count++
		out.Summary.PlannedArea += float32(r.PlannedArea)
		out.Summary.DaySprayedArea += float32(r.DaySprayedArea)
		out.Summary.SprayedArea += float32(r.SprayedArea)
		if r.State == "done" {
			out.Summary.DoneCount++
		}
		i, ok := lines[r.PaintLine]
		if !ok {
			i = len(out.Summary.ByLine)
			lines[r.PaintLine] = i
			out.Summary.ByLine = append(out.Summary.ByLine, api.SprayLineSummary{PaintLine: r.PaintLine})
		}
		v := &out.Summary.ByLine[i]
		v.Count++
		v.PlannedArea += float32(r.PlannedArea)
		v.DaySprayedArea += float32(r.DaySprayedArea)
	}
	return out, nil
}
func (s *Server) GetSheetDailyBoard(ctx context.Context, req api.GetSheetDailyBoardRequestObject) (api.GetSheetDailyBoardResponseObject, error) {
	rows, err := s.store.DailyBoard(ctx, auth.FromContext(ctx).Scope, boardDate(req.Params.Date), "sheet_daily", optionalString(req.Params.Workshop), req.Params.JobId)
	if err != nil {
		return nil, err
	}
	out := api.GetSheetDailyBoard200JSONResponse{Rows: []api.SheetBoardRow{}}
	out.Summary.ByJob = []api.SheetJobSummary{}
	jobs := map[int64]int{}
	for _, r := range rows {
		out.Rows = append(out.Rows, api.SheetBoardRow{PlanId: r.PlanID, Seq: r.Seq, ProjectName: r.ProjectName, LotNo: r.LotNo, Workshop: r.Workshop, JobId: r.JobID, JobName: r.JobName, Urgency: r.Urgency, StartDate: apiBoardDate(r.StartDate), EndDate: apiBoardDate(r.EndDate), OrderQty: r.OrderQty, OrderArea: float32(r.OrderArea), DayPlanArea: float32(r.DayPlanArea), DoneQty: r.DoneQty, DoneArea: float32(r.DoneArea), DayDoneQty: r.DayDoneQty, DayDoneArea: float32(r.DayDoneArea), RemainingQty: r.RemainingQty, Progress: float32(r.Progress), State: r.State, DueToday: r.DueToday})
		out.Summary.Count++
		out.Summary.OrderArea += float32(r.OrderArea)
		out.Summary.DayPlanArea += float32(r.DayPlanArea)
		out.Summary.DayDoneArea += float32(r.DayDoneArea)
		if r.State == "done" {
			out.Summary.DoneCount++
		}
		i, ok := jobs[r.JobID]
		if !ok {
			i = len(out.Summary.ByJob)
			jobs[r.JobID] = i
			out.Summary.ByJob = append(out.Summary.ByJob, api.SheetJobSummary{JobName: r.JobName})
		}
		v := &out.Summary.ByJob[i]
		v.Count++
		v.DayPlanArea += float32(r.DayPlanArea)
		v.DayDoneArea += float32(r.DayDoneArea)
	}
	return out, nil
}
