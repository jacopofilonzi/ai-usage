// Package claude reads Claude Code's usage limits.
//
// CONTRACT STUB (phase 0): implemented by the providers agent.
package claude

import (
	"context"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

type Provider struct{}

func New() *Provider { return &Provider{} }

func (p *Provider) ID() string              { return "claude" }
func (p *Provider) Name() string            { return "Claude Code" }
func (p *Provider) Detect() bool            { return false }
func (p *Provider) Interval() time.Duration { return 2 * time.Minute }

func (p *Provider) Poll(ctx context.Context) ([]provider.Limit, error) { return nil, nil }

var _ provider.Provider = (*Provider)(nil)
