package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"connectrpc.com/connect"
	"github.com/go-logr/logr"
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/rickliujh/kickstart-gogrpc/pkg/api/v1"
	"github.com/rickliujh/kickstart-gogrpc/pkg/sql"
	"github.com/rickliujh/kickstart-gogrpc/pkg/utils"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/pkg/errors"
)

const (
	protocol = "tcp" // network protocol
	success  = "processed successfully"
)

func NewServer(name, version, environment string, db *sql.Queries) (*Server, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}
	if version == "" {
		return nil, errors.New("version is required")
	}
	if environment == "" {
		return nil, errors.New("environment is required")
	}

	return &Server{
		counter:     atomic.Uint64{},
		logger:      utils.NewLogger(4),
		db:          db,
		name:        name,
		version:     version,
		environment: environment,
	}, nil
}

// Server is used to implement your Service.
type Server struct {
	counter     atomic.Uint64 // counter for messages
	logger      *logr.Logger
	db          *sql.Queries
	name        string // server name
	version     string // server version
	environment string // server environment
}

func (s *Server) String() string {
	return fmt.Sprintf("%s (%s) %s", s.name, s.environment, s.version)
}

func (s *Server) GetCounter() int64 {
	return int64(s.counter.Load())
}

func (s *Server) GetName() string {
	return s.name
}

func (s *Server) GetVersion() string {
	return s.version
}

func (s *Server) GetEnvironment() string {
	return s.environment
}

// Scalar implements the single method of the Service.
func (s *Server) Scalar(ctx context.Context, req *connect.Request[v1.ScalarRequest]) (*connect.Response[v1.ScalarResponse], error) {
	c := req.Msg.GetContent()

	var jsonOjb map[string]interface{}
	if err := unpackAnyToJSON(c.GetData(), &jsonOjb); err != nil {
		s.logger.Error(err, "can't unmarshal from Content.data Any")
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	s.logger.V(4).Info("received scalar", "data", jsonOjb)

	s.counter.Add(1)
	res := connect.NewResponse(&v1.ScalarResponse{
		RequestId:         c.GetId(),
		MessageCount:      s.GetCounter(),
		MessagesProcessed: s.GetCounter(),
		ProcessingDetails: success,
	})

	items, err := s.db.ListAuthors(ctx)
	if err != nil {
		s.logger.Error(err, "failed to list authors")
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	s.logger.V(4).Info("list result", "authers", items)

	return res, nil
}

// Stream implements the Stream method of the Service.
func (s *Server) Stream(ctx context.Context, strm *connect.BidiStream[v1.StreamRequest, v1.StreamResponse]) error {
	for {
		in, err := strm.Receive()
		if err != nil {
			s.logger.Error(err, "failed to receive")
			return connect.NewError(connect.CodeDataLoss, err)
		}

		c := in.GetContent()
		s.logger.V(4).Info("received stream", "data", c.GetData())

		s.counter.Add(1)
		if err := strm.Send(&v1.StreamResponse{
			RequestId:         in.Content.GetId(),
			MessageCount:      s.GetCounter(),
			MessagesProcessed: s.GetCounter(),
			ProcessingDetails: success,
		}); err != nil {
			s.logger.Error(err, "failed to send")
			return connect.NewError(connect.CodeUnavailable, err)
		}

		var jsonOjb map[string]string
		if err := unpackAnyToJSON(c.GetData(), &jsonOjb); err != nil {
			s.logger.Error(err, "can't unmarshal from Content.data Any")
			return connect.NewError(connect.CodeInvalidArgument, err)
		}

		_, err = s.db.CreateAuthor(ctx, sql.CreateAuthorParams{
			Name: jsonOjb["name"],
			Bio:  pgtype.Text{String: jsonOjb["bio"], Valid: true},
		})
		if err != nil {
			s.logger.Error(err, "unable to create author to db")
		}
	}
}

// packJSONIntoAny converts a JSON-serializable struct into an Any message
func packJSONIntoAny(v interface{}) (*anypb.Any, error) {
	// Convert the struct to JSON bytes
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	// Create Any message
	anyMsg := &anypb.Any{
		// Use a custom type URL for JSON content
		TypeUrl: "type.googleapis.com/json",
		// Store JSON bytes as the value
		Value: jsonBytes,
	}

	return anyMsg, nil
}

// unpackAnyToJSON extracts JSON data from an Any message and unmarshals it into the target struct
func unpackAnyToJSON(anyMsg *anypb.Any, target interface{}) error {
	// Verify type URL (optional, but recommended)
	if anyMsg.TypeUrl != "type.googleapis.com/json" {
		return fmt.Errorf("unexpected type URL: %s", anyMsg.TypeUrl)
	}

	// Unmarshal the value bytes into the target struct
	if err := json.Unmarshal(anyMsg.Value, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON from Any: %w", err)
	}

	return nil
}
