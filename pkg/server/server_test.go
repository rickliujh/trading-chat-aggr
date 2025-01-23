package server
import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	pb "github.com/rickliujh/kickstart-gogrpc/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	anypb "google.golang.org/protobuf/types/known/anypb"
	tspb "google.golang.org/protobuf/types/known/timestamppb"
)

var (
	maxTestRunDuration = 180 * time.Second // 3 minutes
)

func TestScalar(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), maxTestRunDuration)
	defer cancel()

	s := Server{
		counter:     atomic.Uint64{},
		name:        "test-server",
		version:     "v0.0.1",
		environment: "test",
	}

	t.Run("scalar sans args", func(t *testing.T) {
		_, err := NewServer("", "", "")
		assert.Error(t, err)
		_, err = NewServer("test", "", "")
		assert.Error(t, err)
		_, err = NewServer("test", "test", "")
		assert.Error(t, err)
		_, err = NewServer("test", "", "test")
		assert.Error(t, err)
		_, err = NewServer("", "", "test")
		assert.Error(t, err)
	})

	t.Run("scalar sans args", func(t *testing.T) {
		assert.Panics(t, func() { s.Scalar(ctx, nil) }, "should panics if argument is not given")
	})

	t.Run("scalar with args", func(t *testing.T) {
		dict := map[string]interface{}{"John": "Doe", "foo": "bar"}
		bs, _ := json.Marshal(dict)
		data := &anypb.Any{
			TypeUrl: "type.googleapis.com/json",
			Value:   bs,
		}
		req := &connect.Request[pb.ScalarRequest]{
			Msg: &pb.ScalarRequest{
				Content: &pb.Content{
					Id:   uuid.New().String(),
					Data: data,
				},
				Sent: tspb.Now(),
			},
		}

		// Scalar example
		reswrap, err := s.Scalar(ctx, req)
		res := reswrap.Msg
		if err != nil {
			t.Fatalf("error on scalar: %v", err)
		}

		assert.NotEmpty(t, res.GetRequestId())
		assert.Greater(t, res.GetMessageCount(), int64(0))
		assert.Equal(t, res.GetMessagesProcessed(), res.GetMessageCount())
		assert.Equal(t, success, res.GetProcessingDetails())
	})

	t.Run("stream sans args", func(t *testing.T) {
		assert.Panics(t, func() { s.Stream(ctx, nil) }, "should panics if argument is not given")
	})
}
