package utils

func OrDone[T any](done <-chan struct{}, stream <-chan T) <-chan T {
	relayStream := make(chan T)
	go func() {
		defer close(relayStream)
		for {
			select {
			case <-done:
				return
			case data := <-stream:
				select {
				case <-done:
					return
				default:
					relayStream <- data
				}
			}
		}
	}()
	return relayStream
}

