package server

import (
	"context"

	"github.com/oapi-codegen/runtime/types"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/store"
)

func toAPIQc(v store.QcRecord) api.QcRecord {
	return api.QcRecord{Id: v.ID, LabelId: v.LabelID, BarCode: v.BarCode, BatchId: v.BatchID, LotNo: v.LotNo, ProjectName: v.ProjectName, OutsourceFactoryName: v.OutsourceFactoryName,
		PieceNo: v.PieceNo, PieceIndex: v.PieceIndex, PieceArea: v.PieceArea, Color: v.Color, JobId: v.JobID, JobName: v.JobName, Result: api.QcResult(v.Result),
		DefectJobId: v.DefectJobID, DefectJobName: v.DefectJobName, DefectCategory: v.DefectCategory, DefectReason: v.DefectReason,
		InspectedBy: v.InspectedBy, StaffNo: v.StaffNo, InspectedAt: v.InspectedAt, Device: v.Device, Reinspected: v.Reinspected, ReinspectedAt: v.ReinspectedAt}
}

func (s *Server) ListQcScans(ctx context.Context, req api.ListQcScansRequestObject) (api.ListQcScansResponseObject, error) {
	q := req.Params
	f := store.QcFilter{BatchID: q.BatchId, JobID: q.JobId, DefectJobID: q.DefectJobId, Reinspected: q.Reinspected, Begin: dateValue(q.BeginDate), Keyword: optionalString(q.Keyword), Staff: optionalString(q.Staff)}
	if q.Result != nil {
		f.Result = string(*q.Result)
	}
	if q.EndDate != nil {
		end := q.EndDate.AddDate(0, 0, 1)
		f.End = &end
	}
	if f.Begin != nil && f.End != nil && !f.Begin.Before(*f.End) {
		return api.ListQcScans400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "日期范围无效")}, nil
	}
	page, size := pageSize(q.Page, q.Size)
	items, total, sum, err := s.store.ListQcRecords(ctx, auth.FromContext(ctx).Scope, f, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListQcScans200JSONResponse{Items: make([]api.QcRecord, 0, len(items)), Total: total}
	out.Summary.Inspected, out.Summary.Passed, out.Summary.Failed = sum.Inspected, sum.Passed, sum.Failed
	out.Summary.FailedArea, out.Summary.Reinspected, out.Summary.Pending = sum.FailedArea, sum.Reinspected, sum.Pending
	for _, v := range items {
		out.Items = append(out.Items, toAPIQc(v))
	}
	return out, nil
}

func (s *Server) GetQualityReport(ctx context.Context, req api.GetQualityReportRequestObject) (api.GetQualityReportResponseObject, error) {
	q := req.Params
	from, to := defaultToday(dateValue(q.From), dateValue(q.To))
	if from.After(*to) {
		return api.GetQualityReport400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "日期范围无效")}, nil
	}
	page, size := pageSize(q.Page, q.Size)
	items, total, sum, err := s.store.QualityReport(ctx, auth.FromContext(ctx).Scope, store.QualityReportFilter{From: from, To: to, JobID: q.JobId, DefectJobID: q.DefectJobId, Keyword: optionalString(q.Keyword)}, page, size)
	if err != nil {
		return nil, err
	}
	out := api.GetQualityReport200JSONResponse{Items: make([]api.QualityReportRow, 0, len(items)), Total: total}
	out.Summary.Inspected, out.Summary.Passed, out.Summary.Failed = sum.Inspected, sum.Passed, sum.Failed
	out.Summary.FailedArea, out.Summary.Reinspected, out.Summary.PassRate = sum.FailedArea, sum.Reinspected, sum.PassRate
	for _, v := range items {
		out.Items = append(out.Items, api.QualityReportRow{Date: types.Date{Time: v.Date}, JobId: v.JobID, JobName: v.JobName, BatchId: v.BatchID, LotNo: v.LotNo, ProjectName: v.ProjectName, OutsourceFactoryName: v.OutsourceFactoryName,
			DefectJobId: v.DefectJobID, DefectJobName: v.DefectJobName, DefectCategory: v.DefectCategory, DefectReason: v.DefectReason, Qty: v.Qty, Area: v.Area, ReinspectedQty: v.ReinspectedQty, PieceNos: v.PieceNos, Inspectors: v.Inspectors})
	}
	return out, nil
}

func (s *Server) GetDefectDailyReport(ctx context.Context, req api.GetDefectDailyReportRequestObject) (api.GetDefectDailyReportResponseObject, error) {
	q := req.Params
	from, to := defaultToday(dateValue(q.From), dateValue(q.To))
	if from.After(*to) {
		return api.GetDefectDailyReport400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "日期范围无效")}, nil
	}
	page, size := pageSize(q.Page, q.Size)
	jobs, items, sum, err := s.store.DefectDailyReport(ctx, auth.FromContext(ctx).Scope, store.DefectDailyFilter{From: from, To: to, Workshop: optionalString(q.Workshop), OutsourceFactoryID: q.OutsourceFactoryId, Keyword: optionalString(q.Keyword)}, page, size)
	if err != nil {
		return nil, err
	}
	out := api.GetDefectDailyReport200JSONResponse{Jobs: make([]api.Job, 0, len(jobs)), Items: make([]api.DefectDailyRow, 0, len(items)), Total: sum.Batches}
	out.Summary.Batches, out.Summary.FailedQty, out.Summary.FailedArea = sum.Batches, sum.FailedQty, sum.FailedArea
	out.Summary.ReinspectedQty, out.Summary.PendingQty, out.Summary.PendingArea = sum.ReinspectedQty, sum.PendingQty, sum.PendingArea
	for _, j := range jobs {
		out.Jobs = append(out.Jobs, toAPIJob(j))
	}
	for _, v := range items {
		cells := make([]api.DefectJobCell, 0, len(v.Cells))
		for _, c := range v.Cells {
			cells = append(cells, api.DefectJobCell{JobId: c.JobID, JobName: c.JobName, FailedQty: c.FailedQty, FailedArea: c.FailedArea, ReinspectedQty: c.ReinspectedQty, ReinspectedArea: c.ReinspectedArea, PendingQty: c.PendingQty, PendingArea: c.PendingArea})
		}
		out.Items = append(out.Items, api.DefectDailyRow{BatchId: v.BatchID, LotNo: v.LotNo, ProjectName: v.ProjectName, CustomerName: v.CustomerName, ClientOrderNo: v.ClientOrderNo, Thickness: v.Thickness, Color: v.Color, Workshop: v.Workshop, OutsourceFactoryName: v.OutsourceFactoryName,
			ItemQty: v.ItemQty, ItemArea: v.ItemArea, FailedQty: v.FailedQty, FailedArea: v.FailedArea, PendingQty: v.PendingQty, Cells: cells})
	}
	return out, nil
}
