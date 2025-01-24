package tradingchat

import (
	bconn "github.com/binance/binance-connector-go"
	"github.com/go-logr/logr"
)

const (
	BinanceStreamURL = "wss://stream.binance.com:443"
)


func BinanceStreamEventGen(logger *logr.Logger, symbols []string, errHandler func(error), done <-chan struct{}) (chan struct{}, <-chan *bconn.WsAggTradeEvent, error) {
	c := bconn.NewWebsocketStreamClient(true, BinanceStreamURL)
	eventCh := make(chan *bconn.WsAggTradeEvent)

	doneCh, stopCh, err := c.WsCombinedAggTradeServe(
		symbols,
		func(event *bconn.WsAggTradeEvent) {
			logger.V(4).Info("incoming event", "event", event)
			eventCh <- event
		},
		func(err error) {
			logger.V(2).Error(err, "driver stopped")
			errHandler(err)
		},
	)

	if err != nil {
		logger.V(2).Error(err, "starting driver failed")
		return nil, nil, err
	}

	stop := func() { stopCh <- struct{}{} }

	go func() {
		defer close(eventCh)
		for {
			select {
			case <-doneCh:
				return
			case <-done:
				stop()
				return
			}
		}
	}()

	return doneCh, eventCh, nil
}
