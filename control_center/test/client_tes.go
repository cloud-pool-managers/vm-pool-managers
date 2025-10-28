package main

import (
	"context"
	"log"
	"time"

	pb "control_center/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connexion au serveur gRPC
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("❌ Impossible de se connecter : %v", err)
	}
	defer conn.Close()

	client := pb.NewControlCenterClient(conn)

	// Préparer la requête
	req := &pb.RessourceRequest{
		Userid: "test-user",
		Data:   map[string]string{"example": "ping"},
	}

	// Appeler la méthode GetRessources
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetRessources(ctx, req)
	if err != nil {
		log.Fatalf("❌ Erreur RPC : %v", err)
	}

	log.Printf("✅ Réponse reçue : %v", resp.Data)
}
