package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/danilobml/logger-service/data"
	"github.com/danilobml/logger-service/logs"
	"google.golang.org/grpc"
)

type LoggerService struct{
	logs.UnimplementedLoggerServiceServer
	Models data.Models
}

func (l *LoggerService) WriteLog(ctx context.Context, req *logs.LogRequest) (*logs.LogResponse, error) {
	input := req.GetLogEntry()

	logEntry := data.LogEntry{
		Name: input.Name,
		Data: input.Data,
	}

	err := l.Models.LogEntry.Insert(logEntry)
	if err != nil {
		res := &logs.LogResponse{Result: "failed"}
		return res, err
	} 

	res := &logs.LogResponse{Result: "logged!"}
	return res, nil
}

func (app *Config) gRPCListen() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", gRpcPort))
	if err != nil {
		log.Fatal("Failed to listen to gRPC", err)
	}

	s := grpc.NewServer()

	logs.RegisterLoggerServiceServer(s, &LoggerService{Models: app.Models})

	log.Printf("gRPC server listening on port %v", gRpcPort)

	if err := s.Serve(lis); err != nil {
		log.Fatal("Failed to listen to gRPC", err)
	}
}
