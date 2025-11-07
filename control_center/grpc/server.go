package grpc

import (
	"context"
	"control_center/config"
	"control_center/frontcontrolpb"
	"control_center/internal/auth"
	"log"
	"net"

	"google.golang.org/grpc"
	"gorm.io/gorm"
)

type GatherDataServer struct {
	frontcontrolpb.UnimplementedGatherDataServiceServer
	DB *gorm.DB
}

type ConfigServer struct {
	frontcontrolpb.UnimplementedConfigServiceServer
	DB *gorm.DB
}

type PoolServer struct {
	frontcontrolpb.UnimplementedPoolServiceServer
	DB *gorm.DB
}

type UserServer struct {
	frontcontrolpb.UnimplementedUserServiceServer
	DB *gorm.DB
}

func Start_grpc(ctx context.Context) {
	log.Println("Démarage du serveur gRPC...")

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Erreur lors de l'écoute du port : %v", err)
	}

	s := grpc.NewServer()

	frontcontrolpb.RegisterAuthServiceServer(s, auth.New())
	frontcontrolpb.RegisterGatherDataServiceServer(s, &GatherDataServer{DB: config.Database})
	frontcontrolpb.RegisterConfigServiceServer(s, &ConfigServer{DB: config.Database})
	frontcontrolpb.RegisterPoolServiceServer(s, &PoolServer{DB: config.Database})
	frontcontrolpb.RegisterUserServiceServer(s, &UserServer{DB: config.Database})

	// Lance le serveur dans une goroutine
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Erreur serveur gRPC: %v", err)
		}
	}()

	log.Println("Serveur gRPC lancé sur le port 50051")

	// Attend que le contexte soit annulé
	<-ctx.Done()
	log.Println("Arrêt du serveur gRPC demandé...")

	// Arrêt propre du serveur
	s.GracefulStop()
	log.Println("Serveur gRPC arrêté proprement ✅")
}

// func (s *ControlCenterServer) GetAllImages(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Image]) error {
// 	rows, err := s.DB.Model(&models.Image{}).Rows()
// 	if err != nil {
// 		log.Println("Error retrieving servers")
// 		return err
// 	}
// 	defer rows.Close()

// 	for rows.Next() {
// 		var img models.Image
// 		if err := s.DB.ScanRows(rows, &img); err != nil {
// 			log.Println("Error rows server")
// 			return err
// 		}
// 		if err := stream.Send(img.ToPb()); err != nil {
// 			log.Println("error sending server")
// 			return err
// 		}
// 	}
// 	return nil
// }

// func (s *ControlCenterServer) GetAllFlavors(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Flavor]) error {
// 	rows, err := s.DB.Model(&models.Flavor{}).Rows()
// 	if err != nil {
// 		log.Println("Error retrieving servers")
// 		return err
// 	}
// 	defer rows.Close()

// 	for rows.Next() {
// 		var f models.Flavor
// 		if err := s.DB.ScanRows(rows, &f); err != nil {
// 			log.Println("Error rows server")
// 			return err
// 		}
// 		if err := stream.Send(f.ToPb()); err != nil {
// 			log.Println("error sending server")
// 			return err
// 		}
// 	}
// 	return nil
// }

// func (s *ControlCenterServer) GetAllNetworks(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Network]) error {
// 	rows, err := s.DB.Model(&models.Network{}).Rows()
// 	if err != nil {
// 		log.Println("Error retrieving servers")
// 		return err
// 	}
// 	defer rows.Close()

// 	for rows.Next() {
// 		var n models.Network
// 		if err := s.DB.ScanRows(rows, &n); err != nil {
// 			log.Println("Error rows server")
// 			return err
// 		}
// 		if err := stream.Send(n.ToPb()); err != nil {
// 			log.Println("error sending server")
// 			return err
// 		}
// 	}
// 	return nil
// }
