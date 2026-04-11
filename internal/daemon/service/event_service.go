package service

import (
	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/hyperplex-tech/bore/internal/event"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EventService implements the EventService gRPC service.
type EventService struct {
	borev1.UnimplementedEventServiceServer
	bus *event.Bus
}

// NewEventService creates a new EventService.
func NewEventService(bus *event.Bus) *EventService {
	return &EventService{bus: bus}
}

func (s *EventService) Subscribe(req *borev1.SubscribeRequest, stream borev1.EventService_SubscribeServer) error {
	id, ch := s.bus.Subscribe(64)
	defer s.bus.Unsubscribe(id)

	// Build a filter set.
	filter := make(map[borev1.EventType]bool)
	for _, t := range req.Types {
		filter[t] = true
	}

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case evt, ok := <-ch:
			if !ok {
				return nil
			}
			protoType := eventTypeToProto(evt.Type)
			if len(filter) > 0 && !filter[protoType] {
				continue
			}
			if err := stream.Send(&borev1.Event{
				Type:       protoType,
				TunnelName: evt.TunnelName,
				Message:    evt.Message,
				Timestamp:  timestamppb.New(evt.Timestamp),
			}); err != nil {
				return err
			}
		}
	}
}

func eventTypeToProto(t event.Type) borev1.EventType {
	switch t {
	case event.TunnelConnected:
		return borev1.EventType_EVENT_TYPE_TUNNEL_CONNECTED
	case event.TunnelDisconnected:
		return borev1.EventType_EVENT_TYPE_TUNNEL_DISCONNECTED
	case event.TunnelError:
		return borev1.EventType_EVENT_TYPE_TUNNEL_ERROR
	case event.TunnelRetrying:
		return borev1.EventType_EVENT_TYPE_TUNNEL_RETRYING
	case event.ConfigReloaded:
		return borev1.EventType_EVENT_TYPE_CONFIG_RELOADED
	default:
		return borev1.EventType_EVENT_TYPE_UNSPECIFIED
	}
}
