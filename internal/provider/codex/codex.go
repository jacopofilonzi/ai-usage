// Package codex reads Codex's usage limits from its local session logs.
//
// CONTRACT STUB (phase 0): implemented by the providers agent.
package codex

import (
	"context"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

type Provider struct{}

func New() *Provider { return &Provider{} }

func (p *Provider) ID() string              { return "codex" }
func (p *Provider) Name() string            { return "Codex" }
func (p *Provider) Detect() bool            { return false }
func (p *Provider) Interval() time.Duration { return 30 * time.Second }

func (p *Provider) Poll(ctx context.Context) ([]provider.Limit, error) { return nil, nil }

var _ provider.Provider = (*Provider)(nil)
