package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rmpalgo/tcgapi-mcp/internal/analysis"
	"github.com/rmpalgo/tcgapi-mcp/internal/buildinfo"
	"github.com/rmpalgo/tcgapi-mcp/internal/catalog"
	"github.com/rmpalgo/tcgapi-mcp/internal/tcgapi"
)

type Dependencies struct {
	Logger   *slog.Logger
	API      tcgapi.API
	Analyzer analysis.Service
	Resolver catalog.Resolver
	PageSize int
	Build    buildinfo.Info
}

type Server struct {
	logger            *slog.Logger
	api               tcgapi.API
	analyzer          analysis.Service
	resolver          catalog.Resolver
	pageSize          int
	raw               *mcp.Server
	tools             []*mcp.Tool
	resources         []*mcp.Resource
	resourceTemplates []*mcp.ResourceTemplate
	prompts           []*mcp.Prompt
}

func New(d Dependencies) (*Server, error) {
	if d.Logger == nil {
		return nil, fmt.Errorf("server logger is required")
	}
	if d.API == nil {
		return nil, fmt.Errorf("server API dependency is required")
	}
	if d.Analyzer == nil {
		return nil, fmt.Errorf("server analyzer dependency is required")
	}
	if d.Resolver == nil {
		return nil, fmt.Errorf("server resolver dependency is required")
	}
	if d.PageSize <= 0 {
		return nil, fmt.Errorf("page size must be > 0")
	}

	s := &Server{
		logger:   d.Logger,
		api:      d.API,
		analyzer: d.Analyzer,
		resolver: d.Resolver,
		pageSize: d.PageSize,
		raw: mcp.NewServer(&mcp.Implementation{
			Name:    "tcgapi-mcp",
			Version: implementationVersion(d.Build),
		}, nil),
	}

	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	return s, nil
}

func (s *Server) ServeStdio(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	if stdin == nil {
		return fmt.Errorf("stdin reader is required")
	}
	if stdout == nil {
		return fmt.Errorf("stdout writer is required")
	}

	return s.raw.Run(ctx, &mcp.IOTransport{
		Reader: io.NopCloser(stdin),
		Writer: nopWriteCloser{Writer: stdout},
	})
}

func (s *Server) Tools() []*mcp.Tool {
	return append([]*mcp.Tool(nil), s.tools...)
}

func (s *Server) Resources() []*mcp.Resource {
	return append([]*mcp.Resource(nil), s.resources...)
}

func (s *Server) ResourceTemplates() []*mcp.ResourceTemplate {
	return append([]*mcp.ResourceTemplate(nil), s.resourceTemplates...)
}

func (s *Server) Prompts() []*mcp.Prompt {
	return append([]*mcp.Prompt(nil), s.prompts...)
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}

func implementationVersion(info buildinfo.Info) string {
	if info.Version != "" {
		return info.Version
	}
	return "dev"
}
