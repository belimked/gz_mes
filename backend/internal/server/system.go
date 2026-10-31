package server

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/store"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{2,32}$`)

func adminOnly(ctx context.Context) bool { return auth.FromContext(ctx).IsAdmin }
func forbidden() api.ErrorJSONResponse   { return apiErr("forbidden", "仅管理员可操作") }
func badReference() api.ErrorJSONResponse {
	return apiErr("invalid_reference", "厂区或菜单不存在")
}
func toAPIFactory(f *store.Factory) api.Factory {
	return api.Factory{Id: f.ID, Code: f.Code, Name: f.Name, Kind: api.FactoryKind(f.Kind), IsStop: f.IsStop}
}
func toAPIUserDetail(d *store.UserDetail) api.UserDetail {
	return api.UserDetail{Id: d.ID, Username: d.Username, DisplayName: d.DisplayName, StaffNo: d.StaffNo, IsAdmin: d.IsAdmin, Enabled: d.Enabled, FactoryIds: d.FactoryIDs, MenuIds: d.MenuIDs, CreatedAt: d.CreatedAt}
}
func validPassword(p string) bool { return len(p) >= 6 && len(p) <= 72 }

func (s *Server) CreateFactory(ctx context.Context, req api.CreateFactoryRequestObject) (api.CreateFactoryResponseObject, error) {
	if !adminOnly(ctx) {
		return api.CreateFactory403JSONResponse(forbidden()), nil
	}
	b := req.Body
	code, name := strings.TrimSpace(b.Code), strings.TrimSpace(b.Name)
	if code == "" || name == "" || len(code) > 32 || len(name) > 100 || (b.Kind != "own" && b.Kind != "outsource") {
		return api.CreateFactory400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "厂区编码、名称或类型无效")}, nil
	}
	f, err := s.store.CreateFactory(ctx, code, name, string(b.Kind))
	if errors.Is(err, store.ErrConflict) {
		return api.CreateFactory409JSONResponse(apiErr("duplicate_code", "厂区编码已存在")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.CreateFactory201JSONResponse(toAPIFactory(f)), nil
}

func (s *Server) UpdateFactory(ctx context.Context, req api.UpdateFactoryRequestObject) (api.UpdateFactoryResponseObject, error) {
	if !adminOnly(ctx) {
		return api.UpdateFactory403JSONResponse(forbidden()), nil
	}
	name := strings.TrimSpace(req.Body.Name)
	if name == "" || len(name) > 100 {
		return api.UpdateFactory400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "厂区名称不能为空")}, nil
	}
	f, err := s.store.UpdateFactory(ctx, req.Id, name, req.Body.IsStop)
	if errors.Is(err, store.ErrNotFound) {
		return api.UpdateFactory404JSONResponse(apiErr("not_found", "厂区不存在")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.UpdateFactory200JSONResponse(toAPIFactory(f)), nil
}

func (s *Server) ListMenus(ctx context.Context, _ api.ListMenusRequestObject) (api.ListMenusResponseObject, error) {
	if !adminOnly(ctx) {
		return api.ListMenus403JSONResponse(forbidden()), nil
	}
	menus, err := s.store.AllMenus(ctx)
	if err != nil {
		return nil, err
	}
	return api.ListMenus200JSONResponse(toAPIMenus(menus)), nil
}

func (s *Server) ListUsers(ctx context.Context, req api.ListUsersRequestObject) (api.ListUsersResponseObject, error) {
	if !adminOnly(ctx) {
		return api.ListUsers403JSONResponse(forbidden()), nil
	}
	page, size := 1, 20
	if req.Params.Page != nil {
		page = *req.Params.Page
	}
	if req.Params.Size != nil {
		size = *req.Params.Size
	}
	if page < 1 || size < 1 || size > 200 {
		return api.ListUsers400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "分页参数无效")}, nil
	}
	f := store.UserFilter{Enabled: req.Params.Enabled}
	if req.Params.Keyword != nil {
		f.Keyword = *req.Params.Keyword
	}
	items, total, err := s.store.ListUsers(ctx, f, page, size)
	if err != nil {
		return nil, err
	}
	out := api.UserPage{Items: make([]api.UserDetail, 0, len(items)), Total: total}
	for i := range items {
		out.Items = append(out.Items, toAPIUserDetail(&items[i]))
	}
	return api.ListUsers200JSONResponse(out), nil
}

func (s *Server) CreateUser(ctx context.Context, req api.CreateUserRequestObject) (api.CreateUserResponseObject, error) {
	if !adminOnly(ctx) {
		return api.CreateUser403JSONResponse(forbidden()), nil
	}
	b := req.Body
	in := store.UserCreate{Username: strings.TrimSpace(b.Username), DisplayName: strings.TrimSpace(b.DisplayName), Password: b.Password}
	if b.StaffNo != nil {
		in.StaffNo = strings.TrimSpace(*b.StaffNo)
	}
	if b.IsAdmin != nil {
		in.IsAdmin = *b.IsAdmin
	}
	if b.FactoryIds != nil {
		in.FactoryIDs = *b.FactoryIds
	}
	if b.MenuIds != nil {
		in.MenuIDs = *b.MenuIds
	}
	if !usernamePattern.MatchString(in.Username) || in.DisplayName == "" || len(in.DisplayName) > 50 || !validPassword(in.Password) {
		return api.CreateUser400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "用户名、显示名称或密码无效")}, nil
	}
	if !in.IsAdmin && len(in.FactoryIDs) == 0 {
		return api.CreateUser400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "非管理员用户至少需要一个厂区")}, nil
	}
	d, err := s.store.CreateUserFull(ctx, in)
	switch {
	case errors.Is(err, store.ErrBadRef):
		return api.CreateUser400JSONResponse{ErrorJSONResponse: badReference()}, nil
	case errors.Is(err, store.ErrConflict):
		return api.CreateUser409JSONResponse(apiErr("duplicate_code", "用户名已存在")), nil
	case err != nil:
		return nil, err
	}
	return api.CreateUser201JSONResponse(toAPIUserDetail(d)), nil
}

func (s *Server) GetUser(ctx context.Context, req api.GetUserRequestObject) (api.GetUserResponseObject, error) {
	if !adminOnly(ctx) {
		return api.GetUser403JSONResponse(forbidden()), nil
	}
	d, err := s.store.GetUser(ctx, req.Id)
	if errors.Is(err, store.ErrNotFound) {
		return api.GetUser404JSONResponse(apiErr("not_found", "用户不存在")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetUser200JSONResponse(toAPIUserDetail(d)), nil
}

func (s *Server) UpdateUser(ctx context.Context, req api.UpdateUserRequestObject) (api.UpdateUserResponseObject, error) {
	if !adminOnly(ctx) {
		return api.UpdateUser403JSONResponse(forbidden()), nil
	}
	b := req.Body
	if req.Id == auth.FromContext(ctx).UserID && (!b.Enabled || !b.IsAdmin) {
		return api.UpdateUser409JSONResponse(apiErr("self_lock", "不能停用自己或取消自己的管理员权限")), nil
	}
	if !b.IsAdmin && len(b.FactoryIds) == 0 {
		return api.UpdateUser400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "非管理员用户至少需要一个厂区")}, nil
	}
	name := strings.TrimSpace(b.DisplayName)
	if name == "" || len(name) > 50 {
		return api.UpdateUser400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "显示名称不能为空")}, nil
	}
	staffNo := ""
	if b.StaffNo != nil {
		staffNo = strings.TrimSpace(*b.StaffNo)
	}
	d, err := s.store.UpdateUser(ctx, req.Id, store.UserUpdate{DisplayName: name, StaffNo: staffNo, Enabled: b.Enabled, IsAdmin: b.IsAdmin, FactoryIDs: b.FactoryIds, MenuIDs: b.MenuIds})
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.UpdateUser404JSONResponse(apiErr("not_found", "用户不存在")), nil
	case errors.Is(err, store.ErrBadRef):
		return api.UpdateUser400JSONResponse{ErrorJSONResponse: badReference()}, nil
	case err != nil:
		return nil, err
	}
	return api.UpdateUser200JSONResponse(toAPIUserDetail(d)), nil
}

func (s *Server) ResetUserPassword(ctx context.Context, req api.ResetUserPasswordRequestObject) (api.ResetUserPasswordResponseObject, error) {
	if !adminOnly(ctx) {
		return api.ResetUserPassword403JSONResponse(forbidden()), nil
	}
	if !validPassword(req.Body.Password) {
		return api.ResetUserPassword400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "密码长度必须为6到72位")}, nil
	}
	err := s.store.ResetPassword(ctx, req.Id, req.Body.Password)
	if errors.Is(err, store.ErrNotFound) {
		return api.ResetUserPassword404JSONResponse(apiErr("not_found", "用户不存在")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.ResetUserPassword204Response{}, nil
}
