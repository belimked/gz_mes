// Package server wires the generated OpenAPI handlers to the store.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	oapimw "github.com/oapi-codegen/nethttp-middleware"

	"github.com/belimked/gz_mes/backend/internal/api"
	"github.com/belimked/gz_mes/backend/internal/auth"
	"github.com/belimked/gz_mes/backend/internal/storage"
	"github.com/belimked/gz_mes/backend/internal/store"
)

// Server implements api.StrictServerInterface.
type Server struct {
	store     *store.Store
	tokens    *auth.Tokens
	version   string
	files     storage.Files
	maxUpload int64
}

var _ api.StrictServerInterface = (*Server)(nil)

func NewWithFiles(st *store.Store, tokens *auth.Tokens, version string, files storage.Files, maxUpload int64) *Server {
	return &Server{store: st, tokens: tokens, version: version, files: files, maxUpload: maxUpload}
}

// Handler builds the full HTTP handler mounted under /api.
func (s *Server) Handler() (http.Handler, error) {
	spec, err := api.GetSpec()
	if err != nil {
		return nil, err
	}
	spec.Servers = nil // match paths without the /api prefix after mounting

	validator := oapimw.OapiRequestValidatorWithOptions(spec, &oapimw.Options{
		Options: openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
		ErrorHandler: func(w http.ResponseWriter, msg string, status int) {
			writeErr(w, status, "invalid_request", msg)
		},
	})

	strict := api.NewStrictHandlerWithOptions(s, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeErr(w, http.StatusBadRequest, "invalid_request", err.Error())
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.ErrorContext(r.Context(), "handler error", "err", err, "path", r.URL.Path)
			writeErr(w, http.StatusInternalServerError, "internal", "internal server error")
		},
	})

	inner := chi.NewRouter()
	// Multipart validation consumes the stream before MaxBytesReader can enforce our
	// upload limit. Binary responses also cannot be serialized by the strict handler.
	// These two routes use raw HTTP handlers after the same authentication middleware.
	inner.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if (r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/batches/") && strings.HasSuffix(r.URL.Path, "/drawings")) || (r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/drawings/") && strings.HasSuffix(r.URL.Path, "/file")) {
				next.ServeHTTP(w, r)
				return
			}
			validator(next).ServeHTTP(w, r)
		})
	}, s.authenticate)
	api.HandlerFromMux(strict, inner)

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, middleware.Timeout(60*time.Second))
	// Strip the prefix so the validator, auth check and generated routes all see
	// spec-relative paths such as /health.
	r.Handle("/api/*", http.StripPrefix("/api", inner))
	return r, nil
}

// publicOps are reachable without a token (method + path relative to /api).
var publicOps = map[string]bool{
	"GET /health":      true,
	"POST /auth/login": true,
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if publicOps[r.Method+" "+r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}
		tok, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || tok == "" {
			writeErr(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
			return
		}
		uid, err := s.tokens.Parse(tok)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
			return
		}
		p, err := s.store.Principal(r.Context(), uid)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusUnauthorized, "unauthorized", "user disabled or removed")
			return
		}
		if err != nil {
			slog.ErrorContext(r.Context(), "load principal", "err", err)
			writeErr(w, http.StatusInternalServerError, "internal", "internal server error")
			return
		}
		r = r.WithContext(auth.WithPrincipal(r.Context(), p))
		if r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/batches/") && strings.HasSuffix(r.URL.Path, "/drawings") {
			s.rawUpload(w, r)
			return
		}
		if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/drawings/") && strings.HasSuffix(r.URL.Path, "/file") {
			s.rawFile(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.Error{Code: code, Message: msg})
}

func apiErr(code, msg string) api.ErrorJSONResponse {
	return api.ErrorJSONResponse{Code: code, Message: msg}
}

// ---- system ----

// GetHealth implements GET /health.
func (s *Server) GetHealth(context.Context, api.GetHealthRequestObject) (api.GetHealthResponseObject, error) {
	return api.GetHealth200JSONResponse{Status: "ok", Version: s.version}, nil
}

// ListFactories implements GET /factories.
func (s *Server) ListFactories(ctx context.Context, _ api.ListFactoriesRequestObject) (api.ListFactoriesResponseObject, error) {
	p := auth.FromContext(ctx)
	fs, err := s.store.Factories(ctx, p.Scope)
	if err != nil {
		return nil, err
	}
	out := make(api.ListFactories200JSONResponse, 0, len(fs))
	for _, f := range fs {
		out = append(out, toAPIFactory(&f))
	}
	return out, nil
}

// ---- auth ----

// Login implements POST /auth/login.
func (s *Server) Login(ctx context.Context, req api.LoginRequestObject) (api.LoginResponseObject, error) {
	bad := api.Login401JSONResponse{ErrorJSONResponse: apiErr("invalid_credentials", "用户名或密码错误")}
	u, err := s.store.UserByUsername(ctx, req.Body.Username)
	if errors.Is(err, store.ErrNotFound) {
		return bad, nil
	}
	if err != nil {
		return nil, err
	}
	if !u.Enabled || !auth.CheckPassword(u.PasswordHash, req.Body.Password) {
		return bad, nil
	}
	tok, err := s.tokens.Issue(u.ID)
	if err != nil {
		return nil, err
	}
	return api.Login200JSONResponse{Token: tok, User: toAPIUser(u.ID, u.Username, u.DisplayName, u.StaffNo, u.IsAdmin)}, nil
}

// GetMe implements GET /auth/me.
func (s *Server) GetMe(ctx context.Context, _ api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	p := auth.FromContext(ctx)
	menus, err := s.store.MenusFor(ctx, p)
	if err != nil {
		return nil, err
	}
	return api.GetMe200JSONResponse{
		User:  toAPIUser(p.UserID, p.Username, p.DisplayName, p.StaffNo, p.IsAdmin),
		Menus: toAPIMenus(menus),
	}, nil
}

func toAPIUser(id int64, username, name, staffNo string, admin bool) api.User {
	return api.User{Id: id, Username: username, DisplayName: name, StaffNo: staffNo, IsAdmin: admin}
}

func toAPIMenus(ms []*store.Menu) []api.MenuItem {
	out := make([]api.MenuItem, 0, len(ms))
	for _, m := range ms {
		item := api.MenuItem{Id: m.ID, Title: m.Title, Path: m.Path}
		if m.Icon != "" {
			icon := m.Icon
			item.Icon = &icon
		}
		if len(m.Children) > 0 {
			children := toAPIMenus(m.Children)
			item.Children = &children
		}
		out = append(out, item)
	}
	return out
}

// ---- sale: projects ----

// ListProjects implements GET /projects.
func (s *Server) ListProjects(ctx context.Context, req api.ListProjectsRequestObject) (api.ListProjectsResponseObject, error) {
	p := auth.FromContext(ctx)
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
	items, total, err := s.store.ListProjects(ctx, p.Scope, kw, page, size)
	if err != nil {
		return nil, err
	}
	out := api.ListProjects200JSONResponse{Items: make([]api.Project, 0, len(items)), Total: total}
	for i := range items {
		out.Items = append(out.Items, toAPIProject(&items[i]))
	}
	return out, nil
}

// CreateProject implements POST /projects.
func (s *Server) CreateProject(ctx context.Context, req api.CreateProjectRequestObject) (api.CreateProjectResponseObject, error) {
	p := auth.FromContext(ctx)
	b := req.Body
	if !p.Scope.Allows(b.FactoryId) {
		return api.CreateProject403JSONResponse(apiErr("forbidden_factory", "无权在该厂区创建数据")), nil
	}
	in := store.ProjectCreate{
		FactoryID: b.FactoryId,
		Code:      strings.TrimSpace(b.Code),
		FullName:  strings.TrimSpace(b.FullName),
		Kind:      string(api.Offline),
		CreatedBy: p.UserID,
	}
	if b.ShortName != nil {
		in.ShortName = strings.TrimSpace(*b.ShortName)
	}
	if b.Address != nil {
		in.Address = strings.TrimSpace(*b.Address)
	}
	if b.Kind != nil {
		in.Kind = string(*b.Kind)
	}
	if in.Code == "" || in.FullName == "" {
		return api.CreateProject400JSONResponse{ErrorJSONResponse: apiErr("invalid_request", "项目编号和名称不能为空")}, nil
	}
	proj, err := s.store.CreateProject(ctx, in)
	if errors.Is(err, store.ErrConflict) {
		return api.CreateProject409JSONResponse(apiErr("duplicate_code", "项目编号已存在")), nil
	}
	if errors.Is(err, store.ErrBadRef) {
		return api.CreateProject400JSONResponse{ErrorJSONResponse: apiErr("invalid_factory", "厂区不存在")}, nil
	}
	if err != nil {
		return nil, err
	}
	return api.CreateProject201JSONResponse(toAPIProject(proj)), nil
}

// ConfirmProject implements POST /projects/{id}/confirm.
func (s *Server) ConfirmProject(ctx context.Context, req api.ConfirmProjectRequestObject) (api.ConfirmProjectResponseObject, error) {
	p := auth.FromContext(ctx)
	proj, err := s.store.ConfirmProject(ctx, p.Scope, req.Id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return api.ConfirmProject404JSONResponse(apiErr("not_found", "项目不存在")), nil
	case errors.Is(err, store.ErrConflict):
		return api.ConfirmProject409JSONResponse(apiErr("already_confirmed", "项目已确认")), nil
	case err != nil:
		return nil, err
	}
	return api.ConfirmProject200JSONResponse(toAPIProject(proj)), nil
}

func toAPIProject(p *store.Project) api.Project {
	return api.Project{
		Id: p.ID, FactoryId: p.FactoryID, Code: p.Code, FullName: p.FullName, ShortName: p.ShortName,
		Address: p.Address, Kind: api.ProjectKind(p.Kind), Confirmed: p.Confirmed,
		CreatedBy: p.CreatedBy, CreatedAt: p.CreatedAt,
	}
}
