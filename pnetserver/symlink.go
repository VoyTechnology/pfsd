package pnetserver

import (
	"github.com/cpssd/paranoid/libpfs/commands"
	"github.com/cpssd/paranoid/libpfs/returncodes"
	pb "github.com/cpssd/paranoid/proto/paranoidnetwork"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"log"
)

func (s *ParanoidServer) Symlink(ctx context.Context, req *pb.LinkRequest) (*pb.EmptyMessage, error) {
	code, err := commands.SymlinkCommand(ParanoidDir, req.OldPath, req.NewPath, false)
	if code != returncodes.OK {
		log.Printf("ERROR: Could not link file %s to %s: %v.\n", req.OldPath, req.NewPath, err)
		returnError := grpc.Errorf(codes.Internal, "could not link file %s to %s: %v",
			req.OldPath, req.NewPath, err)
		return &pb.EmptyMessage{}, returnError
	}

	return &pb.EmptyMessage{}, nil
}
