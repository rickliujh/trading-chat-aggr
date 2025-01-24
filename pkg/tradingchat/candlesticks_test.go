package tradingchat

import (
	"testing"

	bconn "github.com/binance/binance-connector-go"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
)

func TestOHLCCalc(t *testing.T) {
	logger := testr.New(t)
	logger.Info("start")

	t.Run("update should work correctly", func(t *testing.T) {
		// test time from 16:05 to 16:06 on Jan 24th 2025
		// tiemstamp ranging from 1737734700 to	1737734760
		events := []*bconn.WsAggTradeEvent{
			{
				Symbol:    "BNBBTC",
				Price:     "0.11111",
				TradeTime: 1737734701,
			},
			{
				Symbol:    "BNBBTC",
				Price:     "0.11121",
				TradeTime: 1737734711,
			},
			{
				Symbol:    "BNBBTC",
				Price:     "0.11109",
				TradeTime: 1737734709,
			},
			{
				Symbol:    "BNBBTC",
				Price:     "0.11131",
				TradeTime: 1737734744,
			},
			{
				Symbol:    "BNBBTC",
				Price:     "0.11104",
				TradeTime: 1737734759,
			},
			{
				Symbol:    "BNBBTC",
				Price:     "0.11134",
				TradeTime: 1737734731,
			},
		}
		specialEvent := &bconn.WsAggTradeEvent{
			Symbol:    "BNBBTC",
			Price:     "0.11101",
			TradeTime: 1737734760,
		}

		var inittime int64 = 1737734700 - 1
		calc := NewOHLCCalc(inittime)
		assert.Equal(t,
			&OHLCCalc{
				OHLCItem{
					H: "0",
					L: "0",
					O: "0",
					C: "0",
					T: 0,
				},
				inittime,
			},
			calc,
		)

		for _, v := range events {
			calc.Update(v)
		}
		assert.Equal(t,
			&OHLCCalc{
				OHLCItem{
					H: "0.11134",
					L: "0.11104",
					O: "0.11111",
					C: "0.11104",
					T: 1737734759,
				},
				inittime + 60,
			},
			calc,
		)

		calc.Update(specialEvent)
		assert.Equal(t, OHLCItem{
			H: specialEvent.Price,
			L: specialEvent.Price,
			O: specialEvent.Price,
			C: specialEvent.Price,
			T: specialEvent.TradeTime,
		}, calc.Item())
	})

	t.Run("item should be a copy of it", func(t *testing.T) {
		calc := NewOHLCCalc(0)
		oldItem := calc.Item()
		expectedItem := OHLCItem{
			H: "0",
			L: "0",
			O: "0",
			C: "0",
			T: 0,
		}
		assert.Equal(t, expectedItem, oldItem)

		specialEvent := &bconn.WsAggTradeEvent{
			Symbol:    "BNBBTC",
			Price:     "0.11101",
			TradeTime: 1737734760,
		}
		calc.Update(specialEvent)

		assert.Equal(t, expectedItem, oldItem, "oldItem should not change after update of item")

		newItem := calc.Item()
		assert.NotEqual(t, &oldItem, &newItem)
	})
}
