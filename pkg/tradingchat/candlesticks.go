package tradingchat

import (
	"time"

	bconn "github.com/binance/binance-connector-go"
	"github.com/rickliujh/trading-chat-aggr/pkg/utils"
)

const (
	Interval1M = time.Second * 60
)

type OHLCBar struct {
	H string `json:"high"`
	L string `json:"low"`
	O string `json:"open"`
	C string `json:"close"`
	T int64  `json:"time"` // Newest time of item
}

type OHLCCalc struct {
	bar    OHLCBar
	endedAt int64
}

func NewOHLCCalc() *OHLCCalc {
	return &OHLCCalc{
		bar: OHLCBar{
			H: "0",
			L: "0",
			O: "0",
			C: "0",
			T: 0,
		},
		endedAt: 0,
	}
}

func (c *OHLCCalc) update(event *bconn.WsAggTradeEvent) {
	logger := utils.NewLogger(0)
	logger.Info("Update", event)
	price := event.Price
	ts := event.TradeTime

	if c.endedAt >= ts {
		if c.bar.H < price {
			c.bar.H = price
		}
		if c.bar.L > price {
			c.bar.L = price
		}
		if c.bar.T < ts {
			c.bar.C = price
			c.bar.T = ts
		}
	} else {
		c.bar.H = price
		c.bar.L = price
		c.bar.O = price
		c.bar.C = price
		c.bar.T = ts
		c.tick(ts)
	}
	logger.Info("After", c.endedAt, c.bar)
}

func (c *OHLCCalc) tick(newTick int64) {
	c.endedAt = time.Unix(newTick, 0).Truncate(time.Minute).Add(59 * time.Second).Unix()
}

func (c *OHLCCalc) Item() OHLCBar {
	return c.bar
}
