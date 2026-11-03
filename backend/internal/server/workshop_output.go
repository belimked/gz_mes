package server

import (
	"context"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/store"
)

func workshopAPIRow(r store.WorkshopOutputRow) api.WorkshopOutputRow {
	a := api.WorkshopOutputRow{Period: r.Period, BatchCount: r.BatchCount, BatchArea: float32(r.BatchArea), ScheduledCount: r.ScheduledCount, ScheduledArea: float32(r.ScheduledArea), SheetQty: r.SheetQty, SheetArea: float32(r.SheetArea), SheetOwnArea: float32(r.SheetOwnArea), SheetOutsourceArea: float32(r.SheetOutsourceArea), SprayQty: r.SprayQty, SprayArea: float32(r.SprayArea), DeliveredQty: r.DeliveredQty, DeliveredArea: float32(r.DeliveredArea), WorkshopSheetArea: make([]float32, 0, len(r.WorkshopSheetArea))}
	for _, v := range r.WorkshopSheetArea {
		a.WorkshopSheetArea = append(a.WorkshopSheetArea, float32(v))
	}
	return a
}

func (s *Server) GetWorkshopOutputReport(ctx context.Context, req api.GetWorkshopOutputReportRequestObject) (api.GetWorkshopOutputReportResponseObject, error) {
	q := req.Params
	now := time.Now()
	month := q.Granularity == "month"
	from, to := dateValue(q.From), dateValue(q.To)
	if from == nil {
		v := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		if month {
			v = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		}
		from = &v
	}
	if to == nil {
		v := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		to = &v
	}
	if month {
		a := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
		b := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC)
		from = &a
		to = &b
	}
	invalid := from.After(*to)
	if month {
		invalid = invalid || (to.Year()-from.Year())*12+int(to.Month()-from.Month()) >= 36
	} else {
		invalid = invalid || to.Sub(*from).Hours()/24 >= 366
	}
	if invalid {
		return api.GetWorkshopOutputReport400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "日期范围无效")}, nil
	}
	periods, workshops, total, comparison, err := s.store.WorkshopOutput(ctx, auth.FromContext(ctx).Scope, store.WorkshopOutputFilter{From: *from, To: *to, Month: month, FactoryID: q.FactoryId, Workshop: optionalString(q.Workshop)})
	if err != nil {
		return nil, err
	}
	out := api.GetWorkshopOutputReport200JSONResponse{Periods: make([]api.WorkshopOutputRow, 0, len(periods)), Workshops: workshops, Total: workshopAPIRow(total)}
	for _, r := range periods {
		out.Periods = append(out.Periods, workshopAPIRow(r))
	}
	if month {
		items := make([]api.WorkshopComparison, 0, len(comparison))
		for _, c := range comparison {
			r := api.WorkshopComparison{Workshop: c.Workshop, PrevPrevArea: float32(c.PrevPrevArea), PrevArea: float32(c.PrevArea), CurrentArea: float32(c.CurrentArea)}
			if c.ChainRate != nil {
				v := float32(*c.ChainRate)
				r.ChainRate = &v
			}
			items = append(items, r)
		}
		out.Comparison = &items
	}
	return out, nil
}
