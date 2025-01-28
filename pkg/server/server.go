package server

import (
	"context"
	"errors"
	"io"
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
	"github.com/rickliujh/trading-chat-aggr/pkg/utils"
)

var (
	_ apiv1connect.AggrHandler = (*Service)(nil)

	ErrNotRevieved         = connect.NewError(connect.CodeUnknown, errors.New("unable to receive"))
	ErrInvalidRequest      = connect.NewError(connect.CodeInvalidArgument, errors.New("invalid request id or symbols"))
	ErrSymbolsNotSupported = connect.NewError(connect.CodeInvalidArgument, errors.New("some of symbols are not supported"))
)

func NewService(logger logr.Logger, db *sql.Queries, symbols []string, done <-chan struct{}, push, persist bool) (*Service, error) {
	logger.Info("registering symbols", "symbols", symbols)
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

	regSymbols := make(map[string]bool, len(symbols))
	for _, s := range symbols {
		regSymbols[s] = true
	}

	s := &Service{
		logger:      logger,
		db:          db,
		regSymbols:  regSymbols,
		aggr:        aggr,
		notifyList:  map[string][]*connect.BidiStream[apiv1.Candlesticks1MStreamRequest, apiv1.Candlesticks1MStreamResponse]{},
		rw:          &sync.RWMutex{},
		oncePush:    &sync.Once{},
		oncePersist: &sync.Once{},
	}

	logger.Info("function enables", "push", push, "persist", persist)
	if push && persist {
		updateStrm1 := make(chan string, 500)
		updateStrm2 := make(chan string, 500)
		go func() {
			defer close(updateStrm1)
			defer close(updateStrm2)
			for v := range utils.OrDone(done, updateCh) {
				updateStrm1 <- v
				updateStrm2 <- v
			}
		}()
		s.push(done, updateStrm1)
		s.persist(done, updateStrm2)
	} else if push {
		s.push(done, updateCh)
	} else if persist {
		s.persist(done, updateCh)
	}

	return s, nil
}

type Service struct {
	logger      logr.Logger
	db          *sql.Queries
	regSymbols  map[string]bool
	aggr        tradingchat.Aggr
	notifyList  map[string][]*connect.BidiStream[apiv1.Candlesticks1MStreamRequest, apiv1.Candlesticks1MStreamResponse]
	rw          *sync.RWMutex
	oncePush    *sync.Once
	oncePersist *sync.Once
}

// Candlesticks1MStream implements apiv1connect.AggrHandler.
func (s *Service) Candlesticks1MStream(ctx context.Context, strm *connect.BidiStream[apiv1.Candlesticks1MStreamRequest, apiv1.Candlesticks1MStreamResponse]) error {
	var id string
	var symbols []string
	defer func() {
		s.logger.V(2).Info("removing subscriber", "req_id", id, "symbols", symbols)
		s.removeFromList(symbols, strm)
		s.logger.V(2).Info("client disconnected", "req_id", id, "symbols", symbols)
	}()
	for {
		req, err := strm.Receive()
		if err != nil {
			if errors.Is(err, io.EOF) {
				s.logger.Info("user disconnected", "req_id", id, "symbols", symbols)
				return nil
			}
			s.logger.Error(err, "error when reading request message")
			return ErrNotRevieved
		}

		reqID := req.GetRequestId()
		reqSbs := req.GetSymbols()

		// validation
		isReqValid := reqID != "" && len(reqSbs) != 0 && (reqID == id || id == "")
		if !isReqValid {
			s.logger.Info("found invalid request disconnecting with client", "req_id", id, "req_id_new", reqID)
			return ErrInvalidRequest
		}
		if !s.isSymbolRegistered(reqSbs) {
			s.logger.Info("client request unregistered symbols", "symbols", reqSbs)
			return ErrSymbolsNotSupported
		}

		// track request id and subscribed symbols
		id = reqID
		var toBeAdd []string
		for _, sNew := range reqSbs {
			exist := false
			for _, sOld := range symbols {
				if sNew == sOld {
					exist = true
					break
				}
			}
			if !exist {
				toBeAdd = append(toBeAdd, sNew)
			}
		}
		symbols = append(symbols, toBeAdd...)

		s.addToList(toBeAdd, strm)
		s.logger.Info("user registered for OHLC 1m stream updates", "req_id", id, "symbols", symbols, "symbols-added", toBeAdd)
	}
}

// removeFromList removes elements to be deleted by swapping it with last element (order don't matter)
// and slices arr with (length of arr before swapping - number of elements that removed)
// to achieve most efficiency of deletion
func (s *Service) removeFromList(symbols []string, strm *connect.BidiStream[apiv1.Candlesticks1MStreamRequest, apiv1.Candlesticks1MStreamResponse]) {
	s.rw.Lock()
	for _, symbol := range symbols {
		sublist, _ := s.notifyList[symbol]

		// next swap location pointer
		count := len(sublist) - 1
		for i, to := range sublist {
			// compare the address of the stream
			if to == strm {
				sublist[i] = sublist[count]
				count--
			}
		}
		s.notifyList[symbol] = sublist[:count+1]
	}
	s.rw.Unlock()
}

func (s *Service) addToList(symbols []string, to *connect.BidiStream[apiv1.Candlesticks1MStreamRequest, apiv1.Candlesticks1MStreamResponse]) {
	s.rw.Lock()
	for _, symbol := range symbols {
		sublist, _ := s.notifyList[symbol]
		sublist = append(sublist, to)
		s.notifyList[symbol] = sublist
	}
	s.rw.Unlock()
}
func (s *Service) push(done <-chan struct{}, updateStream <-chan string) {
	s.oncePush.Do(func() {
		go func() {
			for symbol := range utils.OrDone(done, updateStream) {
				s.logger.V(4).Info("new update to push", symbol)
				s.rw.RLock()
				sublist := s.notifyList[symbol]

				bar, err := s.aggr.OHLCBar(symbol)
				if err != nil {
					s.logger.Error(err, "registered symbol not exist in aggr stream", "symbol", symbol)
				}

				for _, to := range sublist {
					to.Send(&apiv1.Candlesticks1MStreamResponse{
						Update: &apiv1.Candlesticks1MStreamResponse_Bar{
							High:      bar.H,
							Low:       bar.L,
							Open:      bar.O,
							Close:     bar.C,
							UpdatedAt: timestamppb.New(time.Unix(bar.T, 0)),
						},
					})
				}
				s.rw.RUnlock()
			}
		}()
	})
}

func (s *Service) persist(done <-chan struct{}, updateStream <-chan string) {
	s.oncePersist.Do(func() {
		go func() {
			for symbol := range utils.OrDone(done, updateStream) {
				s.logger.V(4).Info("new update to persist", symbol)

				bar, err := s.aggr.OHLCBar(symbol)
				if err != nil {
					s.logger.Error(err, "registered symbol not exist in aggr stream", "symbol", symbol)
				}

				ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
				defer cancel()

				bar4db, err := toDBBar(bar)
				if err != nil {
					s.logger.Error(err, "failed to conver OHLCBar to db model", "bar", bar)
					continue
				}
				_, err = s.db.CreateBar(ctx, bar4db)
				if err != nil {
					s.logger.Error(err, "failed to persist to db", "bar", bar)
					continue
				}
			}
		}()
	})
}

func (s *Service) isSymbolRegistered(symbols []string) bool {
	for _, sb := range symbols {
		if _, ok := s.regSymbols[sb]; !ok {
			return false
		}
	}
	return true
}

func toDBBar(bar tradingchat.OHLCBar) (sql.CreateBarParams, error) {
	var h pgtype.Numeric
	if err := h.Scan(bar.H); err != nil {
		return sql.CreateBarParams{}, err
	}
	var l pgtype.Numeric
	if err := l.Scan(bar.L); err != nil {
		return sql.CreateBarParams{}, err
	}
	var o pgtype.Numeric
	if err := o.Scan(bar.O); err != nil {
		return sql.CreateBarParams{}, err
	}
	var c pgtype.Numeric
	if err := c.Scan(bar.C); err != nil {
		return sql.CreateBarParams{}, err
	}
	var ts pgtype.Timestamp
	if err := ts.Scan(time.Unix(bar.T, 0)); err != nil {
		return sql.CreateBarParams{}, err
	}
	return sql.CreateBarParams{
		H:  h,
		L:  l,
		O:  o,
		C:  c,
		Ts: ts,
	}, nil
}
