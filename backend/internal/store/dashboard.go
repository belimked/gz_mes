package store

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

type DashboardKPI struct {
	InProduction, PendingSchedule, DoneToday int
	TodayScanQty                             int
	TodayScanArea                            float64
	LabelQty, DeliveredQty                   int
	DeliveredArea                            float64
	UnaccountedNotes                         int
	UnaccountedArea                          float64
	UrgentOpen, OverdueOpen                  int
}
type StatusCount struct {
	Status string
	Count  int
}
type JobLoad struct {
	JobID            int64
	JobCode, JobName string
	Seq, Qty         int
	Area             float64
	CycleSamples     int
	CycleHours       *float64
	CycleP90Hours    *float64
	DwellHours       *float64
	DwellMaxHours    *float64
}
type DailyScan struct {
	Date          time.Time
	Qty           int
	Area          float64
	DeliveredQty  int
	DeliveredArea float64
	ActiveBatches int
	DoneBatches   int
}
type CustomerArea struct {
	CustomerID   *int64
	CustomerName string
	Area         float64
	Batches      int
}
type ActiveBatch struct {
	ID                               int64
	LotNo, ProjectName, CustomerName string
	Thickness                        float64
	Color                            string
	ItemQty                          int
	ItemArea                         float64
	LabelQty, CurrentSeq             int
	CurrentJobName                   string
	Progress                         int
	RequiredDate                     *time.Time
	Urgency                          string
}
type ScanActivity struct {
	BatchID       int64
	LotNo         string
	JobID         int64
	JobName       string
	ScannedBy     string
	StaffNo       string
	Qty           int
	Area          float64
	LastScannedAt time.Time
}
type DashboardAlert struct {
	Rule, Level, Title, Detail, Link string
	BatchID, JobID                   *int64
	LotNo                            *string
	Count, Days                      *int
	Hours                            *float64
}

type Dashboard struct {
	GeneratedAt    time.Time
	KPI            DashboardKPI
	BatchStatus    []StatusCount
	JobLoad        []JobLoad
	ScanTrend      []DailyScan
	CustomerArea   []CustomerArea
	ActiveBatches  []ActiveBatch
	RecentScans    []Scan
	RecentActivity []ScanActivity
	Alerts         []DashboardAlert
}

// GetDashboard computes each metric from the source rows, restricting every batch to the caller's scope.
func (s *Store) GetDashboard(ctx context.Context, sc auth.Scope, days int) (*Dashboard, error) {
	d := &Dashboard{GeneratedAt: time.Now(), BatchStatus: []StatusCount{}, JobLoad: []JobLoad{}, ScanTrend: []DailyScan{}, CustomerArea: []CustomerArea{}, ActiveBatches: []ActiveBatch{}, RecentScans: []Scan{}, RecentActivity: []ScanActivity{}, Alerts: []DashboardAlert{}}
	args := []any{sc.All, sc.FactoryIDs}
	err := s.db.QueryRow(ctx, `SELECT count(*) FILTER (WHERE status='in_production')::int,
 count(*) FILTER (WHERE status='pending')::int,
 count(*) FILTER (WHERE status='done' AND updated_at::date=current_date)::int,
 count(*) FILTER (WHERE urgency='urgent' AND status!='done')::int,
 count(*) FILTER (WHERE required_date<current_date AND status!='done')::int
 FROM batch b WHERE ($1 OR b.factory_id=ANY($2))`, args...).Scan(&d.KPI.InProduction, &d.KPI.PendingSchedule, &d.KPI.DoneToday, &d.KPI.UrgentOpen, &d.KPI.OverdueOpen)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRow(ctx, `SELECT count(*)::int,coalesce(sum(bi.piece_area),0)::float8 FROM scan_record sr JOIN batch b ON b.id=sr.batch_id JOIN label l ON l.id=sr.label_id JOIN batch_item bi ON bi.id=l.batch_item_id WHERE ($1 OR b.factory_id=ANY($2)) AND sr.scanned_at::date=current_date`, args...).Scan(&d.KPI.TodayScanQty, &d.KPI.TodayScanArea)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRow(ctx, `SELECT count(*)::int FROM label l JOIN batch b ON b.id=l.batch_id WHERE ($1 OR b.factory_id=ANY($2)) AND NOT l.cancelled AND b.status!='done'`, args...).Scan(&d.KPI.LabelQty)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRow(ctx, `SELECT coalesce(sum(dn.qty) FILTER (WHERE dn.delivered_at BETWEEN current_date-($3::int-1) AND current_date),0)::int,
 coalesce(sum(dn.area) FILTER (WHERE dn.delivered_at BETWEEN current_date-($3::int-1) AND current_date),0)::float8,
 count(*) FILTER (WHERE dn.account_id IS NULL)::int,
 coalesce(sum(dn.area) FILTER (WHERE dn.account_id IS NULL),0)::float8
 FROM delivery_note dn JOIN batch b ON b.id=dn.batch_id WHERE ($1 OR b.factory_id=ANY($2))`, sc.All, sc.FactoryIDs, days).Scan(&d.KPI.DeliveredQty, &d.KPI.DeliveredArea, &d.KPI.UnaccountedNotes, &d.KPI.UnaccountedArea)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `SELECT statuses.status,count(b.id)::int FROM (VALUES ('pending'),('scheduled'),('in_production'),('done')) statuses(status) LEFT JOIN batch b ON b.status=statuses.status AND ($1 OR b.factory_id=ANY($2)) GROUP BY statuses.status ORDER BY array_position(ARRAY['pending','scheduled','in_production','done'],statuses.status)`, args...)
	if err != nil {
		return nil, err
	}
	d.BatchStatus, err = pgx.CollectRows(rows, pgx.RowToStructByPos[StatusCount])
	if err != nil {
		return nil, err
	}
	rows, err = s.db.Query(ctx, `WITH latest AS (
 SELECT DISTINCT ON (sr.label_id) sr.label_id,sr.job_id,sr.scanned_at,l.batch_item_id
 FROM scan_record sr JOIN job j ON j.id=sr.job_id JOIN label l ON l.id=sr.label_id JOIN batch b ON b.id=l.batch_id
 WHERE b.status='in_production' AND NOT l.cancelled AND ($1 OR b.factory_id=ANY($2))
 ORDER BY sr.label_id,j.seq DESC
 ), load AS (
 SELECT latest.job_id,count(*)::int qty,coalesce(sum(bi.piece_area),0)::float8 area,
 round(avg(EXTRACT(EPOCH FROM (now()-latest.scanned_at))/3600),2)::float8 dwell_hours,
 round(max(EXTRACT(EPOCH FROM (now()-latest.scanned_at))/3600),2)::float8 dwell_max_hours
 FROM latest JOIN batch_item bi ON bi.id=latest.batch_item_id GROUP BY latest.job_id
 ), cycle AS (
 SELECT s.job_id,count(*)::int samples,
 round(avg(EXTRACT(EPOCH FROM (s.scanned_at-CASE WHEN j.seq=1 THEN l.created_at ELSE p.scanned_at END))/3600),2)::float8 cycle_hours,
 round((percentile_cont(0.9) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (s.scanned_at-CASE WHEN j.seq=1 THEN l.created_at ELSE p.scanned_at END))/3600))::numeric,2)::float8 cycle_p90_hours
 FROM scan_record s JOIN job j ON j.id=s.job_id JOIN label l ON l.id=s.label_id JOIN batch b ON b.id=l.batch_id
 LEFT JOIN job prev ON prev.seq=j.seq-1 LEFT JOIN scan_record p ON p.label_id=s.label_id AND p.job_id=prev.id
 WHERE ($1 OR b.factory_id=ANY($2)) AND s.scanned_at>=current_date-($3::int-1)
 AND (j.seq=1 OR p.id IS NOT NULL) GROUP BY s.job_id
 )
 SELECT j.id,j.code,j.name,j.seq,coalesce(load.qty,0),coalesce(load.area,0)::float8,
 coalesce(cycle.samples,0),cycle.cycle_hours,cycle.cycle_p90_hours,load.dwell_hours,load.dwell_max_hours
 FROM job j LEFT JOIN load ON load.job_id=j.id LEFT JOIN cycle ON cycle.job_id=j.id ORDER BY j.seq`, sc.All, sc.FactoryIDs, days)
	if err != nil {
		return nil, err
	}
	d.JobLoad, err = pgx.CollectRows(rows, pgx.RowToStructByPos[JobLoad])
	if err != nil {
		return nil, err
	}
	rows, err = s.db.Query(ctx, `SELECT dates.dt::date,coalesce(sc.qty,0)::int,coalesce(sc.area,0)::float8,
 coalesce(sh.qty,0)::int,coalesce(sh.area,0)::float8,coalesce(sc.active_batches,0)::int,coalesce(done.qty,0)::int
 FROM generate_series(current_date-($3::int-1),current_date,'1 day'::interval) dates(dt)
 LEFT JOIN (SELECT sr.scanned_at::date dt,count(*) qty,sum(bi.piece_area) area,count(DISTINCT sr.batch_id) active_batches FROM scan_record sr JOIN batch b ON b.id=sr.batch_id JOIN label l ON l.id=sr.label_id JOIN batch_item bi ON bi.id=l.batch_item_id WHERE ($1 OR b.factory_id=ANY($2)) AND sr.scanned_at::date BETWEEN current_date-($3::int-1) AND current_date GROUP BY 1) sc ON sc.dt=dates.dt
 LEFT JOIN (SELECT dn.delivered_at dt,sum(dn.qty) qty,sum(dn.area) area FROM delivery_note dn JOIN batch b ON b.id=dn.batch_id WHERE ($1 OR b.factory_id=ANY($2)) AND dn.delivered_at BETWEEN current_date-($3::int-1) AND current_date GROUP BY 1) sh ON sh.dt=dates.dt
 LEFT JOIN (SELECT b.updated_at::date dt,count(*) qty FROM batch b WHERE b.status='done' AND ($1 OR b.factory_id=ANY($2)) AND b.updated_at::date BETWEEN current_date-($3::int-1) AND current_date GROUP BY 1) done ON done.dt=dates.dt
 ORDER BY dates.dt`, sc.All, sc.FactoryIDs, days)
	if err != nil {
		return nil, err
	}
	d.ScanTrend, err = pgx.CollectRows(rows, pgx.RowToStructByPos[DailyScan])
	if err != nil {
		return nil, err
	}
	rows, err = s.db.Query(ctx, `SELECT c.id,c.name,sum(b.order_area)::float8,count(*)::int FROM batch b JOIN contract ct ON ct.id=b.contract_id JOIN customer c ON c.id=ct.customer_id WHERE b.status!='done' AND ($1 OR b.factory_id=ANY($2)) GROUP BY c.id ORDER BY sum(b.order_area) DESC,c.id`, args...)
	if err != nil {
		return nil, err
	}
	customers, err := pgx.CollectRows(rows, pgx.RowToStructByPos[CustomerArea])
	if err != nil {
		return nil, err
	}
	for i, c := range customers {
		if i < 6 {
			d.CustomerArea = append(d.CustomerArea, c)
		} else {
			if i == 6 {
				d.CustomerArea = append(d.CustomerArea, CustomerArea{CustomerName: "其他"})
			}
			other := &d.CustomerArea[6]
			other.Area += c.Area
			other.Batches += c.Batches
		}
	}
	rows, err = s.db.Query(ctx, `SELECT b.id,b.lot_no,p.full_name,c.name,b.thickness::float8,b.color,b.item_qty,b.item_area::float8,
 coalesce(labels.qty,0)::int,coalesce(scans.seq,0)::int,coalesce(j.name,''),
 CASE WHEN coalesce(labels.qty,0)=0 THEN 0 ELSE (coalesce(scans.qty,0)*100/(labels.qty*11))::int END,b.required_date,b.urgency
 FROM batch b JOIN contract ct ON ct.id=b.contract_id JOIN project p ON p.id=ct.project_id JOIN customer c ON c.id=ct.customer_id
 LEFT JOIN LATERAL (SELECT count(*) qty FROM label l WHERE l.batch_id=b.id AND NOT l.cancelled) labels ON true
 LEFT JOIN LATERAL (SELECT count(*) qty,max(j.seq) seq FROM scan_record sr JOIN job j ON j.id=sr.job_id WHERE sr.batch_id=b.id) scans ON true
 LEFT JOIN job j ON j.seq=scans.seq
 WHERE b.status='in_production' AND ($1 OR b.factory_id=ANY($2)) ORDER BY b.required_date NULLS LAST,b.id LIMIT 8`, args...)
	if err != nil {
		return nil, err
	}
	d.ActiveBatches, err = pgx.CollectRows(rows, pgx.RowToStructByPos[ActiveBatch])
	if err != nil {
		return nil, err
	}
	rows, err = s.db.Query(ctx, scanSelect+` WHERE ($1 OR b.factory_id=ANY($2)) ORDER BY sr.scanned_at DESC,sr.id DESC LIMIT 8`, args...)
	if err != nil {
		return nil, err
	}
	d.RecentScans, err = pgx.CollectRows(rows, pgx.RowToStructByPos[Scan])
	if err != nil {
		return nil, err
	}
	rows, err = s.db.Query(ctx, `SELECT sr.batch_id,b.lot_no,sr.job_id,j.name,u.display_name,u.staff_no,
 count(*)::int,sum(bi.piece_area)::float8,max(sr.scanned_at)
 FROM scan_record sr JOIN batch b ON b.id=sr.batch_id JOIN job j ON j.id=sr.job_id
 JOIN app_user u ON u.id=sr.scanned_by JOIN label l ON l.id=sr.label_id
 JOIN batch_item bi ON bi.id=l.batch_item_id
 WHERE ($1 OR b.factory_id=ANY($2))
 GROUP BY sr.batch_id,b.lot_no,sr.job_id,j.name,sr.scanned_by,u.display_name,u.staff_no,
 to_timestamp(floor(extract(epoch from sr.scanned_at)/600)*600)
 ORDER BY max(sr.scanned_at) DESC LIMIT 8`, args...)
	if err != nil {
		return nil, err
	}
	d.RecentActivity, err = pgx.CollectRows(rows, pgx.RowToStructByPos[ScanActivity])
	if err != nil {
		return nil, err
	}
	d.Alerts, err = s.dashboardAlerts(ctx, sc)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Store) dashboardAlerts(ctx context.Context, sc auth.Scope) ([]DashboardAlert, error) {
	alerts := make([]DashboardAlert, 0)
	args := []any{sc.All, sc.FactoryIDs}
	// The date rules share one scoped batch scan. An alert can satisfy more than one rule.
	rows, err := s.db.Query(ctx, `SELECT b.id,b.lot_no,p.full_name,b.required_date,
 coalesce(j.name,'未开工'),b.status,b.plan_sheet_start,b.plan_cut_date,
 b.required_date-current_date,b.plan_sheet_start<current_date,b.plan_cut_date<current_date,
 EXISTS(SELECT 1 FROM scan_record sr WHERE sr.batch_id=b.id),
 EXISTS(SELECT 1 FROM batch_item bi WHERE bi.batch_id=b.id),
 EXISTS(SELECT 1 FROM label l WHERE l.batch_id=b.id)
 FROM batch b JOIN contract ct ON ct.id=b.contract_id JOIN project p ON p.id=ct.project_id
 LEFT JOIN LATERAL (SELECT sj.name FROM scan_record sr JOIN job sj ON sj.id=sr.job_id WHERE sr.batch_id=b.id ORDER BY sj.seq DESC,sr.scanned_at DESC LIMIT 1) j ON true
 WHERE ($1 OR b.factory_id=ANY($2)) AND b.status IN ('scheduled','in_production')
 AND (b.required_date<=current_date+3 OR b.plan_sheet_start<current_date OR b.plan_cut_date<current_date)`, args...)
	if err != nil {
		return nil, err
	}
	type batchAlertRow struct {
		ID                         int64
		Lot, Project               string
		Required                   *time.Time
		Job, Status                string
		Sheet, Cut                 *time.Time
		DueDays                    *int
		SheetPast, CutPast         *bool
		HasScan, HasItem, HasLabel bool
	}
	batches, err := pgx.CollectRows(rows, pgx.RowToStructByPos[batchAlertRow])
	if err != nil {
		return nil, err
	}
	for _, b := range batches {
		link := "/scheduling/batches?keyword=" + url.QueryEscape(b.Lot)
		if b.DueDays != nil {
			delta := *b.DueDays
			if delta < 0 {
				n := -delta
				alerts = append(alerts, DashboardAlert{Rule: "overdue", Level: "error", Title: fmt.Sprintf("批次 %s 已逾期 %d 天", b.Lot, n), Detail: fmt.Sprintf("%s · 需求交期 %s · 当前工序 %s", b.Project, b.Required.Format("2006-01-02"), b.Job), Link: link, BatchID: &b.ID, LotNo: &b.Lot, Days: &n})
			} else if delta <= 3 {
				title := fmt.Sprintf("批次 %s %d 天后到期", b.Lot, delta)
				if delta == 0 {
					title = fmt.Sprintf("批次 %s 今天到期", b.Lot)
				}
				alerts = append(alerts, DashboardAlert{Rule: "due_soon", Level: "warning", Title: title, Detail: fmt.Sprintf("%s · 需求交期 %s · 当前工序 %s", b.Project, b.Required.Format("2006-01-02"), b.Job), Link: link, BatchID: &b.ID, LotNo: &b.Lot, Days: &delta})
			}
		}
		if b.Status == "scheduled" && b.Sheet != nil && b.SheetPast != nil && *b.SheetPast && !b.HasScan {
			alerts = append(alerts, DashboardAlert{Rule: "idle", Level: "warning", Title: fmt.Sprintf("批次 %s 计划 %s 开工但未报工", b.Lot, b.Sheet.Format("2006-01-02")), Detail: b.Project, Link: link, BatchID: &b.ID, LotNo: &b.Lot})
		}
		if b.Status == "scheduled" && b.Cut != nil && b.CutPast != nil && *b.CutPast && b.HasItem && !b.HasLabel {
			alerts = append(alerts, DashboardAlert{Rule: "no_labels", Level: "info", Title: fmt.Sprintf("批次 %s 已过开料日期但未生成标签", b.Lot), Detail: b.Project, Link: link, BatchID: &b.ID, LotNo: &b.Lot})
		}
	}
	// Same latest-by-job-sequence definition as jobLoad; only labels older than 24h count.
	rows, err = s.db.Query(ctx, `WITH latest AS (
 SELECT DISTINCT ON (sr.label_id) sr.label_id,sr.job_id,sr.scanned_at,l.batch_id
 FROM scan_record sr JOIN job j ON j.id=sr.job_id JOIN label l ON l.id=sr.label_id JOIN batch b ON b.id=l.batch_id
 WHERE b.status='in_production' AND NOT l.cancelled AND ($1 OR b.factory_id=ANY($2))
 ORDER BY sr.label_id,j.seq DESC
 ) SELECT j.id,j.name,count(*)::int,max(extract(epoch FROM now()-latest.scanned_at)/3600)::float8,
 (array_agg(DISTINCT b.lot_no ORDER BY b.lot_no))[1:3]
 FROM latest JOIN job j ON j.id=latest.job_id JOIN batch b ON b.id=latest.batch_id
 WHERE latest.scanned_at<now()-interval '24 hours' GROUP BY j.id,j.name`, args...)
	if err != nil {
		return nil, err
	}
	type stalledRow struct {
		ID    int64
		Name  string
		Count int
		Hours float64
		Lots  []string
	}
	stalled, err := pgx.CollectRows(rows, pgx.RowToStructByPos[stalledRow])
	if err != nil {
		return nil, err
	}
	for _, v := range stalled {
		level := "warning"
		if v.Hours > 72 {
			level = "error"
		}
		age := fmt.Sprintf("%d 小时", int(v.Hours))
		if v.Hours >= 48 {
			age = fmt.Sprintf("%d 天", int(v.Hours/24))
		}
		alerts = append(alerts, DashboardAlert{Rule: "stalled", Level: level, Title: fmt.Sprintf("%s 有 %d 件停留超 %s", v.Name, v.Count, age), Detail: "涉及批次 " + strings.Join(v.Lots, "、"), Link: fmt.Sprintf("/production/labels?jobId=%d", v.ID), JobID: &v.ID, Count: &v.Count, Hours: &v.Hours})
	}
	var count int
	var hours, area float64
	var earliest *time.Time
	err = s.db.QueryRow(ctx, `SELECT count(*)::int,coalesce(extract(epoch FROM now()-min(dn.delivered_at))/3600,0)::float8,
 min(dn.delivered_at),coalesce(sum(dn.area),0)::float8
 FROM delivery_note dn WHERE dn.account_id IS NULL AND dn.delivered_at<now()-interval '7 days'
 AND ($1 OR dn.factory_id=ANY($2))`, args...).Scan(&count, &hours, &earliest, &area)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		alerts = append(alerts, DashboardAlert{Rule: "unaccounted", Level: "warning", Title: fmt.Sprintf("%d 张发货单超 7 天未对账", count), Detail: fmt.Sprintf("最早 %s，合计 %g m²", earliest.Format("2006-01-02"), area), Link: "/finance/deliver-accounts", Count: &count, Hours: &hours})
	}
	priority := map[string]int{"error": 0, "warning": 1, "info": 2}
	sort.SliceStable(alerts, func(i, j int) bool {
		a, b := alerts[i], alerts[j]
		if a.Level != b.Level {
			return priority[a.Level] < priority[b.Level]
		}
		var aa, bb float64
		if a.Days != nil {
			aa = float64(*a.Days)
		} else if a.Hours != nil {
			aa = *a.Hours / 24
		}
		if b.Days != nil {
			bb = float64(*b.Days)
		} else if b.Hours != nil {
			bb = *b.Hours / 24
		}
		return aa > bb
	})
	if len(alerts) > 20 {
		alerts = alerts[:20]
	}
	return alerts, nil
}
