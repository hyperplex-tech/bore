package service

import (
	"context"
	"sort"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/hyperplex-tech/bore/internal/config"
	"github.com/hyperplex-tech/bore/internal/engine"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GroupService implements the GroupService gRPC service.
type GroupService struct {
	borev1.UnimplementedGroupServiceServer
	getConfig func() *config.Config
	engine    *engine.Engine
}

// NewGroupService creates a new GroupService.
func NewGroupService(getConfig func() *config.Config, eng *engine.Engine) *GroupService {
	return &GroupService{getConfig: getConfig, engine: eng}
}

func (s *GroupService) ListGroups(ctx context.Context, req *borev1.ListGroupsRequest) (*borev1.ListGroupsResponse, error) {
	var groups []*borev1.Group
	for name, g := range s.getConfig().Groups {
		// Count active tunnels in this group from the engine.
		infos := s.engine.List(name)
		activeCount := 0
		for _, info := range infos {
			if info.Status == engine.StatusActive {
				activeCount++
			}
		}
		groups = append(groups, &borev1.Group{
			Name:        name,
			Description: g.Description,
			TunnelCount: int32(len(g.Tunnels)),
			ActiveCount: int32(activeCount),
		})
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name < groups[j].Name
	})
	return &borev1.ListGroupsResponse{Groups: groups}, nil
}

func (s *GroupService) RenameGroup(ctx context.Context, req *borev1.RenameGroupRequest) (*borev1.RenameGroupResponse, error) {
	if req.OldName == "" || req.NewName == "" {
		return nil, status.Error(codes.InvalidArgument, "old_name and new_name are required")
	}
	configPath := config.ConfigPath()
	if err := config.RenameGroup(configPath, req.OldName, req.NewName); err != nil {
		return nil, status.Errorf(codes.Internal, "renaming group: %v", err)
	}
	return &borev1.RenameGroupResponse{}, nil
}
