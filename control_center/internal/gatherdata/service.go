package gatherdata

import (
	"control_center/frontcontrolpb"
	"control_center/models"
	"log"

	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
)

type Service struct {
	frontcontrolpb.UnimplementedGatherDataServiceServer
	DB *gorm.DB
}

func New() *Service {
	return &Service{}
}

func (s *Service) GetAllImages(req *emptypb.Empty, stream frontcontrolpb.GatherDataService_GetAllImagesServer) error {
	rows, err := s.DB.Model(&models.Image{}).Rows()
	if err != nil {
		log.Println("Error retrieving images: ", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var img models.Image
		if err := s.DB.ScanRows(rows, &img); err != nil {
			log.Println("Error scanning image row: ", err)
			return err
		}
		if err := stream.Send(img.ToFrontControlPb()); err != nil {
			log.Println("Error sending image: ", err)
			return err
		}
	}
	return nil
}

func (s *Service) GetAllFlavors(req *emptypb.Empty, stream frontcontrolpb.GatherDataService_GetAllFlavorsServer) error {
	rows, err := s.DB.Model(&models.Flavor{}).Rows()
	if err != nil {
		log.Println("Error retrieving flavors: ", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var f models.Flavor
		if err := s.DB.ScanRows(rows, &f); err != nil {
			log.Println("Error scanning flavor row: ", err)
			return err
		}
		if err := stream.Send(f.ToFrontControlPb()); err != nil {
			log.Println("error sending flavor: ", err)
			return err
		}
	}
	return nil
}

func (s *Service) GetAllNetworks(req *emptypb.Empty, stream frontcontrolpb.GatherDataService_GetAllNetworksServer) error {
	rows, err := s.DB.Model(&models.Network{}).Rows()
	if err != nil {
		log.Println("Error retrieving networks: ", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var n models.Network
		if err := s.DB.ScanRows(rows, &n); err != nil {
			log.Println("Error scanning network row: ", err)
			return err
		}
		if err := stream.Send(n.ToFrontControlPb()); err != nil {
			log.Println("Error sending network: ", err)
			return err
		}
	}
	return nil
}
