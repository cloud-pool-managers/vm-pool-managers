package main

import (
	"context"
	"io"
	"log"
	"time"

	"PoolManagerVM/backend/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

func main() {
	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Erreur de connexion: %v", err)
	}
	defer conn.Close()

	client := pb.NewPoolManagerClient(conn)

	// Test 1 : SendRessources
	_, err = client.SendRessources(context.Background(), &pb.RessourceRequest{
		User: "alice",
		Data: map[string]string{"action": "init"},
	})
	if err != nil {
		log.Fatalf("Erreur SendRessources: %v", err)
	}
	log.Println("✅ SendRessources OK")

	// Test 2 : GetStreamRessources
	stream, err := client.GetStreamRessources(context.Background(), &emptypb.Empty{})
	if err != nil {
		log.Fatalf("Erreur stream: %v", err)
	}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Erreur lecture stream: %v", err)
		}
		log.Printf("📦 Ressource: %+v", resp)
		time.Sleep(time.Second)
	}
}
