package server

import (
	"context"
	"time"

	"github.com/oapi-codegen/runtime/types"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/store"
)

// defaultToday fills a missing bound with today's date (server local time).
func defaultToday(from, to *time.Time) (*time.Time, *time.Time) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if from == nil {
		f := today
		from = &f
	}
	if to == nil {
		t := today
		to = &t
	}
	return from, to
}

func toAPICells(cells []store.JobCell) []api.JobCell {
	out := make([]api.JobCell, 0, len(cells))
	for _, c := range cells {
		out = append(out, api.JobCell{JobId: c.JobID, Qty: c.Qty, Area: c.Area, Pct: c.Pct})
	}
	return out
}

func (s *Server) GetBatchJobReport(ctx context.Context, req api.GetBatchJobReportRequestObject) (api.GetBatchJobReportResponseObject, error) {
	q := req.Params
	f := store.BatchJobFilter{Mode: string(q.Mode), From: dateValue(q.From), To: dateValue(q.To), Workshop: optionalString(q.Workshop), OutsourceFactoryID: q.OutsourceFactoryId, JobID: q.JobId, Scanned: q.Scanned, Keyword: optionalString(q.Keyword)}
	if q.Status != nil {
		f.Status = string(*q.Status)
	}
	if f.Mode == "daily" {
		f.From, f.To = defaultToday(f.From, f.To)
	}
	if f.From != nil && f.To != nil && f.From.After(*f.To) {
		return api.GetBatchJobReport400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "日期范围无效")}, nil
	}
	page, size := pageSize(q.Page, q.Size)
	jobs, items, sum, err := s.store.BatchJobReport(ctx, auth.FromContext(ctx).Scope, f, page, size)
	if err != nil {
		return nil, err
	}
	out := api.GetBatchJobReport200JSONResponse{Jobs: make([]api.Job, 0, len(jobs)), Items: make([]api.BatchJobRow, 0, len(items)), Total: sum.Batches, Page: page, Size: size}
	out.Summary.Batches, out.Summary.ItemQty, out.Summary.ItemArea, out.Summary.LabelQty, out.Summary.Cells = sum.Batches, sum.ItemQty, sum.ItemArea, sum.LabelQty, toAPICells(sum.Cells)
	for _, j := range jobs {
		out.Jobs = append(out.Jobs, toAPIJob(j))
	}
	for _, r := range items {
		var required *types.Date
		if r.RequiredDate != nil {
			required = &types.Date{Time: *r.RequiredDate}
		}
		out.Items = append(out.Items, api.BatchJobRow{BatchId: r.BatchID, LotNo: r.LotNo, ProjectName: r.ProjectName, CustomerName: r.CustomerName, ClientOrderNo: r.ClientOrderNo, Thickness: r.Thickness, Color: r.Color, Status: api.BatchStatus(r.Status), Urgency: api.Urgency(r.Urgency), Workshop: r.Workshop, OutsourceFactoryName: r.OutsourceFactoryName, RequiredDate: required, ItemQty: r.ItemQty, ItemArea: r.ItemArea, LabelQty: r.LabelQty, CurrentJobName: r.CurrentJobName, LastScanAt: r.LastScanAt, Cells: toAPICells(r.Cells)})
	}
	return out, nil
}

func (s *Server) GetStaffOutputReport(ctx context.Context, req api.GetStaffOutputReportRequestObject) (api.GetStaffOutputReportResponseObject, error) {
	q := req.Params
	f := store.StaffOutputFilter{JobID: q.JobId, Staff: optionalString(q.Staff), BatchKeyword: optionalString(q.BatchKeyword)}
	f.From, f.To = defaultToday(dateValue(q.From), dateValue(q.To))
	if q.Detail != nil {
		f.Detail = *q.Detail
	}
	if f.From.After(*f.To) {
		return api.GetStaffOutputReport400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "日期范围无效")}, nil
	}
	page, size := pageSize(q.Page, q.Size)
	items, sum, err := s.store.StaffOutputReport(ctx, auth.FromContext(ctx).Scope, f, page, size)
	if err != nil {
		return nil, err
	}
	out := api.GetStaffOutputReport200JSONResponse{Items: make([]api.StaffOutputRow, 0, len(items)), Total: sum.Rows, Page: page, Size: size}
	out.Summary.Rows, out.Summary.Staff, out.Summary.Qty, out.Summary.Area, out.Summary.SprayArea, out.Summary.WeldLine = sum.Rows, sum.Staff, sum.Qty, sum.Area, sum.SprayArea, sum.WeldLine
	for _, r := range items {
		out.Items = append(out.Items, api.StaffOutputRow{UserId: r.UserID, StaffNo: r.StaffNo, DisplayName: r.DisplayName, JobId: r.JobID, JobName: r.JobName, BatchId: r.BatchID, LotNo: r.LotNo, ProjectName: r.ProjectName, Qty: r.Qty, Area: r.Area, SprayArea: r.SprayArea, WeldLine: r.WeldLine, FirstScanAt: r.FirstScanAt, LastScanAt: r.LastScanAt})
	}
	return out, nil
}
