package tradingchat

import (
	"testing"

	bconn "github.com/binance/binance-connector-go"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
)

func TestAggr(t *testing.T) {
	logger := testr.New(t)
	logger.Info("start")

	t.Run("events update should be notified to downstream", func(t *testing.T) {
		events := []*bconn.WsAggTradeEvent{
			{
				Symbol:    "BNBBTC",
				Price:     "0.11111",
				TradeTime: 1737734701,
			},
			{
				Symbol:    "ETHBTC",
				Price:     "0.11121",
				TradeTime: 1737734711,
			},
		}
		symbols := []string{}
		for _, e := range events {
			symbols = append(symbols, e.Symbol)
		}

		done := make(chan struct{})
		stream := make(chan *bconn.WsAggTradeEvent)
		defer close(stream)
		ag, updateCh := NewAggrStream(&logger, done, stream, symbols)

		go func() {
			for _, e := range events {
				stream <- e
			}
		}()

		res := []string{}
		go func() {
			for s := range updateCh {
				logger.Info("updateCh", s)
				res = append(res, s)
				if len(res) == len(symbols) {
					close(done)
				}
			}
		}()

		<-done

		assert.Equal(t, symbols, res, "symbols of events should be notified in updateCh")

		bar, err := ag.OHLCBar("ETHBTC")
		assert.NoError(t, err, "should not throw error for existing symbol")
		assert.Equal(t,
			OHLCBar{
				H: "0.11121",
				L: "0.11121",
				O: "0.11121",
				C: "0.11121",
				T: 1737734711,
			},
			bar,
		)

		_, err = ag.OHLCBar("NOEXIST")
		assert.ErrorIs(t, err, ErrNotSymbolRegistered, "should throw error when symbol not existed")
	})
}
