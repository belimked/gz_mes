package store

import (
	"context"
	"sort"
	"time"

	"github.com/belimked/gz_mes/backend/internal/auth"
)

type WorkshopOutputRow struct {
	Period             string    `json:"period"`
	BatchCount         int       `json:"batchCount"`
	BatchArea          float64   `json:"batchArea"`
	ScheduledCount     int       `json:"scheduledCount"`
	ScheduledArea      float64   `json:"scheduledArea"`
	SheetQty           int       `json:"sheetQty"`
	SheetArea          float64   `json:"sheetArea"`
	SheetOwnArea       float64   `json:"sheetOwnArea"`
	SheetOutsourceArea float64   `json:"sheetOutsourceArea"`
	SprayQty           int       `json:"sprayQty"`
	SprayArea          float64   `json:"sprayArea"`
	DeliveredQty       int       `json:"deliveredQty"`
	DeliveredArea      float64   `json:"deliveredArea"`
	WorkshopSheetArea  []float64 `json:"workshopSheetArea"`
}
type WorkshopComparison struct {
	Workshop     string   `json:"workshop"`
	PrevPrevArea float64  `json:"prevPrevArea"`
	PrevArea     float64  `json:"prevArea"`
	CurrentArea  float64  `json:"currentArea"`
	ChainRate    *float64 `json:"chainRate"`
}
type WorkshopOutputFilter struct {
	From, To  time.Time
	Month     bool
	FactoryID *int64
	Workshop  string
}

// Event dates use the database session's ::date conversion, as in the other reports.
const workshopEvents = `WITH events AS (
 SELECT b.created_at::date AS d, 1 AS bc, coalesce(nullif(b.order_area,0),b.item_area)::float8 AS ba, 0 AS pc, 0::float8 AS pa, 0 AS sq, 0::float8 AS sa, 0::float8 AS own, 0::float8 AS ext, 0 AS pq, 0::float8 AS pra, 0 AS dq, 0::float8 AS da, NULL::text AS workshop
 FROM batch b WHERE ($4::bigint IS NULL OR b.factory_id=$4) AND ($1 OR b.factory_id=ANY($2)) AND ($5='' OR coalesce(nullif(b.workshop,''),'未分配')=$5)
 UNION ALL
 SELECT p.first_at::date,0,0,1,b.item_area::float8,0,0,0,0,0,0,0,0,NULL
 FROM batch b JOIN LATERAL (SELECT min(audited_at) AS first_at FROM plan WHERE batch_id=b.id AND audit_state='audited' AND stop_state<>'stopped') p ON p.first_at IS NOT NULL
 WHERE ($4::bigint IS NULL OR b.factory_id=$4) AND ($1 OR b.factory_id=ANY($2)) AND ($5='' OR coalesce(nullif(b.workshop,''),'未分配')=$5)
 UNION ALL
 SELECT sr.scanned_at::date,0,0,0,0,CASE WHEN j.code='20' THEN 1 ELSE 0 END,CASE WHEN j.code='20' THEN bi.piece_area::float8 ELSE 0 END,
 CASE WHEN j.code='20' AND b.outsource_factory_id IS NULL THEN bi.piece_area::float8 ELSE 0 END,
 CASE WHEN j.code='20' AND b.outsource_factory_id IS NOT NULL THEN bi.piece_area::float8 ELSE 0 END,
 CASE WHEN j.code='40' THEN 1 ELSE 0 END,CASE WHEN j.code='40' THEN bi.piece_area::float8 ELSE 0 END,0,0,
 CASE WHEN j.code='20' THEN coalesce(nullif(b.workshop,''),'未分配') ELSE NULL END
 FROM scan_record sr JOIN job j ON j.id=sr.job_id JOIN label l ON l.id=sr.label_id AND NOT l.cancelled
 JOIN batch_item bi ON bi.id=l.batch_item_id JOIN batch b ON b.id=sr.batch_id
 WHERE j.code IN ('20','40') AND ($4::bigint IS NULL OR b.factory_id=$4) AND ($1 OR b.factory_id=ANY($2)) AND ($5='' OR coalesce(nullif(b.workshop,''),'未分配')=$5)
 UNION ALL
 SELECT dn.delivered_at,0,0,0,0,0,0,0,0,0,0,dn.qty,dn.area::float8,NULL
 FROM delivery_note dn JOIN batch b ON b.id=dn.batch_id
 WHERE ($4::bigint IS NULL OR dn.factory_id=$4) AND ($1 OR dn.factory_id=ANY($2)) AND ($5='' OR coalesce(nullif(b.workshop,''),'未分配')=$5)
)`

func (s *Store) WorkshopOutput(ctx context.Context, sc auth.Scope, f WorkshopOutputFilter) ([]WorkshopOutputRow, []string, WorkshopOutputRow, []WorkshopComparison, error) {
	args := []any{sc.All, sc.FactoryIDs, f.Month, f.FactoryID, f.Workshop, f.From, f.To}
	workshops := []string{}
	rows, err := s.db.Query(ctx, workshopEvents+` SELECT workshop FROM events WHERE workshop IS NOT NULL AND d >= $6::date AND d < ($7::date + CASE WHEN $3 THEN interval '1 month' ELSE interval '1 day' END) GROUP BY workshop ORDER BY workshop='未分配',workshop`, args...)
	if err != nil {
		return nil, nil, WorkshopOutputRow{}, nil, err
	}
	for rows.Next() {
		var w string
		if err = rows.Scan(&w); err != nil {
			rows.Close()
			return nil, nil, WorkshopOutputRow{}, nil, err
		}
		workshops = append(workshops, w)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, WorkshopOutputRow{}, nil, err
	}
	sql := workshopEvents + ` , periods AS (SELECT generate_series(date_trunc(CASE WHEN $3 THEN 'month' ELSE 'day' END,$6::date),date_trunc(CASE WHEN $3 THEN 'month' ELSE 'day' END,$7::date),CASE WHEN $3 THEN interval '1 month' ELSE interval '1 day' END)::date AS d), grouped AS (
 SELECT date_trunc(CASE WHEN $3 THEN 'month' ELSE 'day' END,d)::date AS d,sum(bc)::int bc,sum(ba) ba,sum(pc)::int pc,sum(pa) pa,sum(sq)::int sq,sum(sa) sa,sum(own) own,sum(ext) ext,sum(pq)::int pq,sum(pra) pra,sum(dq)::int dq,sum(da) da
 FROM events WHERE d >= $6::date AND d < ($7::date + CASE WHEN $3 THEN interval '1 month' ELSE interval '1 day' END) GROUP BY 1)
 SELECT to_char(p.d,CASE WHEN $3 THEN 'YYYY-MM' ELSE 'YYYY-MM-DD' END),coalesce(g.bc,0),coalesce(g.ba,0),coalesce(g.pc,0),coalesce(g.pa,0),coalesce(g.sq,0),coalesce(g.sa,0),coalesce(g.own,0),coalesce(g.ext,0),coalesce(g.pq,0),coalesce(g.pra,0),coalesce(g.dq,0),coalesce(g.da,0) FROM periods p LEFT JOIN grouped g ON g.d=p.d ORDER BY p.d`
	rows, err = s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, nil, WorkshopOutputRow{}, nil, err
	}
	result := []WorkshopOutputRow{}
	total := WorkshopOutputRow{Period: "合计", WorkshopSheetArea: make([]float64, len(workshops))}
	for rows.Next() {
		var r WorkshopOutputRow
		err = rows.Scan(&r.Period, &r.BatchCount, &r.BatchArea, &r.ScheduledCount, &r.ScheduledArea, &r.SheetQty, &r.SheetArea, &r.SheetOwnArea, &r.SheetOutsourceArea, &r.SprayQty, &r.SprayArea, &r.DeliveredQty, &r.DeliveredArea)
		if err != nil {
			rows.Close()
			return nil, nil, total, nil, err
		}
		r.WorkshopSheetArea = make([]float64, len(workshops))
		result = append(result, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, total, nil, err
	}
	// Distribution is queried independently so every returned period retains a zero-filled vector.
	rows, err = s.db.Query(ctx, workshopEvents+` SELECT to_char(date_trunc(CASE WHEN $3 THEN 'month' ELSE 'day' END,d),CASE WHEN $3 THEN 'YYYY-MM' ELSE 'YYYY-MM-DD' END),workshop,sum(sa)::float8 FROM events WHERE workshop IS NOT NULL AND d >= $6::date AND d < ($7::date + CASE WHEN $3 THEN interval '1 month' ELSE interval '1 day' END) GROUP BY 1,2`, args...)
	if err != nil {
		return nil, nil, total, nil, err
	}
	for rows.Next() {
		var p, w string
		var area float64
		if err = rows.Scan(&p, &w, &area); err != nil {
			rows.Close()
			return nil, nil, total, nil, err
		}
		for i := range result {
			if result[i].Period == p {
				for k, name := range workshops {
					if name == w {
						result[i].WorkshopSheetArea[k] = area
					}
				}
			}
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, total, nil, err
	}
	for _, r := range result {
		total.BatchCount += r.BatchCount
		total.BatchArea += r.BatchArea
		total.ScheduledCount += r.ScheduledCount
		total.ScheduledArea += r.ScheduledArea
		total.SheetQty += r.SheetQty
		total.SheetArea += r.SheetArea
		total.SheetOwnArea += r.SheetOwnArea
		total.SheetOutsourceArea += r.SheetOutsourceArea
		total.SprayQty += r.SprayQty
		total.SprayArea += r.SprayArea
		total.DeliveredQty += r.DeliveredQty
		total.DeliveredArea += r.DeliveredArea
		for k, v := range r.WorkshopSheetArea {
			total.WorkshopSheetArea[k] += v
		}
	}
	var comparison []WorkshopComparison
	if f.Month {
		comparison = []WorkshopComparison{}
		// The comparison is not restricted by the report's from bound.
		rows, err = s.db.Query(ctx, workshopEvents+` SELECT workshop,to_char(date_trunc('month',d),'YYYY-MM'),sum(sa)::float8 FROM events WHERE workshop IS NOT NULL AND $3::bool AND $6::date IS NOT NULL AND d>=date_trunc('month',$7::date)-interval '2 months' AND d<date_trunc('month',$7::date)+interval '1 month' GROUP BY 1,2`, args...)
		if err != nil {
			return nil, nil, total, nil, err
		}
		byName := map[string]*WorkshopComparison{}
		names := []string{}
		current := f.To.Format("2006-01")
		first := time.Date(f.To.Year(), f.To.Month(), 1, 0, 0, 0, 0, time.UTC)
		for rows.Next() {
			var w, p string
			var area float64
			if err = rows.Scan(&w, &p, &area); err != nil {
				rows.Close()
				return nil, nil, total, nil, err
			}
			c := byName[w]
			if c == nil {
				c = &WorkshopComparison{Workshop: w}
				byName[w] = c
				names = append(names, w)
			}
			switch p {
			case first.AddDate(0, -2, 0).Format("2006-01"):
				c.PrevPrevArea = area
			case first.AddDate(0, -1, 0).Format("2006-01"):
				c.PrevArea = area
			case current:
				c.CurrentArea = area
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, nil, total, nil, err
		}
		// Include zero-output workshops from the requested window.
		for _, w := range workshops {
			if byName[w] == nil {
				byName[w] = &WorkshopComparison{Workshop: w}
				names = append(names, w)
			}
		}
		sort.Slice(names, func(i, j int) bool {
			if names[i] == "未分配" {
				return false
			}
			if names[j] == "未分配" {
				return true
			}
			return names[i] < names[j]
		})
		sum := WorkshopComparison{Workshop: "合计"}
		for _, w := range names {
			c := byName[w]
			if c.PrevArea != 0 {
				v := (c.CurrentArea - c.PrevArea) / c.PrevArea * 100
				c.ChainRate = &v
			}
			comparison = append(comparison, *c)
			sum.PrevPrevArea += c.PrevPrevArea
			sum.PrevArea += c.PrevArea
			sum.CurrentArea += c.CurrentArea
		}
		if sum.PrevArea != 0 {
			v := (sum.CurrentArea - sum.PrevArea) / sum.PrevArea * 100
			sum.ChainRate = &v
		}
		comparison = append(comparison, sum)
	}
	return result, workshops, total, comparison, nil
}
