package tradingchat

import (
	"time"

	bconn "github.com/binance/binance-connector-go"
	"github.com/rickliujh/trading-chat-aggr/pkg/utils"
)

const (
	Interval1M = time.Second * 60
)

type OHLCItem struct {
	H string `json:"high"`
	L string `json:"low"`
	O string `json:"open"`
	C string `json:"close"`
	T int64  `json:"time"` // Newest time of item
}

type OHLCCalc struct {
	item    OHLCItem
	endedAt int64
}

// inittime should be 1 second before the beginning of the minute of
// the trade time of first event that updates it
func NewOHLCCalc(inittime int64) *OHLCCalc {
	return &OHLCCalc{
		item: OHLCItem{
			H: "0",
			L: "0",
			O: "0",
			C: "0",
			T: 0,
		},
		endedAt: inittime,
	}
}

func (c *OHLCCalc) Update(event *bconn.WsAggTradeEvent) {
	logger := utils.NewLogger(0)
	logger.Info("Update", event)
	price := event.Price
	ts := event.TradeTime

	if c.endedAt >= ts {
		if c.item.H < price {
			c.item.H = price
		}
		if c.item.L > price {
			c.item.L = price
		}
		if c.item.T < ts {
			c.item.C = price
			c.item.T = ts
		}
	} else {
		c.item.H = price
		c.item.L = price
		c.item.O = price
		c.item.C = price
		c.item.T = ts
		c.endedAt = calcEndedTime(c.endedAt, Interval1M)
	}
	logger.Info("After", c.endedAt, c.item)
}

func calcEndedTime(last int64, duration time.Duration) int64 {
	logger := utils.NewLogger(0)
	logger.Info("cacalcEndedTime", last, duration)
	return time.Unix(last, 0).Add(duration).Unix()
}

func (c *OHLCCalc) Item() OHLCItem {
	return c.item
}
