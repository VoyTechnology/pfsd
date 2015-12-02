package pnetserver

import (
	"fmt"
	pb "github.com/cpssd/paranoid/proto/paranoidnetwork"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"log"
)

func (s *ParanoidServer) Mkdir(ctx context.Context, req *pb.MkdirRequest) (*pb.EmptyMessage, error) {
	code, _, err := runCommand(nil, "mkdir", ParanoidDir, req.directory, fmt.Sprint(req.mode))
	if err != nil {
		log.Printf("ERROR: Could not make directory: %v with mode: %v \n", req.directory, req.mode, err)
		returnError := grpc.Errorf(codes.Internal, "Could not make directory: %v with mode: %v\n",
			req.directory, req.mode, err)
		return &pb.EmptyMessage{}, returnError
	}

	returnError := convertCodeToError(code, req.directory)
	// If returnError is nil here, it's equivalent to returning OK
	return &pb.EmptyMessage{}, returnError
}
