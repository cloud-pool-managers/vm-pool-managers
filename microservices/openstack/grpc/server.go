package grpc

import (
	"PoolManagerVM/backend/config"
	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/notifier"
	"PoolManagerVM/backend/pb"
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
)

type ServerMicroOpenstack struct {
	pb.UnimplementedPoolManagerServer
	DB *gorm.DB
}

func Start_grpc() {
	log.Println("gRPC started")
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Error listening: %v", err)
	}

	grpcServer := grpc.NewServer()

	pb.RegisterPoolManagerServer(grpcServer, &ServerMicroOpenstack{DB: config.Database})
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Erreur serveur gRPC: %v", err)
	}
}

func (s *ServerMicroOpenstack) SendRessources(ctx context.Context, req *pb.RessourceRequest) (*emptypb.Empty, error) {
	log.Printf("[SendRessources] User=%s Data=%v", req.GetUser(), req.GetData())
	//create ressources here
	return &emptypb.Empty{}, nil
}

func (s *ServerMicroOpenstack) GetStreamRessources(req *emptypb.Empty, stream pb.PoolManager_GetStreamRessourcesServer) error {
	log.Println("[GetStreamRessources] Stream global started")
	//send all ressources existing
	rows, err := s.DB.Model(&models.Server{}).Rows()
	if err != nil {
		log.Println("Error retrieving servers")
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var serv models.Server
		if err := s.DB.ScanRows(rows, &serv); err != nil {
			log.Println("Error rows server")
			return err
		}
		ret := &pb.StreamRessourceResponse{
			User:   serv.UserID,
			Status: pb.Status_STATUS_UNKNOWN,
			Type:   pb.Type_SERVER,
			Data:   serv.ToMap(),
		}

		if err := stream.Send(ret); err != nil {
			log.Println("error sending server")
			return err
		}
	}

	for {
		select {
		case evt := <-notifier.GlobalChan:
			server, ok := evt.Server.(models.Server)
			if !ok {
				continue
			}
			var status pb.Status
			switch evt.Action {
			case "created":
				status = pb.Status_CREATE
			case "updated":
				status = pb.Status_UPDATE
			case "deleted":
				status = pb.Status_DELETE
			default:
				status = pb.Status_STATUS_UNKNOWN
			}

			err := stream.Send(&pb.StreamRessourceResponse{
				User:   server.UserID,
				Type:   pb.Type_SERVER,
				Status: status,
				Data:   server.ToMap(),
			})
			if err != nil {
				log.Printf("Stream closed for client: %v", err)
				return err
			}

		case <-stream.Context().Done():
			log.Println("[GetStreamRessources] Client disconnected, end of stream")
			return nil
		}
	}
}

func (s *ServerMicroOpenstack) GetStreamRessourcesUser(req *pb.UserRequest, stream grpc.ServerStreamingServer[pb.StreamRessourceResponse]) error {
	log.Println("[GetStreamRessourcesUser] Stream User started")
	// stream user-specific ressources
	return nil
}
