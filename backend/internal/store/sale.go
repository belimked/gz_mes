package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

type Customer struct {
	ID                                                              int64
	FactoryID                                                       int64
	Name, Contacts, Phone, Address, Bank, Account, EasCode, EasName string
	IsInternal, Confirmed                                           bool
	CreatedBy                                                       string
	CreatedAt                                                       time.Time
}

const customerSelect = `SELECT c.id, c.factory_id, c.name, c.contacts, c.phone, c.address, c.bank, c.account, c.eas_code, c.eas_name, c.is_internal, c.confirmed, u.display_name, c.created_at FROM customer c JOIN app_user u ON u.id = c.created_by`

func (s *Store) ListCustomers(ctx context.Context, sc auth.Scope, keyword string, page, size int) ([]Customer, int64, error) {
	where := " WHERE ($1 OR c.factory_id = ANY($2))"
	args := []any{sc.All, sc.FactoryIDs}
	if kw := strings.TrimSpace(keyword); kw != "" {
		args = append(args, "%"+escapeLike(kw)+"%")
		where += fmt.Sprintf(" AND (c.name ILIKE $%[1]d OR c.eas_code ILIKE $%[1]d OR c.eas_name ILIKE $%[1]d)", len(args))
	}
	var total int64
	if err := s.db.QueryRow(ctx, "SELECT count(*) FROM customer c"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, customerSelect+where+fmt.Sprintf(" ORDER BY c.created_at DESC, c.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Customer])
	return items, total, err
}

type CustomerCreate struct {
	FactoryID                                                       int64
	Name, Contacts, Phone, Address, Bank, Account, EasCode, EasName string
	IsInternal                                                      bool
	CreatedBy                                                       int64
}

func (s *Store) CreateCustomer(ctx context.Context, in CustomerCreate) (*Customer, error) {
	var id int64
	err := s.db.QueryRow(ctx, `INSERT INTO customer (factory_id, name, contacts, phone, address, bank, account, eas_code, eas_name, is_internal, created_by)
 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`, in.FactoryID, in.Name, in.Contacts, in.Phone, in.Address, in.Bank, in.Account, in.EasCode, in.EasName, in.IsInternal, in.CreatedBy).Scan(&id)
	if err != nil {
		return nil, mapErr(err)
	}
	return s.getCustomer(ctx, auth.Scope{All: true}, id)
}

func (s *Store) getCustomer(ctx context.Context, sc auth.Scope, id int64) (*Customer, error) {
	rows, err := s.db.Query(ctx, customerSelect+" WHERE c.id=$1 AND ($2 OR c.factory_id=ANY($3))", id, sc.All, sc.FactoryIDs)
	if err != nil {
		return nil, err
	}
	item, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Customer])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &item, err
}

func (s *Store) ConfirmCustomer(ctx context.Context, sc auth.Scope, id int64) (*Customer, error) {
	tag, err := s.db.Exec(ctx, "UPDATE customer SET confirmed=true WHERE id=$1 AND NOT confirmed AND ($2 OR factory_id=ANY($3))", id, sc.All, sc.FactoryIDs)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		if _, err := s.getCustomer(ctx, sc, id); err != nil {
			return nil, err
		}
		return nil, ErrConflict
	}
	return s.getCustomer(ctx, sc, id)
}

type Contract struct {
	ID, FactoryID                                                                                                       int64
	ContractNo, CustomerContractNo, InnerContractNo                                                                     string
	CustomerID                                                                                                          int64
	CustomerName                                                                                                        string
	ProjectID                                                                                                           int64
	ProjectName, Salesman                                                                                               string
	OrderQty                                                                                                            int
	OrderArea                                                                                                           float64
	PricingManner, Payment, PaymentPoint, Texture, PaintType, Packing, Receiver, ReceiverPhone, ReceiverAddress, Remark string
	CustomerValuationAudited, OutsourceValuationAudited                                                                 bool
	CreatedBy                                                                                                           string
	CreatedAt                                                                                                           time.Time
}

const contractSelect = `SELECT c.id, c.factory_id, c.contract_no, c.customer_contract_no, c.inner_contract_no,
 c.customer_id, cu.name, c.project_id, p.full_name, c.salesman, c.order_qty, c.order_area::float8,
 c.pricing_manner, c.payment, c.payment_point, c.texture, c.paint_type, c.packing, c.receiver,
 c.receiver_phone, c.receiver_address, c.remark, c.customer_valuation_audited,
 c.outsource_valuation_audited, u.display_name, c.created_at
 FROM contract c JOIN customer cu ON cu.id=c.customer_id JOIN project p ON p.id=c.project_id JOIN app_user u ON u.id=c.created_by`
const contractJoins = ` FROM contract c JOIN customer cu ON cu.id=c.customer_id JOIN project p ON p.id=c.project_id`

type ContractFilter struct {
	Keyword               string
	CustomerID, ProjectID *int64
	Begin, End            *time.Time
}

func (s *Store) ListContracts(ctx context.Context, sc auth.Scope, f ContractFilter, page, size int) ([]Contract, int64, error) {
	where := " WHERE ($1 OR c.factory_id=ANY($2))"
	args := []any{sc.All, sc.FactoryIDs}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		args = append(args, "%"+escapeLike(kw)+"%")
		where += fmt.Sprintf(` AND (c.contract_no ILIKE $%[1]d OR c.customer_contract_no ILIKE $%[1]d OR c.inner_contract_no ILIKE $%[1]d OR cu.name ILIKE $%[1]d OR p.full_name ILIKE $%[1]d OR c.salesman ILIKE $%[1]d)`, len(args))
	}
	if f.CustomerID != nil {
		args = append(args, *f.CustomerID)
		where += fmt.Sprintf(" AND c.customer_id=$%d", len(args))
	}
	if f.ProjectID != nil {
		args = append(args, *f.ProjectID)
		where += fmt.Sprintf(" AND c.project_id=$%d", len(args))
	}
	if f.Begin != nil {
		args = append(args, *f.Begin)
		where += fmt.Sprintf(" AND c.created_at >= $%d", len(args))
	}
	if f.End != nil {
		args = append(args, *f.End)
		where += fmt.Sprintf(" AND c.created_at < $%d", len(args))
	}
	var total int64
	if err := s.db.QueryRow(ctx, "SELECT count(*)"+contractJoins+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, size, (page-1)*size)
	rows, err := s.db.Query(ctx, contractSelect+where+fmt.Sprintf(" ORDER BY c.created_at DESC, c.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Contract])
	return items, total, err
}

type ContractCreate struct {
	FactoryID                                                                                                           int64
	ContractNo, CustomerContractNo, InnerContractNo                                                                     string
	CustomerID, ProjectID                                                                                               int64
	Salesman                                                                                                            string
	OrderQty                                                                                                            int
	OrderArea                                                                                                           float64
	PricingManner, Payment, PaymentPoint, Texture, PaintType, Packing, Receiver, ReceiverPhone, ReceiverAddress, Remark string
	CreatedBy                                                                                                           int64
}

func (s *Store) CreateContract(ctx context.Context, in ContractCreate) (*Contract, error) {
	// The insert only succeeds when both references belong to the contract's factory.
	var id int64
	err := s.db.QueryRow(ctx, `INSERT INTO contract (factory_id, contract_no, customer_contract_no, inner_contract_no, customer_id, project_id, salesman, order_qty, order_area, pricing_manner, payment, payment_point, texture, paint_type, packing, receiver, receiver_phone, receiver_address, remark, created_by)
 SELECT $1,$2,$3,$4,cu.id,p.id,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20
 FROM customer cu CROSS JOIN project p WHERE cu.id=$5 AND p.id=$6 AND cu.factory_id=$1 AND p.factory_id=$1 RETURNING id`,
		in.FactoryID, in.ContractNo, in.CustomerContractNo, in.InnerContractNo, in.CustomerID, in.ProjectID, in.Salesman, in.OrderQty, in.OrderArea, in.PricingManner, in.Payment, in.PaymentPoint, in.Texture, in.PaintType, in.Packing, in.Receiver, in.ReceiverPhone, in.ReceiverAddress, in.Remark, in.CreatedBy).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBadRef
	}
	if err != nil {
		return nil, mapErr(err)
	}
	return s.GetContract(ctx, auth.Scope{All: true}, id)
}

func (s *Store) GetContract(ctx context.Context, sc auth.Scope, id int64) (*Contract, error) {
	rows, err := s.db.Query(ctx, contractSelect+" WHERE c.id=$1 AND ($2 OR c.factory_id=ANY($3))", id, sc.All, sc.FactoryIDs)
	if err != nil {
		return nil, err
	}
	item, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Contract])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &item, err
}

func (s *Store) AuditContractValuation(ctx context.Context, sc auth.Scope, id int64, kind string) (*Contract, error) {
	column := "customer_valuation_audited"
	switch kind {
	case "customer":
	case "outsource":
		column = "outsource_valuation_audited"
	default:
		return nil, fmt.Errorf("invalid valuation kind: %q", kind)
	}
	tag, err := s.db.Exec(ctx, "UPDATE contract SET "+column+"=true WHERE id=$1 AND NOT "+column+" AND ($2 OR factory_id=ANY($3))", id, sc.All, sc.FactoryIDs)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		if _, err := s.GetContract(ctx, sc, id); err != nil {
			return nil, err
		}
		return nil, ErrConflict
	}
	return s.GetContract(ctx, sc, id)
}
