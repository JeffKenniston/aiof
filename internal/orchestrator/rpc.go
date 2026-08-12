package orchestrator

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	orchestratorv1 "aiof/internal/gen/orchestrator/v1"
	"aiof/internal/store"
)

// RPCServer implements the orchestratorv1connect.OrchestratorServiceHandler
type RPCServer struct {
	logger *slog.Logger
	store  *store.Store
}

func NewRPCServer(logger *slog.Logger, store *store.Store) *RPCServer {
	return &RPCServer{
		logger: logger,
		store:  store,
	}
}

// Pull implements the pullHandler for local-first recovery.
func (s *RPCServer) Pull(ctx context.Context, req *connect.Request[orchestratorv1.PullRequest]) (*connect.Response[orchestratorv1.PullResponse], error) {
	s.logger.Info("Received Pull request", slog.Int64("since", req.Msg.SinceUpdatedAt))

	if s.store == nil {
		s.logger.Error("Database bypassed. Enterprise determinism requires WAL.")
		panic("FATAL: Database bypassed in Pull. Enterprise determinism requires WAL.")
	}

	docs, err := s.store.ReadMutations(ctx, req.Msg.SinceUpdatedAt, int(req.Msg.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := &orchestratorv1.PullResponse{
		Documents: make([]*orchestratorv1.Document, len(docs)),
	}

	for i, d := range docs {
		res.Documents[i] = &orchestratorv1.Document{
			Id:             d.ID,
			DocumentType:   d.DocumentType,
			Payload:        d.Payload,
			UpdatedAt:      d.UpdatedAt,
			IsDeleted:      d.IsDeleted,
			ValidTimeStart: d.ValidTimeStart,
			ValidTimeEnd:   d.ValidTimeEnd,
			SystemTime:     d.SystemTime,
			Provenance:     d.Provenance,
		}
	}

	return connect.NewResponse(res), nil
}

// Push implements the pushHandler for resolving state conflicts.
func (s *RPCServer) Push(ctx context.Context, req *connect.Request[orchestratorv1.PushRequest]) (*connect.Response[orchestratorv1.PushResponse], error) {
	s.logger.Info("Received Push request", slog.Int("count", len(req.Msg.Documents)))

	if s.store == nil {
		s.logger.Error("Database bypassed. Enterprise determinism requires WAL.")
		panic("FATAL: Database bypassed in Push. Enterprise determinism requires WAL.")
	}

	for _, doc := range req.Msg.Documents {
		err := s.store.WriteMutation(ctx, store.Document{
			ID:             doc.Id,
			DocumentType:   doc.DocumentType,
			Payload:        doc.Payload,
			UpdatedAt:      doc.UpdatedAt,
			IsDeleted:      doc.IsDeleted,
			ValidTimeStart: doc.ValidTimeStart,
			ValidTimeEnd:   doc.ValidTimeEnd,
			SystemTime:     doc.SystemTime,
			Provenance:     doc.Provenance,
		})
		if err != nil {
			s.logger.Error("Failed to write mutation", slog.Any("error", err))
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	return connect.NewResponse(&orchestratorv1.PushResponse{Success: true}), nil
}

// PullStream implements real-time SSE / multiplexed streams.
func (s *RPCServer) PullStream(ctx context.Context, req *connect.Request[orchestratorv1.PullStreamRequest], stream *connect.ServerStream[orchestratorv1.PullStreamResponse]) error {
	s.logger.Info("Received PullStream request", slog.Int64("since", req.Msg.SinceUpdatedAt))

	if s.store == nil {
		s.logger.Error("Database bypassed. Enterprise determinism requires WAL.")
		panic("FATAL: Database bypassed in PullStream. Enterprise determinism requires WAL.")
	}

	for _, doc := range s.store.StreamMutations(ctx) {
		err := stream.Send(&orchestratorv1.PullStreamResponse{
			Document: &orchestratorv1.Document{
				Id:             doc.ID,
				DocumentType:   doc.DocumentType,
				Payload:        doc.Payload,
				UpdatedAt:      doc.UpdatedAt,
				IsDeleted:      doc.IsDeleted,
				ValidTimeStart: doc.ValidTimeStart,
				ValidTimeEnd:   doc.ValidTimeEnd,
				SystemTime:     doc.SystemTime,
				Provenance:     doc.Provenance,
			},
		})
		if err != nil {
			s.logger.Error("Failed to send event to stream", slog.Any("error", err))
			return connect.NewError(connect.CodeInternal, err)
		}
	}
	
	s.logger.Info("PullStream context cancelled")
	return ctx.Err()
}
