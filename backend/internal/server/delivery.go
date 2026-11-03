package server

import (
	"context"
	"errors"
	"strings"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/store"
	"github.com/oapi-codegen/runtime/types"
)

func docMessage(err error) string {
	if _, message, ok := strings.Cut(err.Error(), ": "); ok {
		return message
	}
	return err.Error()
}
func validCodes(codes []string) bool {
	if len(codes) == 0 {
		return false
	}
	for _, code := range codes {
		if strings.TrimSpace(code) == "" {
			return false
		}
	}
	return true
}
func toAPIHandover(v *store.Handover) api.Handover {
	return api.Handover{Id: v.ID, HandoverNo: v.HandoverNo, FactoryId: v.FactoryID, BatchId: v.BatchID, LotNo: v.LotNo, ProjectName: v.ProjectName, JobId: v.JobID, JobCode: v.JobCode, JobName: v.JobName, Qty: v.Qty, Area: v.Area, Remark: v.Remark, PackedBy: v.PackedBy, PackedAt: v.PackedAt, Received: v.Received, ReceivedBy: v.ReceivedBy, ReceivedAt: v.ReceivedAt}
}
func handoverDetail(v *store.Handover, labels []store.Label) api.HandoverDetail {
	x := toAPIHandover(v)
	d := api.HandoverDetail{Id: x.Id, HandoverNo: x.HandoverNo, FactoryId: x.FactoryId, BatchId: x.BatchId, LotNo: x.LotNo, ProjectName: x.ProjectName, JobId: x.JobId, JobCode: x.JobCode, JobName: x.JobName, Qty: x.Qty, Area: x.Area, Remark: x.Remark, PackedBy: x.PackedBy, PackedAt: x.PackedAt, Received: x.Received, ReceivedBy: x.ReceivedBy, ReceivedAt: x.ReceivedAt, Labels: make([]api.Label, 0, len(labels))}
	for i := range labels {
		d.Labels = append(d.Labels, toAPILabel(&labels[i]))
	}
	return d
}
func toAPINote(v *store.DeliveryNote) api.DeliveryNote {
	return api.DeliveryNote{Id: v.ID, NoteNo: v.NoteNo, FactoryId: v.FactoryID, BatchId: v.BatchID, LotNo: v.LotNo, ProjectName: v.ProjectName, CustomerId: v.CustomerID, CustomerName: v.CustomerName, ContractNo: v.ContractNo, DeliveryBatch: v.DeliveryBatch, Qty: v.Qty, Area: v.Area, LogisticsCompany: v.LogisticsCompany, PlateNo: v.PlateNo, Driver: v.Driver, DriverPhone: v.DriverPhone, DeliveredAt: types.Date{Time: v.DeliveredAt}, Remark: v.Remark, Accounted: v.Accounted, AccountId: v.AccountID, PlanId: v.PlanID, PlanNo: v.PlanNo, CreatedBy: v.CreatedBy, CreatedAt: v.CreatedAt}
}
func noteDetail(v *store.DeliveryNote, labels []store.Label) api.DeliveryNoteDetail {
	x := toAPINote(v)
	d := api.DeliveryNoteDetail{Id: x.Id, NoteNo: x.NoteNo, FactoryId: x.FactoryId, BatchId: x.BatchId, LotNo: x.LotNo, ProjectName: x.ProjectName, CustomerId: x.CustomerId, CustomerName: x.CustomerName, ContractNo: x.ContractNo, DeliveryBatch: x.DeliveryBatch, Qty: x.Qty, Area: x.Area, LogisticsCompany: x.LogisticsCompany, PlateNo: x.PlateNo, Driver: x.Driver, DriverPhone: x.DriverPhone, DeliveredAt: x.DeliveredAt, Remark: x.Remark, Accounted: x.Accounted, AccountId: x.AccountId, PlanId: x.PlanId, PlanNo: x.PlanNo, CreatedBy: x.CreatedBy, CreatedAt: x.CreatedAt, Labels: make([]api.Label, 0, len(labels))}
	for i := range labels {
		d.Labels = append(d.Labels, toAPILabel(&labels[i]))
	}
	return d
}
func toAPIAccount(v *store.DeliverAccount) api.DeliverAccount {
	return api.DeliverAccount{Id: v.ID, AccountNo: v.AccountNo, FactoryId: v.FactoryID, CustomerId: v.CustomerID, CustomerName: v.CustomerName, NoteCount: v.NoteCount, TotalQty: v.TotalQty, TotalArea: v.TotalArea, UnitPrice: v.UnitPrice, ExtraAmount: v.ExtraAmount, ExtraRemark: v.ExtraRemark, Amount: v.Amount, Status: api.AccountStatus(v.Status), Remark: v.Remark, CreatedBy: v.CreatedBy, CreatedAt: v.CreatedAt, ConfirmedBy: v.ConfirmedBy, ConfirmedAt: v.ConfirmedAt}
}
func accountDetail(v *store.DeliverAccount, notes []store.DeliveryNote) api.DeliverAccountDetail {
	x := toAPIAccount(v)
	d := api.DeliverAccountDetail{Id: x.Id, AccountNo: x.AccountNo, FactoryId: x.FactoryId, CustomerId: x.CustomerId, CustomerName: x.CustomerName, NoteCount: x.NoteCount, TotalQty: x.TotalQty, TotalArea: x.TotalArea, UnitPrice: x.UnitPrice, ExtraAmount: x.ExtraAmount, ExtraRemark: x.ExtraRemark, Amount: x.Amount, Status: x.Status, Remark: x.Remark, CreatedBy: x.CreatedBy, CreatedAt: x.CreatedAt, ConfirmedBy: x.ConfirmedBy, ConfirmedAt: x.ConfirmedAt, Notes: make([]api.DeliveryNote, 0, len(notes))}
	for i := range notes {
		d.Notes = append(d.Notes, toAPINote(&notes[i]))
	}
	return d
}
func (s *Server) ListHandovers(ctx context.Context, req api.ListHandoversRequestObject) (api.ListHandoversResponseObject, error) {
	f := store.HandoverFilter{BatchID: req.Params.BatchId, JobID: req.Params.JobId, Received: req.Params.Received, Keyword: optionalString(req.Params.Keyword)}
	page, size := pageSize(req.Params.Page, req.Params.Size)
	items, total, err := s.store.ListHandovers(ctx, auth.FromContext(ctx).Scope, f, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListHandovers200JSONResponse{Items: make([]api.Handover, 0, len(items)), Total: total}
	for i := range items {
		out.Items = append(out.Items, toAPIHandover(&items[i]))
	}
	return out, nil
}
func (s *Server) CreateHandover(ctx context.Context, req api.CreateHandoverRequestObject) (api.CreateHandoverResponseObject, error) {
	b := req.Body
	if b == nil || !validCodes(b.BarCodes) || strings.TrimSpace(b.JobCode) == "" {
		return api.CreateHandover400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "条码和工序不能为空")}, nil
	}
	p := auth.FromContext(ctx)
	v, labels, err := s.store.CreateHandover(ctx, p.Scope, b.BatchId, strings.TrimSpace(b.JobCode), b.BarCodes, optionalString(b.Remark), p.UserID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.CreateHandover404JSONResponse(apiErr("not_found", "批次不存在")), nil
	case errors.Is(err, store.ErrBadRef):
		return api.CreateHandover400JSONResponse{ErrorJSONResponse: apiErr("invalid_reference", docMessage(err))}, nil
	case errors.Is(err, store.ErrNotScanned):
		return api.CreateHandover409JSONResponse(apiErr("not_scanned", docMessage(err))), nil
	case errors.Is(err, store.ErrLabelInHandover):
		return api.CreateHandover409JSONResponse(apiErr("label_in_handover", docMessage(err))), nil
	case err != nil:
		return nil, err
	}
	return api.CreateHandover201JSONResponse(handoverDetail(v, labels)), nil
}
func (s *Server) GetHandover(ctx context.Context, req api.GetHandoverRequestObject) (api.GetHandoverResponseObject, error) {
	v, labels, err := s.store.GetHandover(ctx, auth.FromContext(ctx).Scope, req.Id)
	if errors.Is(err, store.ErrNotFound) {
		return api.GetHandover404JSONResponse(apiErr("not_found", "交接单不存在")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetHandover200JSONResponse(handoverDetail(v, labels)), nil
}
func (s *Server) ReceiveHandover(ctx context.Context, req api.ReceiveHandoverRequestObject) (api.ReceiveHandoverResponseObject, error) {
	p := auth.FromContext(ctx)
	v, err := s.store.ReceiveHandover(ctx, p.Scope, req.Id, p.UserID)
	if errors.Is(err, store.ErrNotFound) {
		return api.ReceiveHandover404JSONResponse(apiErr("not_found", "交接单不存在")), nil
	}
	if errors.Is(err, store.ErrConflict) {
		return api.ReceiveHandover409JSONResponse(apiErr("already_received", "交接单已接收")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.ReceiveHandover200JSONResponse(toAPIHandover(v)), nil
}
func (s *Server) ListDeliveryNotes(ctx context.Context, req api.ListDeliveryNotesRequestObject) (api.ListDeliveryNotesResponseObject, error) {
	f := store.DeliveryNoteFilter{BatchID: req.Params.BatchId, CustomerID: req.Params.CustomerId, FactoryID: req.Params.FactoryId, Unplanned: req.Params.Unplanned, Keyword: optionalString(req.Params.Keyword), Accounted: req.Params.Accounted, Begin: dateValue(req.Params.BeginDate), End: dateValue(req.Params.EndDate)}
	if f.Begin != nil && f.End != nil && f.Begin.After(*f.End) {
		return api.ListDeliveryNotes400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "日期范围无效")}, nil
	}
	page, size := pageSize(req.Params.Page, req.Params.Size)
	items, total, err := s.store.ListDeliveryNotes(ctx, auth.FromContext(ctx).Scope, f, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListDeliveryNotes200JSONResponse{Items: make([]api.DeliveryNote, 0, len(items)), Total: total}
	for i := range items {
		out.Items = append(out.Items, toAPINote(&items[i]))
	}
	return out, nil
}
func (s *Server) CreateDeliveryNote(ctx context.Context, req api.CreateDeliveryNoteRequestObject) (api.CreateDeliveryNoteResponseObject, error) {
	b := req.Body
	if b == nil || !validCodes(b.BarCodes) || b.DeliveredAt.IsZero() {
		return api.CreateDeliveryNote400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "条码和发货日期不能为空")}, nil
	}
	p := auth.FromContext(ctx)
	in := store.DeliveryNoteInput{BatchID: b.BatchId, BarCodes: b.BarCodes, DeliveryBatch: optionalString(b.DeliveryBatch), LogisticsCompany: optionalString(b.LogisticsCompany), PlateNo: optionalString(b.PlateNo), Driver: optionalString(b.Driver), DriverPhone: optionalString(b.DriverPhone), DeliveredAt: b.DeliveredAt.Time, Remark: optionalString(b.Remark)}
	v, labels, err := s.store.CreateDeliveryNote(ctx, p.Scope, in, p.UserID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.CreateDeliveryNote404JSONResponse(apiErr("not_found", "批次不存在")), nil
	case errors.Is(err, store.ErrBadRef):
		return api.CreateDeliveryNote400JSONResponse{ErrorJSONResponse: apiErr("invalid_reference", docMessage(err))}, nil
	case errors.Is(err, store.ErrNotScanned):
		return api.CreateDeliveryNote409JSONResponse(apiErr("not_scanned", docMessage(err))), nil
	case errors.Is(err, store.ErrLabelDelivered):
		return api.CreateDeliveryNote409JSONResponse(apiErr("label_delivered", docMessage(err))), nil
	case err != nil:
		return nil, err
	}
	return api.CreateDeliveryNote201JSONResponse(noteDetail(v, labels)), nil
}
func (s *Server) GetDeliveryNote(ctx context.Context, req api.GetDeliveryNoteRequestObject) (api.GetDeliveryNoteResponseObject, error) {
	v, labels, err := s.store.GetDeliveryNote(ctx, auth.FromContext(ctx).Scope, req.Id)
	if errors.Is(err, store.ErrNotFound) {
		return api.GetDeliveryNote404JSONResponse(apiErr("not_found", "发货单不存在")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetDeliveryNote200JSONResponse(noteDetail(v, labels)), nil
}
func (s *Server) ListDeliverAccounts(ctx context.Context, req api.ListDeliverAccountsRequestObject) (api.ListDeliverAccountsResponseObject, error) {
	f := store.DeliverAccountFilter{CustomerID: req.Params.CustomerId, Keyword: optionalString(req.Params.Keyword)}
	if req.Params.Status != nil {
		f.Status = string(*req.Params.Status)
	}
	page, size := pageSize(req.Params.Page, req.Params.Size)
	items, total, err := s.store.ListDeliverAccounts(ctx, auth.FromContext(ctx).Scope, f, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListDeliverAccounts200JSONResponse{Items: make([]api.DeliverAccount, 0, len(items)), Total: total}
	for i := range items {
		out.Items = append(out.Items, toAPIAccount(&items[i]))
	}
	return out, nil
}
func (s *Server) CreateDeliverAccount(ctx context.Context, req api.CreateDeliverAccountRequestObject) (api.CreateDeliverAccountResponseObject, error) {
	b := req.Body
	if b == nil || len(b.DeliveryNoteIds) == 0 || b.UnitPrice < 0 {
		return api.CreateDeliverAccount400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "发货单不能为空且单价不能为负")}, nil
	}
	p := auth.FromContext(ctx)
	in := store.DeliverAccountInput{CustomerID: b.CustomerId, NoteIDs: b.DeliveryNoteIds, UnitPrice: b.UnitPrice, ExtraAmount: optionalFloat(b.ExtraAmount), ExtraRemark: optionalString(b.ExtraRemark), Remark: optionalString(b.Remark)}
	v, notes, err := s.store.CreateDeliverAccount(ctx, p.Scope, in, p.UserID)
	switch {
	case errors.Is(err, store.ErrBadRef):
		return api.CreateDeliverAccount400JSONResponse{ErrorJSONResponse: apiErr("invalid_reference", docMessage(err))}, nil
	case errors.Is(err, store.ErrNoteAccounted):
		return api.CreateDeliverAccount409JSONResponse(apiErr("note_accounted", docMessage(err))), nil
	case err != nil:
		return nil, err
	}
	return api.CreateDeliverAccount201JSONResponse(accountDetail(v, notes)), nil
}
func (s *Server) GetDeliverAccount(ctx context.Context, req api.GetDeliverAccountRequestObject) (api.GetDeliverAccountResponseObject, error) {
	v, notes, err := s.store.GetDeliverAccount(ctx, auth.FromContext(ctx).Scope, req.Id)
	if errors.Is(err, store.ErrNotFound) {
		return api.GetDeliverAccount404JSONResponse(apiErr("not_found", "对账单不存在")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetDeliverAccount200JSONResponse(accountDetail(v, notes)), nil
}
func (s *Server) ConfirmDeliverAccount(ctx context.Context, req api.ConfirmDeliverAccountRequestObject) (api.ConfirmDeliverAccountResponseObject, error) {
	p := auth.FromContext(ctx)
	v, err := s.store.ConfirmDeliverAccount(ctx, p.Scope, req.Id, p.UserID)
	if errors.Is(err, store.ErrNotFound) {
		return api.ConfirmDeliverAccount404JSONResponse(apiErr("not_found", "对账单不存在")), nil
	}
	if errors.Is(err, store.ErrConflict) {
		return api.ConfirmDeliverAccount409JSONResponse(apiErr("already_confirmed", "对账单已确认")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.ConfirmDeliverAccount200JSONResponse(toAPIAccount(v)), nil
}
