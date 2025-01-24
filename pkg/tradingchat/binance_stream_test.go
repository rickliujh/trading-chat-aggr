package tradingchat

import (
	"testing"

	// bconn "github.com/binance/binance-connector-go"
	// "github.com/go-logr/logr/testr"
	// "github.com/stretchr/testify/assert"
)

func TestBinanceStreamDriver(t *testing.T) {
	// logger := testr.New(t)
	// t.Run("test dispatch", func(t *testing.T) {
	// 	events := []*bconn.WsAggTradeEvent{
	// 		{
	// 			Symbol: "BNBBTC",
	// 			Price:  "0.11111",
	// 		},
	// 		{
	// 			Symbol: "ETHBTC",
	// 			Price:  "0.11112",
	// 		},
	// 		{
	// 			Symbol: "BNBBTC",
	// 			Price:  "0.11113",
	// 		},
	// 		{
	// 			Symbol: "ETHBTC",
	// 			Price:  "0.11114",
	// 		},
	// 	}
	//
	// 	eventDriver := func() (<-chan *bconn.WsAggTradeEvent, <-chan error, StopGen) {
	// 		ch := make(chan *bconn.WsAggTradeEvent)
	// 		go func() {
	// 			for _, e := range events {
	// 				ch <- e
	// 			}
	// 			close(ch)
	// 		}()
	// 		return ch, nil, func() {}
	// 	}
	//
	// 	res := []*bconn.WsAggTradeEvent{}
	// 	handler := func(event *bconn.WsAggTradeEvent) {
	// 		res = append(res, event)
	// 	}
	//
	// 	dispatchList := map[string]func(event *bconn.WsAggTradeEvent){
	// 		"BNBBTC": handler,
	// 		"ETHBTC": handler,
	// 	}
	//
	// 	Dispatch(&logger, eventDriver, dispatchList, func(err error) {})
	//
	// 	assert.Equal(t, events, res, "total events visited should match it dispatched")
	// })
}
