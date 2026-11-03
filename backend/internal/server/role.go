package server

import (
	"context"
	"errors"
	"strings"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/store"
)

func apiRole(r *store.Role) api.Role {
	return api.Role{Id: r.ID, Code: r.Code, Name: r.Name, Remark: r.Remark, IsStop: r.IsStop, MenuIds: r.MenuIDs, UserCount: r.UserCount, CreatedAt: r.CreatedAt}
}
func roleInput(b *api.RoleInput) store.RoleInput {
	return store.RoleInput{Code: strings.TrimSpace(b.Code), Name: strings.TrimSpace(b.Name), Remark: optionalString(b.Remark), IsStop: b.IsStop, MenuIDs: b.MenuIds}
}
func validRole(in store.RoleInput) bool {
	return in.Code != "" && len([]rune(in.Code)) <= 50 && !strings.HasPrefix(in.Code, "user:") && in.Name != "" && len([]rune(in.Name)) <= 50
}
func (s *Server) ListRoles(ctx context.Context, _ api.ListRolesRequestObject) (api.ListRolesResponseObject, error) {
	if !adminOnly(ctx) {
		return api.ListRoles403JSONResponse(forbidden()), nil
	}
	rs, err := s.store.Roles(ctx)
	if err != nil {
		return nil, err
	}
	out := make(api.ListRoles200JSONResponse, 0, len(rs))
	for i := range rs {
		out = append(out, apiRole(&rs[i]))
	}
	return out, nil
}
func (s *Server) CreateRole(ctx context.Context, req api.CreateRoleRequestObject) (api.CreateRoleResponseObject, error) {
	if !adminOnly(ctx) {
		return api.CreateRole403JSONResponse(forbidden()), nil
	}
	in := roleInput(req.Body)
	if !validRole(in) {
		return api.CreateRole400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "角色编码或名称无效")}, nil
	}
	r, err := s.store.SaveRole(ctx, nil, in)
	switch {
	case errors.Is(err, store.ErrBadRef):
		return api.CreateRole400JSONResponse{ErrorJSONResponse: apiErr("invalid_reference", "菜单不存在")}, nil
	case errors.Is(err, store.ErrConflict):
		return api.CreateRole409JSONResponse(apiErr("duplicate_code", "角色编码已存在")), nil
	case err != nil:
		return nil, err
	}
	return api.CreateRole201JSONResponse(apiRole(r)), nil
}
func (s *Server) UpdateRole(ctx context.Context, req api.UpdateRoleRequestObject) (api.UpdateRoleResponseObject, error) {
	if !adminOnly(ctx) {
		return api.UpdateRole403JSONResponse(forbidden()), nil
	}
	in := roleInput(req.Body)
	if !validRole(in) {
		return api.UpdateRole400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "角色编码或名称无效")}, nil
	}
	r, err := s.store.SaveRole(ctx, &req.Id, in)
	switch {
	case errors.Is(err, store.ErrBadRef):
		return api.UpdateRole400JSONResponse{ErrorJSONResponse: apiErr("invalid_reference", "菜单不存在")}, nil
	case errors.Is(err, store.ErrNotFound):
		return api.UpdateRole404JSONResponse(apiErr("not_found", "角色不存在")), nil
	case errors.Is(err, store.ErrConflict):
		return api.UpdateRole409JSONResponse(apiErr("duplicate_code", "角色编码已存在")), nil
	case err != nil:
		return nil, err
	}
	return api.UpdateRole200JSONResponse(apiRole(r)), nil
}
func (s *Server) DeleteRole(ctx context.Context, req api.DeleteRoleRequestObject) (api.DeleteRoleResponseObject, error) {
	if !adminOnly(ctx) {
		return api.DeleteRole403JSONResponse(forbidden()), nil
	}
	err := s.store.DeleteRole(ctx, req.Id)
	var occupied *store.RoleOccupiedError
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.DeleteRole404JSONResponse(apiErr("not_found", "角色不存在")), nil
	case errors.As(err, &occupied):
		return api.DeleteRole409JSONResponse(apiErr("role_occupied", occupied.Error())), nil
	case err != nil:
		return nil, err
	}
	return api.DeleteRole204Response{}, nil
}
