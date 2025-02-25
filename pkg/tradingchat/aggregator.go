package tradingchat

import (
	"errors"
	"time"

	bconn "github.com/binance/binance-connector-go"
	"github.com/go-logr/logr"
	"github.com/rickliujh/trading-chat-aggr/pkg/utils"
)

var (
	ErrNotHanlderFound     = errors.New("event of a undeclared symbol found")
	ErrNotSymbolRegistered = errors.New("symbol are not registered")
)

type Aggr map[string]*OHLCCalc

func NewAggrStream(logger logr.Logger, done <-chan struct{}, eventStream <-chan *bconn.WsAggTradeEvent, symbols []string) (Aggr, <-chan string) {
	dict := Aggr{}
	updateCh := make(chan string, 500)
	for _, symbol := range symbols {
		dict[symbol] = NewOHLCCalc(logger.WithName(symbol))
	}

	go func() {
		defer close(updateCh)
		for e := range utils.OrDone(done, eventStream) {
			logger.V(4).Info("aggregator received new event", "event", e)

			calc, ok := dict[e.Symbol]
			if !ok {
				logger.V(2).Error(ErrNotHanlderFound, "unsupported symbol", "symbol", e.Symbol, "event", e)
				continue
			}

			calc.update(e)
			updateCh <- e.Symbol
		}
	}()

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for tick := range utils.OrDone(done, ticker.C) {
			for _, calc := range dict {
				calc.tick(tick.Unix())
			}
		}
	}()

	return dict, updateCh
}

func (ag Aggr) OHLCBar(symbol string) (OHLCBar, error) {
	calc, ok := ag[symbol]
	if !ok {
		return OHLCBar{}, ErrNotSymbolRegistered
	}
	return calc.Bar(), nil
}

