package store

import (
	"context"
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
	Qty           int
	Area          float64
	LastScannedAt time.Time
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
}

// GetDashboard computes each metric from the source rows, restricting every batch to the caller's scope.
func (s *Store) GetDashboard(ctx context.Context, sc auth.Scope, days int) (*Dashboard, error) {
	d := &Dashboard{GeneratedAt: time.Now(), BatchStatus: []StatusCount{}, JobLoad: []JobLoad{}, ScanTrend: []DailyScan{}, CustomerArea: []CustomerArea{}, ActiveBatches: []ActiveBatch{}, RecentScans: []Scan{}, RecentActivity: []ScanActivity{}}
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
	rows, err = s.db.Query(ctx, `SELECT sr.batch_id,b.lot_no,sr.job_id,j.name,u.display_name,
 count(*)::int,sum(bi.piece_area)::float8,max(sr.scanned_at)
 FROM scan_record sr JOIN batch b ON b.id=sr.batch_id JOIN job j ON j.id=sr.job_id
 JOIN app_user u ON u.id=sr.scanned_by JOIN label l ON l.id=sr.label_id
 JOIN batch_item bi ON bi.id=l.batch_item_id
 WHERE ($1 OR b.factory_id=ANY($2))
 GROUP BY sr.batch_id,b.lot_no,sr.job_id,j.name,sr.scanned_by,u.display_name,
 to_timestamp(floor(extract(epoch from sr.scanned_at)/600)*600)
 ORDER BY max(sr.scanned_at) DESC LIMIT 8`, args...)
	if err != nil {
		return nil, err
	}
	d.RecentActivity, err = pgx.CollectRows(rows, pgx.RowToStructByPos[ScanActivity])
	if err != nil {
		return nil, err
	}
	return d, nil
}
