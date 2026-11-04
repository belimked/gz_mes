package store

import (
	"context"
	"time"

	"github.com/belimked/gz_mes/backend/internal/auth"
)

type WorkOrder struct {
	BatchID                                                     int64
	LotNo, ProjectName, CustomerName, Workshop, Urgency, Status string
	OrderDate                                                   time.Time
	RequiredDate, PlanCutDate                                   *time.Time
	ItemQty                                                     int
	ItemArea                                                    float64
	LabelQty                                                    int
	Jobs                                                        []JobCell
}

func (s *Store) WorkOrders(ctx context.Context, sc auth.Scope, status, keyword string, page, size int, today time.Time) ([]Job, []WorkOrder, int, error) {
	jobs, err := s.Jobs(ctx)
	if err != nil {
		return nil, nil, 0, err
	}
	const base = ` FROM batch b JOIN contract c ON c.id=b.contract_id JOIN project p ON p.id=c.project_id JOIN customer cu ON cu.id=c.customer_id
 WHERE ($1 OR b.factory_id=ANY($2)) AND b.status<>'done' AND ($3='' OR b.status=$3)
 AND ($4='' OR b.lot_no ILIKE $4 OR p.full_name ILIKE $4 OR cu.name ILIKE $4)`
	args := []any{sc.All, sc.FactoryIDs, status, likeKeyword(keyword)}
	var total int
	if err = s.db.QueryRow(ctx, `SELECT count(*)::int`+base, args...).Scan(&total); err != nil {
		return nil, nil, 0, err
	}
	rows, err := s.db.Query(ctx, `SELECT b.id,b.lot_no,p.full_name,cu.name,b.workshop,b.urgency,b.status,b.created_at,b.required_date,b.plan_cut_date,b.item_qty,b.item_area::float8,
 (SELECT count(*)::int FROM label l WHERE l.batch_id=b.id AND NOT l.cancelled)`+base+`
 ORDER BY coalesce(b.required_date<$5::date,false) DESC,(b.urgency='urgent') DESC,b.required_date ASC NULLS LAST,b.id LIMIT $6 OFFSET $7`, append(args, today, size, (page-1)*size)...)
	if err != nil {
		return nil, nil, 0, err
	}
	defer rows.Close()
	items := []WorkOrder{}
	for rows.Next() {
		var r WorkOrder
		if err = rows.Scan(&r.BatchID, &r.LotNo, &r.ProjectName, &r.CustomerName, &r.Workshop, &r.Urgency, &r.Status, &r.OrderDate, &r.RequiredDate, &r.PlanCutDate, &r.ItemQty, &r.ItemArea, &r.LabelQty); err != nil {
			return nil, nil, 0, err
		}
		r.Jobs = make([]JobCell, len(jobs))
		for i, j := range jobs {
			r.Jobs[i].JobID = j.ID
		}
		items = append(items, r)
	}
	if err = rows.Err(); err != nil {
		return nil, nil, 0, err
	}
	if len(items) == 0 {
		return jobs, items, total, nil
	}
	ids := make([]int64, 0, len(items))
	idx := map[int64]int{}
	jobIdx := map[int64]int{}
	for i, r := range items {
		ids = append(ids, r.BatchID)
		idx[r.BatchID] = i
	}
	for i, j := range jobs {
		jobIdx[j.ID] = i
	}
	rows2, err := s.db.Query(ctx, `SELECT l.batch_id,sr.job_id,count(*)::int FROM scan_record sr JOIN label l ON l.id=sr.label_id AND NOT l.cancelled WHERE l.batch_id=ANY($1) GROUP BY l.batch_id,sr.job_id`, ids)
	if err != nil {
		return nil, nil, 0, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var bid, jid int64
		var qty int
		if err = rows2.Scan(&bid, &jid, &qty); err != nil {
			return nil, nil, 0, err
		}
		if k, ok := jobIdx[jid]; ok {
			r := &items[idx[bid]]
			r.Jobs[k].Qty = qty
			r.Jobs[k].Pct = pct(qty, r.LabelQty)
		}
	}
	return jobs, items, total, rows2.Err()
}

type BoardMetric struct {
	Qty  int     `json:"qty"`
	Area float64 `json:"area"`
}
type BoardDay struct {
	Date string `json:"date"`
	BoardMetric
}
type BoardJob struct {
	JobID int64  `json:"jobId"`
	Code  string `json:"code"`
	Name  string `json:"name"`
	BoardMetric
}
type BoardMachine struct {
	MachineID int64  `json:"machineId"`
	Name      string `json:"name"`
	BoardMetric
}
type BoardBacklog struct {
	Workshop   string  `json:"workshop"`
	NoScanQty  int     `json:"noScanQty"`
	NoScanArea float64 `json:"noScanArea"`
}
type BoardProject struct {
	ProjectName string  `json:"projectName"`
	LotNos      string  `json:"lotNos"`
	PlanQty     int     `json:"planQty"`
	PlanArea    float64 `json:"planArea"`
	FinishQty   int     `json:"finishQty"`
	FinishArea  float64 `json:"finishArea"`
	Pct         int     `json:"pct"`
}
type BoardKPI struct {
	InProductionBatches int     `json:"inProductionBatches"`
	OpenBatches         int     `json:"openBatches"`
	OverdueBatches      int     `json:"overdueBatches"`
	TodayScanQty        int     `json:"todayScanQty"`
	TodayScanArea       float64 `json:"todayScanArea"`
	TodayShipQty        int     `json:"todayShipQty"`
	TodayShipArea       float64 `json:"todayShipArea"`
	SemiStockQty        int     `json:"semiStockQty"`
	SemiStockArea       float64 `json:"semiStockArea"`
	FinishedStockQty    int     `json:"finishedStockQty"`
	FinishedStockArea   float64 `json:"finishedStockArea"`
}
type Cockpit struct {
	KPI             BoardKPI       `json:"kpi"`
	WeeklySheet     []BoardDay     `json:"weeklySheet"`
	WeeklySpray     []BoardDay     `json:"weeklySpray"`
	WeeklyShipment  []BoardDay     `json:"weeklyShipment"`
	JobOutput       []BoardJob     `json:"jobOutput"`
	MachineCapacity []BoardMachine `json:"machineCapacity"`
	SheetDaily      struct {
		AllQty   int     `json:"allQty"`
		AllArea  float64 `json:"allArea"`
		DoneQty  int     `json:"doneQty"`
		DoneArea float64 `json:"doneArea"`
		QtyRate  float64 `json:"qtyRate"`
		AreaRate float64 `json:"areaRate"`
	} `json:"sheetDaily"`
	WorkshopBacklog []BoardBacklog `json:"workshopBacklog"`
	Projects        []BoardProject `json:"projects"`
}

func (s *Store) Cockpit(ctx context.Context, sc auth.Scope, today time.Time) (Cockpit, error) {
	var out Cockpit
	args := []any{sc.All, sc.FactoryIDs, today}
	err := s.db.QueryRow(ctx, `SELECT count(*) FILTER (WHERE status='in_production')::int,count(*) FILTER (WHERE status<>'done')::int,count(*) FILTER (WHERE status<>'done' AND required_date<$3::date)::int FROM batch WHERE ($1 OR factory_id=ANY($2))`, args...).Scan(&out.KPI.InProductionBatches, &out.KPI.OpenBatches, &out.KPI.OverdueBatches)
	if err != nil {
		return out, err
	}
	// One row per active label; stock is defined by presence of inbound and absence of outbound scan.
	err = s.db.QueryRow(ctx, `SELECT count(*) FILTER (WHERE s20 AND NOT s40)::int,coalesce(sum(piece_area) FILTER (WHERE s20 AND NOT s40),0)::float8,
 count(*) FILTER (WHERE s40 AND NOT s60)::int,coalesce(sum(piece_area) FILTER (WHERE s40 AND NOT s60),0)::float8 FROM (
 SELECT bi.piece_area,exists(SELECT 1 FROM scan_record sr JOIN job j ON j.id=sr.job_id WHERE sr.label_id=l.id AND j.code='20') s20,
 exists(SELECT 1 FROM scan_record sr JOIN job j ON j.id=sr.job_id WHERE sr.label_id=l.id AND j.code='40') s40,
 exists(SELECT 1 FROM scan_record sr JOIN job j ON j.id=sr.job_id WHERE sr.label_id=l.id AND j.code='60') s60
 FROM label l JOIN batch b ON b.id=l.batch_id JOIN batch_item bi ON bi.id=l.batch_item_id WHERE NOT l.cancelled AND ($1 OR b.factory_id=ANY($2))) x`, sc.All, sc.FactoryIDs).Scan(&out.KPI.SemiStockQty, &out.KPI.SemiStockArea, &out.KPI.FinishedStockQty, &out.KPI.FinishedStockArea)
	if err != nil {
		return out, err
	}
	rows, err := s.db.Query(ctx, `SELECT sr.scanned_at::date,j.code,sr.job_id,count(*)::int,coalesce(sum(bi.piece_area),0)::float8 FROM scan_record sr JOIN label l ON l.id=sr.label_id AND NOT l.cancelled JOIN batch b ON b.id=l.batch_id JOIN batch_item bi ON bi.id=l.batch_item_id JOIN job j ON j.id=sr.job_id WHERE ($1 OR b.factory_id=ANY($2)) AND sr.scanned_at::date BETWEEN $3::date-6 AND $3::date GROUP BY 1,2,3`, args...)
	if err != nil {
		return out, err
	}
	type scanDay struct {
		date  string
		code  string
		jobID int64
		qty   int
		area  float64
	}
	scans := []scanDay{}
	for rows.Next() {
		var d time.Time
		var r scanDay
		if err = rows.Scan(&d, &r.code, &r.jobID, &r.qty, &r.area); err != nil {
			rows.Close()
			return out, err
		}
		r.date = d.Format("2006-01-02")
		scans = append(scans, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	jobs, err := s.Jobs(ctx)
	if err != nil {
		return out, err
	}
	for _, j := range jobs {
		v := BoardJob{JobID: j.ID, Code: j.Code, Name: j.Name}
		for _, r := range scans {
			if r.date == today.Format("2006-01-02") && r.jobID == j.ID {
				v.Qty += r.qty
				v.Area += r.area
				out.KPI.TodayScanQty += r.qty
				out.KPI.TodayScanArea += r.area
			}
		}
		out.JobOutput = append(out.JobOutput, v)
	}
	for i := 6; i >= 0; i-- {
		day := today.AddDate(0, 0, -i).Format("2006-01-02")
		sheet, spray, ship := BoardDay{Date: day}, BoardDay{Date: day}, BoardDay{Date: day}
		for _, r := range scans {
			if r.date != day {
				continue
			}
			switch r.code {
			case "20":
				sheet.Qty += r.qty
				sheet.Area += r.area
			case "40":
				spray.Qty += r.qty
				spray.Area += r.area
			}
		}
		out.WeeklySheet = append(out.WeeklySheet, sheet)
		out.WeeklySpray = append(out.WeeklySpray, spray)
		out.WeeklyShipment = append(out.WeeklyShipment, ship)
	}
	// Shipment uses delivery notes created on each date (not scans at job 60).
	shipRows, shipErr := s.db.Query(ctx, `SELECT dn.created_at::date,count(dnl.label_id)::int,coalesce(sum(bi.piece_area),0)::float8 FROM delivery_note dn JOIN delivery_note_label dnl ON dnl.note_id=dn.id JOIN label l ON l.id=dnl.label_id AND NOT l.cancelled JOIN batch_item bi ON bi.id=l.batch_item_id WHERE ($1 OR dn.factory_id=ANY($2)) AND dn.created_at::date BETWEEN $3::date-6 AND $3::date GROUP BY 1`, args...)
	if shipErr != nil {
		return out, shipErr
	}
	for shipRows.Next() {
		var d time.Time
		var qty int
		var a float64
		if err = shipRows.Scan(&d, &qty, &a); err != nil {
			shipRows.Close()
			return out, err
		}
		for i := range out.WeeklyShipment {
			if out.WeeklyShipment[i].Date == d.Format("2006-01-02") {
				out.WeeklyShipment[i].Qty = qty
				out.WeeklyShipment[i].Area = a
			}
		}
	}
	err = shipRows.Err()
	shipRows.Close()
	if err != nil {
		return out, err
	}
	err = s.db.QueryRow(ctx, `SELECT count(dnl.label_id)::int,coalesce(sum(bi.piece_area),0)::float8 FROM delivery_note dn JOIN delivery_note_label dnl ON dnl.note_id=dn.id JOIN label l ON l.id=dnl.label_id AND NOT l.cancelled JOIN batch_item bi ON bi.id=l.batch_item_id WHERE ($1 OR dn.factory_id=ANY($2)) AND dn.created_at::date=$3::date`, args...).Scan(&out.KPI.TodayShipQty, &out.KPI.TodayShipArea)
	if err != nil {
		return out, err
	}
	rows, err = s.db.Query(ctx, `SELECT m.id,m.name,count(sr.id)::int,coalesce(sum(bi.piece_area),0)::float8 FROM machine m LEFT JOIN scan_record sr ON sr.machine_id=m.id AND sr.scanned_at::date=$3::date AND EXISTS(SELECT 1 FROM label l WHERE l.id=sr.label_id AND NOT l.cancelled) LEFT JOIN label l ON l.id=sr.label_id LEFT JOIN batch_item bi ON bi.id=l.batch_item_id WHERE NOT m.is_stop AND ($1 OR m.factory_id=ANY($2)) GROUP BY m.id ORDER BY m.code`, args...)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var v BoardMachine
		if err = rows.Scan(&v.MachineID, &v.Name, &v.Qty, &v.Area); err != nil {
			rows.Close()
			return out, err
		}
		out.MachineCapacity = append(out.MachineCapacity, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	plans, err := s.DailyBoard(ctx, sc, today, "sheet_daily", "", nil)
	if err != nil {
		return out, err
	}
	for _, p := range plans {
		out.SheetDaily.AllQty += p.PlannedQty
		out.SheetDaily.AllArea += p.PlannedArea
		out.SheetDaily.DoneQty += p.DoneQty
		out.SheetDaily.DoneArea += p.DoneArea
	}
	if out.SheetDaily.AllQty > 0 {
		out.SheetDaily.QtyRate = float64(out.SheetDaily.DoneQty) * 100 / float64(out.SheetDaily.AllQty)
	}
	if out.SheetDaily.AllArea > 0 {
		out.SheetDaily.AreaRate = out.SheetDaily.DoneArea * 100 / out.SheetDaily.AllArea
	}
	rows, err = s.db.Query(ctx, `SELECT coalesce(nullif(b.workshop,''),'未分配'),count(l.id)::int,coalesce(sum(bi.piece_area),0)::float8 FROM batch b JOIN label l ON l.batch_id=b.id AND NOT l.cancelled JOIN batch_item bi ON bi.id=l.batch_item_id WHERE b.status<>'done' AND ($1 OR b.factory_id=ANY($2)) AND NOT EXISTS(SELECT 1 FROM scan_record sr JOIN job j ON j.id=sr.job_id WHERE sr.label_id=l.id AND j.code='20') GROUP BY 1 ORDER BY 1`, sc.All, sc.FactoryIDs)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var v BoardBacklog
		if err = rows.Scan(&v.Workshop, &v.NoScanQty, &v.NoScanArea); err != nil {
			rows.Close()
			return out, err
		}
		out.WorkshopBacklog = append(out.WorkshopBacklog, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = s.db.Query(ctx, `WITH top AS (SELECT p.id,p.full_name,min(b.required_date) AS due FROM batch b JOIN contract c ON c.id=b.contract_id JOIN project p ON p.id=c.project_id WHERE b.status<>'done' AND ($1 OR b.factory_id=ANY($2)) GROUP BY p.id ORDER BY due ASC NULLS LAST,p.id LIMIT 10),
 batches AS (SELECT t.id,t.full_name,b.id bid,b.lot_no,b.item_qty,b.item_area FROM top t JOIN contract c ON c.project_id=t.id JOIN batch b ON b.contract_id=c.id WHERE b.status<>'done' AND ($1 OR b.factory_id=ANY($2))),
 finished AS (SELECT b.bid,count(sr.id)::int qty,coalesce(sum(bi.piece_area) FILTER (WHERE sr.id IS NOT NULL),0)::float8 area FROM batches b LEFT JOIN label l ON l.batch_id=b.bid AND NOT l.cancelled LEFT JOIN batch_item bi ON bi.id=l.batch_item_id LEFT JOIN scan_record sr ON sr.label_id=l.id AND sr.job_id=(SELECT id FROM job WHERE code='55') GROUP BY b.bid)
 SELECT b.full_name,string_agg(b.lot_no,', ' ORDER BY b.lot_no),sum(b.item_qty)::int,sum(b.item_area)::float8,sum(f.qty)::int,sum(f.area)::float8 FROM batches b JOIN finished f ON f.bid=b.bid GROUP BY b.id,b.full_name ORDER BY min((SELECT required_date FROM batch WHERE id=b.bid)) ASC NULLS LAST,b.id`, sc.All, sc.FactoryIDs)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var v BoardProject
		if err = rows.Scan(&v.ProjectName, &v.LotNos, &v.PlanQty, &v.PlanArea, &v.FinishQty, &v.FinishArea); err != nil {
			rows.Close()
			return out, err
		}
		v.Pct = pct(v.FinishQty, v.PlanQty)
		out.Projects = append(out.Projects, v)
	}
	err = rows.Err()
	rows.Close()
	return out, err
}
