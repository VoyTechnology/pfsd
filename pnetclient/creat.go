package network

import (
	"github.com/cpssd/paranoid/pfsd/globals"
	pb "github.com/cpssd/paranoid/proto/paranoidnetwork"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"strconv"
)

func creat(ips []globals.Node, filename, permissions string) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	for _, ipAddr := range ips {
		sendCreateMessage(ipAddr, filename, permissions, opts)
		log.Println("Connecting to: ", ipAddr)
	}
}

func sendCreateMessage(ipAddress globals.Node, filename, permissions string, opts []grpc.DialOption) {
	var permissionsInt uint32
	permissions64, _ := strconv.ParseUint(permissions, 10, 32)
	permissionsInt = uint32(permissions64)

	conn, err := grpc.Dial(ipAddress.IP+":"+ipAddress.Port, opts...)
	if err != nil {
		log.Fatalln("fail to dial: ", err)
	}

	defer conn.Close()
	client := pb.NewParanoidNetworkClient(conn)

	response, err := client.Creat(context.Background(), &pb.CreatRequest{filename, permissionsInt})
	if err != nil {
		log.Println("Failure connecting to", ipAddress.IP+":"+ipAddress.Port)
	}
	log.Println(response)
}
