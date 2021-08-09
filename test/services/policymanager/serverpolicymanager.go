// Copyright 2020 IBM Corp.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"
	"net"

	mockup "fybrik.io/fybrik/manager/controllers/mockup"
	"fybrik.io/fybrik/manager/controllers/utils"
	pb "fybrik.io/fybrik/pkg/connectors/protobuf"
	"google.golang.org/grpc"
)

const (
	PORT = 50082
)

func main() {
	address := utils.ListeningAddress(PORT)
	log.Printf("starting mock policy manager server on address %s", address)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("listening error: %v", err)
	}

	server := grpc.NewServer()
	service := &mockup.MockPolicyManager{}

	pb.RegisterPolicyManagerServiceServer(server, service)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("cannot serve mock policy manager: %v", err)
	}
}
