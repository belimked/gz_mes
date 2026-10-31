package server

import (
	"context"
	"errors"
	"strings"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/store"
)

func optionalString(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

func (s *Server) ListCustomers(ctx context.Context, req api.ListCustomersRequestObject) (api.ListCustomersResponseObject, error) {
	page, size, kw := 1, 20, ""
	if req.Params.Page != nil {
		page = *req.Params.Page
	}
	if req.Params.Size != nil {
		size = *req.Params.Size
	}
	if req.Params.Keyword != nil {
		kw = *req.Params.Keyword
	}
	items, total, err := s.store.ListCustomers(ctx, auth.FromContext(ctx).Scope, kw, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListCustomers200JSONResponse{Items: make([]api.Customer, 0, len(items)), Total: total}
	for i := range items {
		out.Items = append(out.Items, toAPICustomer(&items[i]))
	}
	return out, nil
}

func (s *Server) CreateCustomer(ctx context.Context, req api.CreateCustomerRequestObject) (api.CreateCustomerResponseObject, error) {
	p := auth.FromContext(ctx)
	b := req.Body
	if !p.Scope.Allows(b.FactoryId) {
		return api.CreateCustomer403JSONResponse(apiErr("forbidden_factory", "无权在该厂区创建数据")), nil
	}
	in := store.CustomerCreate{FactoryID: b.FactoryId, Name: strings.TrimSpace(b.Name), Contacts: optionalString(b.Contacts), Phone: optionalString(b.Phone), Address: optionalString(b.Address), Bank: optionalString(b.Bank), Account: optionalString(b.Account), EasCode: optionalString(b.EasCode), EasName: optionalString(b.EasName), CreatedBy: p.UserID}
	if b.IsInternal != nil {
		in.IsInternal = *b.IsInternal
	}
	if in.Name == "" {
		return api.CreateCustomer400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "客户名称不能为空")}, nil
	}
	item, err := s.store.CreateCustomer(ctx, in)
	switch {
	case errors.Is(err, store.ErrConflict):
		return api.CreateCustomer409JSONResponse(apiErr("duplicate_code", "客户名称已存在")), nil
	case errors.Is(err, store.ErrBadRef):
		return api.CreateCustomer400JSONResponse{ErrorJSONResponse: apiErr("invalid_factory", "厂区不存在")}, nil
	case err != nil:
		return nil, err
	}
	return api.CreateCustomer201JSONResponse(toAPICustomer(item)), nil
}

func (s *Server) ConfirmCustomer(ctx context.Context, req api.ConfirmCustomerRequestObject) (api.ConfirmCustomerResponseObject, error) {
	item, err := s.store.ConfirmCustomer(ctx, auth.FromContext(ctx).Scope, req.Id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.ConfirmCustomer404JSONResponse(apiErr("not_found", "客户不存在")), nil
	case errors.Is(err, store.ErrConflict):
		return api.ConfirmCustomer409JSONResponse(apiErr("already_confirmed", "客户已确认")), nil
	case err != nil:
		return nil, err
	}
	return api.ConfirmCustomer200JSONResponse(toAPICustomer(item)), nil
}

func toAPICustomer(c *store.Customer) api.Customer {
	return api.Customer{Id: c.ID, FactoryId: c.FactoryID, Name: c.Name, Contacts: c.Contacts, Phone: c.Phone, Address: c.Address, Bank: c.Bank, Account: c.Account, EasCode: c.EasCode, EasName: c.EasName, IsInternal: c.IsInternal, Confirmed: c.Confirmed, CreatedBy: c.CreatedBy, CreatedAt: c.CreatedAt}
}

func (s *Server) ListContracts(ctx context.Context, req api.ListContractsRequestObject) (api.ListContractsResponseObject, error) {
	page, size := 1, 20
	if req.Params.Page != nil {
		page = *req.Params.Page
	}
	if req.Params.Size != nil {
		size = *req.Params.Size
	}
	f := store.ContractFilter{CustomerID: req.Params.CustomerId, ProjectID: req.Params.ProjectId}
	if req.Params.Keyword != nil {
		f.Keyword = *req.Params.Keyword
	}
	if req.Params.BeginDate != nil {
		v := req.Params.BeginDate.Time
		f.Begin = &v
	}
	if req.Params.EndDate != nil {
		v := req.Params.EndDate.AddDate(0, 0, 1)
		f.End = &v
	}
	if f.Begin != nil && f.End != nil && !f.Begin.Before(*f.End) {
		return api.ListContracts400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "日期范围无效")}, nil
	}
	items, total, err := s.store.ListContracts(ctx, auth.FromContext(ctx).Scope, f, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListContracts200JSONResponse{Items: make([]api.Contract, 0, len(items)), Total: total}
	for i := range items {
		out.Items = append(out.Items, toAPIContract(&items[i]))
	}
	return out, nil
}

func (s *Server) CreateContract(ctx context.Context, req api.CreateContractRequestObject) (api.CreateContractResponseObject, error) {
	p := auth.FromContext(ctx)
	b := req.Body
	if !p.Scope.Allows(b.FactoryId) {
		return api.CreateContract403JSONResponse(apiErr("forbidden_factory", "无权在该厂区创建数据")), nil
	}
	in := store.ContractCreate{FactoryID: b.FactoryId, ContractNo: strings.TrimSpace(b.ContractNo), CustomerContractNo: optionalString(b.CustomerContractNo), InnerContractNo: optionalString(b.InnerContractNo), CustomerID: b.CustomerId, ProjectID: b.ProjectId, Salesman: optionalString(b.Salesman), PricingManner: optionalString(b.PricingManner), Payment: optionalString(b.Payment), PaymentPoint: optionalString(b.PaymentPoint), Texture: optionalString(b.Texture), PaintType: optionalString(b.PaintType), Packing: optionalString(b.Packing), Receiver: optionalString(b.Receiver), ReceiverPhone: optionalString(b.ReceiverPhone), ReceiverAddress: optionalString(b.ReceiverAddress), Remark: optionalString(b.Remark), CreatedBy: p.UserID}
	if b.OrderQty != nil {
		in.OrderQty = *b.OrderQty
	}
	if b.OrderArea != nil {
		in.OrderArea = *b.OrderArea
	}
	if in.ContractNo == "" {
		return api.CreateContract400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "合同单号不能为空")}, nil
	}
	item, err := s.store.CreateContract(ctx, in)
	switch {
	case errors.Is(err, store.ErrConflict):
		return api.CreateContract409JSONResponse(apiErr("duplicate_code", "合同单号已存在")), nil
	case errors.Is(err, store.ErrBadRef):
		return api.CreateContract400JSONResponse{ErrorJSONResponse: apiErr("invalid_reference", "厂区/客户/项目不存在或不属于该厂区")}, nil
	case err != nil:
		return nil, err
	}
	return api.CreateContract201JSONResponse(toAPIContract(item)), nil
}

func (s *Server) GetContract(ctx context.Context, req api.GetContractRequestObject) (api.GetContractResponseObject, error) {
	item, err := s.store.GetContract(ctx, auth.FromContext(ctx).Scope, req.Id)
	if errors.Is(err, store.ErrNotFound) {
		return api.GetContract404JSONResponse(apiErr("not_found", "合同不存在")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetContract200JSONResponse(toAPIContract(item)), nil
}

func (s *Server) AuditContractValuation(ctx context.Context, req api.AuditContractValuationRequestObject) (api.AuditContractValuationResponseObject, error) {
	kind := string(req.Body.Kind)
	if kind != "customer" && kind != "outsource" {
		return api.AuditContractValuation400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "审核类型无效")}, nil
	}
	item, err := s.store.AuditContractValuation(ctx, auth.FromContext(ctx).Scope, req.Id, kind)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.AuditContractValuation404JSONResponse(apiErr("not_found", "合同不存在")), nil
	case errors.Is(err, store.ErrConflict):
		return api.AuditContractValuation409JSONResponse(apiErr("already_audited", "计价已审核")), nil
	case err != nil:
		return nil, err
	}
	return api.AuditContractValuation200JSONResponse(toAPIContract(item)), nil
}

func toAPIContract(c *store.Contract) api.Contract {
	return api.Contract{Id: c.ID, FactoryId: c.FactoryID, ContractNo: c.ContractNo, CustomerContractNo: c.CustomerContractNo, InnerContractNo: c.InnerContractNo, CustomerId: c.CustomerID, CustomerName: c.CustomerName, ProjectId: c.ProjectID, ProjectName: c.ProjectName, Salesman: c.Salesman, OrderQty: c.OrderQty, OrderArea: c.OrderArea, PricingManner: c.PricingManner, Payment: c.Payment, PaymentPoint: c.PaymentPoint, Texture: c.Texture, PaintType: c.PaintType, Packing: c.Packing, Receiver: c.Receiver, ReceiverPhone: c.ReceiverPhone, ReceiverAddress: c.ReceiverAddress, Remark: c.Remark, CustomerValuationAudited: c.CustomerValuationAudited, OutsourceValuationAudited: c.OutsourceValuationAudited, CreatedBy: c.CreatedBy, CreatedAt: c.CreatedAt}
}
