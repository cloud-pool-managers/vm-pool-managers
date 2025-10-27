package grpc

import (
	"context"
	"log"

	"control_center/pb"

	"gorm.io/gorm"
)

type ControlCenterServer struct {
	pb.UnimplementedControlCenterServer
	DB *gorm.DB
}

func (s *ControlCenterServer) GetRessources(ctx context.Context, req *pb.RessourceRequest) (*pb.RessourceResponse, error) {
	log.Printf("Requête gRPC reçue pour userID=%s", req.Userid)

	// Exemple : lire une ressource dans la DB
	var result map[string]string
	result = map[string]string{"message": "Hello " + req.Userid}

	return &pb.RessourceResponse{
		Userid: req.Userid,
		Data:   result,
	}, nil
}
