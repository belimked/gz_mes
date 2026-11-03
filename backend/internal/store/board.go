package store

import (
	"context"
	"math"
	"time"

	"github.com/belimked/gz_mes/backend/internal/auth"
)

type BoardRow struct {
	PlanID         int64   `json:"planId"`
	Seq            int     `json:"seq"`
	ProjectName    string  `json:"projectName"`
	LotNo          string  `json:"lotNo"`
	Color          string  `json:"color"`
	ColorNo        string  `json:"colorNo"`
	SprayingType   string  `json:"sprayingType"`
	PaintLine      string  `json:"paintLine"`
	Shift          string  `json:"shift"`
	Workshop       string  `json:"workshop"`
	JobID          int64   `json:"jobId"`
	JobName        string  `json:"jobName"`
	Urgency        string  `json:"urgency"`
	StartDate      string  `json:"startDate"`
	EndDate        string  `json:"endDate"`
	PlannedArea    float64 `json:"plannedArea"`
	PlannedQty     int     `json:"plannedQty"`
	SheetArea      float64 `json:"sheetArea"`
	SprayedArea    float64 `json:"sprayedArea"`
	DaySprayedArea float64 `json:"daySprayedArea"`
	UnsprayedArea  float64 `json:"unsprayedArea"`
	OrderQty       int     `json:"orderQty"`
	OrderArea      float64 `json:"orderArea"`
	DayPlanArea    float64 `json:"dayPlanArea"`
	DoneQty        int     `json:"doneQty"`
	DoneArea       float64 `json:"doneArea"`
	DayDoneQty     int     `json:"dayDoneQty"`
	DayDoneArea    float64 `json:"dayDoneArea"`
	RemainingQty   int     `json:"remainingQty"`
	Progress       float64 `json:"progress"`
	State          string  `json:"state"`
	DueToday       bool    `json:"dueToday"`
}

// DailyBoard reads only plans covering the requested day. Scans are deliberately not
// restricted to the plan window: cumulative progress is the batch's live progress.
func (s *Store) DailyBoard(ctx context.Context, sc auth.Scope, date time.Time, kind, filter string, jobID *int64) ([]BoardRow, error) {
	const query = `SELECT pl.id,p.full_name,b.lot_no,b.color,b.color_no,pl.spray_type,pl.paint_line,pl.shift,pl.workshop,
 coalesce(j.id,0),coalesce(j.name,''),pl.urgency,to_char(pl.start_date,'YYYY-MM-DD'),to_char(pl.end_date,'YYYY-MM-DD'),
 b.item_area::float8,lb.qty,coalesce(b.order_qty,0),coalesce(nullif(b.order_area,0),b.item_area)::float8,
 (pl.end_date-pl.start_date+1)::int,
 coalesce(sc.sheet_area,0)::float8,coalesce(sc.done_qty,0)::int,coalesce(sc.done_area,0)::float8,
 coalesce(sc.day_qty,0)::int,coalesce(sc.day_area,0)::float8
 FROM plan pl JOIN batch b ON b.id=pl.batch_id JOIN contract c ON c.id=b.contract_id JOIN project p ON p.id=c.project_id
 LEFT JOIN job j ON j.id=pl.job_id
 CROSS JOIN LATERAL (SELECT count(*)::int qty FROM label l WHERE l.batch_id=b.id AND NOT l.cancelled) lb
 LEFT JOIN LATERAL (
 SELECT sum(bi.piece_area) FILTER (WHERE sj.code='20') sheet_area,
 count(*) FILTER (WHERE sr.job_id=CASE WHEN pl.kind='sheet_daily' THEN pl.job_id ELSE (SELECT id FROM job WHERE code='40') END) done_qty,
 sum(bi.piece_area) FILTER (WHERE sr.job_id=CASE WHEN pl.kind='sheet_daily' THEN pl.job_id ELSE (SELECT id FROM job WHERE code='40') END) done_area,
 count(*) FILTER (WHERE sr.job_id=CASE WHEN pl.kind='sheet_daily' THEN pl.job_id ELSE (SELECT id FROM job WHERE code='40') END AND sr.scanned_at::date=$1::date) day_qty,
 sum(bi.piece_area) FILTER (WHERE sr.job_id=CASE WHEN pl.kind='sheet_daily' THEN pl.job_id ELSE (SELECT id FROM job WHERE code='40') END AND sr.scanned_at::date=$1::date) day_area
 FROM scan_record sr JOIN label l ON l.id=sr.label_id AND NOT l.cancelled JOIN batch_item bi ON bi.id=l.batch_item_id JOIN job sj ON sj.id=sr.job_id
 WHERE sr.batch_id=b.id AND (sj.code='20' OR sj.code='40' OR sr.job_id=pl.job_id)
 ) sc ON true
 WHERE pl.kind=$2 AND pl.audit_state='audited' AND pl.stop_state<>'stopped' AND pl.start_date<=$1::date AND pl.end_date>=$1::date
 AND ($3 OR pl.factory_id=ANY($4)) AND ($5='' OR CASE WHEN pl.kind='spray_daily' THEN pl.paint_line ELSE pl.workshop END=$5)
 AND ($6::bigint IS NULL OR pl.job_id=$6)
 ORDER BY (pl.urgency='urgent') DESC,CASE WHEN pl.kind='spray_daily' THEN pl.paint_line ELSE pl.workshop END,CASE WHEN pl.kind='spray_daily' THEN pl.shift ELSE '' END, j.seq, b.lot_no,pl.id`
	rows, err := s.db.Query(ctx, query, date, kind, sc.All, sc.FactoryIDs, filter, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []BoardRow{}
	for rows.Next() {
		var r BoardRow
		var days int
		var dayQty int
		var dayArea, targetArea float64
		err = rows.Scan(&r.PlanID, &r.ProjectName, &r.LotNo, &r.Color, &r.ColorNo, &r.SprayingType, &r.PaintLine, &r.Shift, &r.Workshop, &r.JobID, &r.JobName, &r.Urgency, &r.StartDate, &r.EndDate, &r.PlannedArea, &r.PlannedQty, &r.OrderQty, &r.OrderArea, &days, &r.SheetArea, &r.DoneQty, &targetArea, &dayQty, &dayArea)
		if err != nil {
			return nil, err
		}
		r.Seq = len(result) + 1
		r.DayPlanArea = math.Round(r.PlannedArea/float64(days)*100) / 100
		r.Progress = 0
		if kind == "spray_daily" {
			r.SprayedArea = targetArea
			r.DaySprayedArea = dayArea
			r.UnsprayedArea = math.Max(0, r.PlannedArea-targetArea)
			if r.PlannedArea > 0 {
				r.Progress = math.Round(targetArea/r.PlannedArea*1000) / 10
			}
			switch {
			case r.Progress >= 100:
				r.State = "done"
			case targetArea > 0:
				r.State = "in_progress"
			case r.SheetArea == 0:
				r.State = "waiting_sheet"
			default:
				r.State = "ready"
			}
		} else {
			r.DoneArea = targetArea
			r.DayDoneQty = dayQty
			r.DayDoneArea = dayArea
			r.RemainingQty = r.PlannedQty - r.DoneQty
			if r.RemainingQty < 0 {
				r.RemainingQty = 0
			}
			if r.PlannedQty > 0 {
				r.Progress = math.Round(float64(r.DoneQty)/float64(r.PlannedQty)*1000) / 10
			}
			switch {
			case r.Progress >= 100:
				r.State = "done"
			case r.DoneQty > 0:
				r.State = "in_progress"
			default:
				r.State = "not_started"
			}
		}
		r.DueToday = r.EndDate == date.Format("2006-01-02") && r.Progress < 100
		result = append(result, r)
	}
	return result, rows.Err()
}
