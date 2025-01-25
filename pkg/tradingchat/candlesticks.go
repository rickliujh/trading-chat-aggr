package tradingchat

import (
	"time"

	bconn "github.com/binance/binance-connector-go"
	"github.com/go-logr/logr"
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
	bar     OHLCBar
	endedAt int64
	logger  logr.Logger
}

func NewOHLCCalc(logger logr.Logger) *OHLCCalc {
	return &OHLCCalc{
		bar: OHLCBar{
			H: "0",
			L: "0",
			O: "0",
			C: "0",
			T: 0,
		},
		logger:  logger,
		endedAt: 0,
	}
}

func (c *OHLCCalc) update(event *bconn.WsAggTradeEvent) {
	price := event.Price
	ts := event.TradeTime

	c.logger.V(4).Info("OHLCCalc before update", "OHLCCalc", c, "event", event)
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
	c.logger.V(4).Info("OHLCCalc updated", "OHLCCalc", c, "event", event)
}

func (c *OHLCCalc) tick(newTick int64) {
	newEndedAt := time.Unix(newTick, 0).Truncate(time.Minute).Add(59 * time.Second).Unix()
	c.endedAt = newEndedAt
	c.logger.V(4).Info("tick updated", "newtick", newTick, "old-endedAt", c.endedAt, "new_endedAt", newEndedAt)
}

func (c *OHLCCalc) Bar() OHLCBar {
	return c.bar
}
