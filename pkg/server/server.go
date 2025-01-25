package server

import (
	"context"
	"errors"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/go-logr/logr"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"

	apiv1 "github.com/rickliujh/trading-chat-aggr/pkg/api/v1"
	"github.com/rickliujh/trading-chat-aggr/pkg/api/v1/apiv1connect"
	"github.com/rickliujh/trading-chat-aggr/pkg/sql"
	"github.com/rickliujh/trading-chat-aggr/pkg/tradingchat"
)

var _ apiv1connect.AggrHandler = (*Service)(nil)

func NewService(logger logr.Logger, db *sql.Queries, done <-chan struct{}) (*Service, error) {
	symbols := []string{"ETHBTC"}
	stream, err := tradingchat.BinanceStreamEventGen(
		logger.WithName("binance-stream"),
		symbols,
		func(err error) {
			logger.Error(err, "binance-stream error")
		},
		done,
	)
	if err != nil {
		panic("can't connect to biance-stream")
	}

	aggr, updateCh := tradingchat.NewAggrStream(logger.WithName("aggr"), done, stream, symbols)

	regSymbols := make(map[string]*struct{}, len(symbols))
	for _, s := range symbols {
		regSymbols[s] = nil
	}

	s := &Service{
		logger:      logger,
		db:          db,
		regSymbols:  regSymbols,
		aggr:        aggr,
		subscribers: map[string]*dispatcher{},
		rw:          &sync.RWMutex{},
	}

	updateStrm1 := make(chan string, 500)
	updateStrm2 := make(chan string, 500)
	go func() {
		defer close(updateStrm1)
		defer close(updateStrm2)
		for v := range tradingchat.OrDone(done, updateCh) {
			updateStrm1 <- v
			updateStrm2 <- v
		}
	}()
	s.push(done, updateStrm1)
	s.persist(done, updateStrm2)

	return s, nil
}

type Service struct {
	logger      logr.Logger
	db          *sql.Queries
	regSymbols  map[string]*struct{}
	aggr        tradingchat.Aggr
	subscribers map[string]*dispatcher
	rw          *sync.RWMutex
}

// Candlesticks1MStream implements apiv1connect.AggrHandler.
func (s *Service) Candlesticks1MStream(ctx context.Context, strm *connect.BidiStream[apiv1.Candlesticks1MStreamRequest, apiv1.Candlesticks1MStreamResponse]) error {
	req, err := strm.Receive()
	if err != nil {
		s.logger.Error(err, "can't read request")
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}

	id := req.GetRequestId()
	symbols := req.GetSymbols()
	if !s.isSymbolRegistered(symbols) {
		s.logger.V(1).Info("client request unregistered symbols", "symbols", symbols)
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("some of symbol are not supported"))
	}

	s.rw.Lock()
	s.subscribers[id] = &dispatcher{
		symbols: symbols,
		strm:    strm,
	}
	s.rw.Unlock()

	for {
	}
}

func (s *Service) push(done <-chan struct{}, stream <-chan string) {
	go func() {
		for {
			select {
			case <-done:
				return
			case symbol := <-stream:
				s.logger.V(4).Info("new update to push", symbol)
				s.rw.RLock()
				for _, sub := range s.subscribers {
					bar, err := s.aggr.OHLCBar(symbol)
					if err != nil {
						s.logger.Error(err, "registered symbol not exist in aggr stream", "symbol", symbol)
					}
					sub.dispatch(symbol, &bar)
				}
				s.rw.RUnlock()
			}
		}
	}()
}

func (s *Service) persist(done <-chan struct{}, stream <-chan string) {
	go func() {
		for {
			select {
			case <-done:
				return
			case symbol := <-stream:
				s.logger.V(4).Info("new update to persist", symbol)
				bar, err := s.aggr.OHLCBar(symbol)
				if err != nil {
					s.logger.Error(err, "registered symbol not exist in aggr stream", "symbol", symbol)
				}
				ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
				defer cancel()

				var h pgtype.Numeric
				if err := h.Scan(bar.H); err != nil {
					s.logger.Error(err, "failed to convert to numeric from string", "numstr", bar.H)
					continue
				}
				var l pgtype.Numeric
				if err := l.Scan(bar.L); err != nil {
					s.logger.Error(err, "failed to convert to numeric from string", "numstr", bar.L)
					continue
				}
				var o pgtype.Numeric
				if err := o.Scan(bar.O); err != nil {
					s.logger.Error(err, "failed to convert to numeric from string", "numstr", bar.O)
					continue
				}
				var c pgtype.Numeric
				if err := c.Scan(bar.C); err != nil {
					s.logger.Error(err, "failed to convert to numeric from string", "numstr", bar.C)
					continue
				}
				var ts pgtype.Timestamp
				if err := ts.Scan(time.Unix(bar.T, 0)); err != nil {
					s.logger.Error(err, "failed to convert to timestamp from int64", "ts", bar.T)
					continue
				}
				_, err = s.db.CreateBar(ctx, sql.CreateBarParams{
					H:  h,
					L:  l,
					O:  o,
					C:  c,
					Ts: ts,
				})
				if err != nil {
					s.logger.Error(err, "failed to persist to db", "bar", bar)
					continue
				}
			}
		}
	}()
}

func (s *Service) isSymbolRegistered(symbols []string) bool {
	for _, sb := range symbols {
		if _, ok := s.regSymbols[sb]; !ok {
			return false
		}
	}
	return true
}

type dispatcher struct {
	symbols []string
	strm    *connect.BidiStream[apiv1.Candlesticks1MStreamRequest, apiv1.Candlesticks1MStreamResponse]
}

func (d dispatcher) dispatch(symbol string, data *tradingchat.OHLCBar) {
	for _, s := range d.symbols {
		if s == symbol {
			d.strm.Send(&apiv1.Candlesticks1MStreamResponse{
				Update: &apiv1.Candlesticks1MStreamResponse_Bar{
					High:      data.H,
					Low:       data.L,
					Open:      data.O,
					Close:     data.C,
					UpdatedAt: timestamppb.New(time.Unix(data.T, 0)),
				},
			})
		}
	}
}
