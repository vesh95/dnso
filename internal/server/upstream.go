package server

import (
	"context"
	"log/slog"

	"github.com/miekg/dns"
)

type UpstreamExchanger struct {
	client    *dns.Client
	addresses []string
	logger    *slog.Logger
}

func NewExchanger(addresses []string, logger *slog.Logger) *UpstreamExchanger {
	client := new(dns.Client)
	return &UpstreamExchanger{
		client:    client,
		addresses: addresses,
		logger:    logger,
	}
}

func (e *UpstreamExchanger) Exchange(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
	answer := make(chan *dns.Msg, len(e.addresses))
	exchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, addr := range e.addresses {
		go func(addr string) {
			e.logger.DebugContext(ctx, "start upstream exchange", "address", addr, "id", req.Id)
			response, _, err := e.client.ExchangeContext(exchCtx, req, addr)
			if err != nil {
				e.logger.ErrorContext(ctx, "upstream exchange error", "address", addr, "id", req.Id, "error", err.Error())
				return
			}
			select {
			case answer <- response:
			case <-exchCtx.Done():
				e.logger.DebugContext(ctx, "upstream exchange canceled", "address", addr, "id", req.Id)
			}

			e.logger.DebugContext(ctx, "upstream exchange completed", "address", addr, "id", req.Id)
		}(addr)
	}

	select {
	case response := <-answer:
		cancel()
		return response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
