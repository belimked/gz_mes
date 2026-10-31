package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/oapi-codegen/runtime/types"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
)

// The generated operation only declares 200/401; the handler also validates days defensively.
type dashboardBadRequest api.Error

func (r dashboardBadRequest) VisitGetDashboardResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	return json.NewEncoder(w).Encode(r)
}

func (s *Server) GetDashboard(ctx context.Context, req api.GetDashboardRequestObject) (api.GetDashboardResponseObject, error) {
	days := 14
	if req.Params.Days != nil {
		days = *req.Params.Days
	}
	if days < 7 || days > 90 {
		return dashboardBadRequest{Code: "invalid_request", Message: "days 必须在 7 到 90 之间"}, nil
	}
	d, err := s.store.GetDashboard(ctx, auth.FromContext(ctx).Scope, days)
	if err != nil {
		return nil, err
	}
	out := api.Dashboard{GeneratedAt: d.GeneratedAt,
		Kpi:         api.DashboardKpi{InProduction: d.KPI.InProduction, PendingSchedule: d.KPI.PendingSchedule, DoneToday: d.KPI.DoneToday, TodayScanQty: d.KPI.TodayScanQty, TodayScanArea: d.KPI.TodayScanArea, LabelQty: d.KPI.LabelQty, DeliveredQty: d.KPI.DeliveredQty, DeliveredArea: d.KPI.DeliveredArea, UnaccountedNotes: d.KPI.UnaccountedNotes, UnaccountedArea: d.KPI.UnaccountedArea, UrgentOpen: d.KPI.UrgentOpen, OverdueOpen: d.KPI.OverdueOpen},
		BatchStatus: make([]api.StatusCount, 0, len(d.BatchStatus)), JobLoad: make([]api.JobLoad, 0, len(d.JobLoad)), ScanTrend: make([]api.DailyScan, 0, len(d.ScanTrend)), CustomerArea: make([]api.CustomerArea, 0, len(d.CustomerArea)), ActiveBatches: make([]api.ActiveBatch, 0, len(d.ActiveBatches)), RecentScans: make([]api.Scan, 0, len(d.RecentScans)), RecentActivity: make([]api.ScanActivity, 0, len(d.RecentActivity))}
	for _, v := range d.BatchStatus {
		out.BatchStatus = append(out.BatchStatus, api.StatusCount{Status: api.BatchStatus(v.Status), Count: v.Count})
	}
	for _, v := range d.JobLoad {
		out.JobLoad = append(out.JobLoad, api.JobLoad{JobId: v.JobID, JobCode: v.JobCode, JobName: v.JobName, Seq: v.Seq, Qty: v.Qty, Area: v.Area, CycleSamples: v.CycleSamples, CycleHours: v.CycleHours, CycleP90Hours: v.CycleP90Hours, DwellHours: v.DwellHours, DwellMaxHours: v.DwellMaxHours})
	}
	for _, v := range d.ScanTrend {
		out.ScanTrend = append(out.ScanTrend, api.DailyScan{Date: types.Date{Time: v.Date}, Qty: v.Qty, Area: v.Area, DeliveredQty: v.DeliveredQty, DeliveredArea: v.DeliveredArea, ActiveBatches: v.ActiveBatches, DoneBatches: v.DoneBatches})
	}
	for _, v := range d.CustomerArea {
		out.CustomerArea = append(out.CustomerArea, api.CustomerArea{CustomerId: v.CustomerID, CustomerName: v.CustomerName, Area: v.Area, Batches: v.Batches})
	}
	for _, v := range d.ActiveBatches {
		var required *types.Date
		if v.RequiredDate != nil {
			required = &types.Date{Time: *v.RequiredDate}
		}
		out.ActiveBatches = append(out.ActiveBatches, api.ActiveBatch{Id: v.ID, LotNo: v.LotNo, ProjectName: v.ProjectName, CustomerName: v.CustomerName, Thickness: v.Thickness, Color: v.Color, ItemQty: v.ItemQty, ItemArea: v.ItemArea, LabelQty: v.LabelQty, CurrentSeq: v.CurrentSeq, CurrentJobName: v.CurrentJobName, Progress: v.Progress, RequiredDate: required, Urgency: api.Urgency(v.Urgency)})
	}
	for _, v := range d.RecentActivity {
		out.RecentActivity = append(out.RecentActivity, api.ScanActivity{BatchId: v.BatchID, LotNo: v.LotNo, JobId: v.JobID, JobName: v.JobName, ScannedBy: v.ScannedBy, Qty: v.Qty, Area: v.Area, LastScannedAt: v.LastScannedAt})
	}
	for _, v := range d.RecentScans {
		out.RecentScans = append(out.RecentScans, toAPIScan(v))
	}
	return api.GetDashboard200JSONResponse(out), nil
}
