package server

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/store"
)

func apiDict(d store.DictItem) api.DictItem {
	return api.DictItem{Id: d.ID, Kind: api.DictItemKind(d.Kind), Name: d.Name, Sort: d.Sort, StartTime: d.StartTime, EndTime: d.EndTime, IsStop: d.IsStop, Remark: &d.Remark, CreatedAt: d.CreatedAt}
}
func dictInput(b *api.DictInput) store.DictInput {
	return store.DictInput{Kind: string(b.Kind), Name: strings.TrimSpace(b.Name), Sort: b.Sort, StartTime: b.StartTime, EndTime: b.EndTime, IsStop: b.IsStop, Remark: optionalString(b.Remark)}
}
func validDict(in store.DictInput) bool {
	if in.Kind != "workshop" && in.Kind != "paint_line" && in.Kind != "shift" || len([]rune(in.Name)) == 0 || len([]rune(in.Name)) > 50 {
		return false
	}
	if (in.StartTime == nil) != (in.EndTime == nil) || in.Kind != "shift" && in.StartTime != nil {
		return false
	}
	for _, v := range []*string{in.StartTime, in.EndTime} {
		if v != nil {
			t, e := time.Parse("15:04", *v)
			if e != nil || t.Format("15:04") != *v {
				return false
			}
		}
	}
	return true
}
func (s *Server) ListDicts(ctx context.Context, req api.ListDictsRequestObject) (api.ListDictsResponseObject, error) {
	kind := ""
	if req.Params.Kind != nil {
		kind = string(*req.Params.Kind)
	}
	items, err := s.store.Dicts(ctx, kind, req.Params.IncludeStopped != nil && *req.Params.IncludeStopped)
	if err != nil {
		return nil, err
	}
	out := make(api.ListDicts200JSONResponse, 0, len(items))
	for _, v := range items {
		out = append(out, apiDict(v))
	}
	return out, nil
}
func (s *Server) CreateDict(ctx context.Context, req api.CreateDictRequestObject) (api.CreateDictResponseObject, error) {
	if !auth.FromContext(ctx).Scope.All {
		return api.CreateDict403JSONResponse(apiErr("forbidden", "无权维护基础字典")), nil
	}
	in := dictInput(req.Body)
	if !validDict(in) {
		return api.CreateDict400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "字典名称、类型或时间无效")}, nil
	}
	v, err := s.store.SaveDict(ctx, nil, in)
	if errors.Is(err, store.ErrConflict) {
		return api.CreateDict409JSONResponse(apiErr("duplicate_name", "名称已存在")), nil
	}
	if err != nil {
		return nil, err
	}
	return api.CreateDict201JSONResponse(apiDict(*v)), nil
}
func (s *Server) UpdateDict(ctx context.Context, req api.UpdateDictRequestObject) (api.UpdateDictResponseObject, error) {
	if !auth.FromContext(ctx).Scope.All {
		return api.UpdateDict403JSONResponse(apiErr("forbidden", "无权维护基础字典")), nil
	}
	in := dictInput(req.Body)
	if !validDict(in) {
		return api.UpdateDict400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "字典名称、类型或时间无效")}, nil
	}
	v, err := s.store.SaveDict(ctx, &req.Id, in)
	switch {
	case errors.Is(err, store.ErrBadRef):
		return api.UpdateDict400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "字典类型不能修改")}, nil
	case errors.Is(err, store.ErrNotFound):
		return api.UpdateDict404JSONResponse(apiErr("not_found", "字典项不存在")), nil
	case errors.Is(err, store.ErrConflict):
		return api.UpdateDict409JSONResponse(apiErr("duplicate_name", "名称已存在")), nil
	case err != nil:
		return nil, err
	}
	return api.UpdateDict200JSONResponse(apiDict(*v)), nil
}
