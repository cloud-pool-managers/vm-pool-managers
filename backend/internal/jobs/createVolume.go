package jobs

import (
	"PoolManagerVM/backend/config"
	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/utils"
	"context"
	"log"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v2/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/volumeattach"
)

func CreateVolumeAndAttach(workerID int, job models.Job) error {

	volumeOpts := volumes.CreateOpts{
		Size:        utils.ParseInt(job.Data["size"]),
		Description: job.Data["description"],
		Name:        job.Data["name"],
		VolumeType:  job.Data["volume_type"],
	}

	volumeSchedulerHintOpts := volumes.SchedulerHintOpts{}

	newVolume, err := volumes.Create(context.Background(), models.BlockstorageClient, volumeOpts, volumeSchedulerHintOpts).Extract()
	if err != nil {
		log.Println("Failed to create volume:", err)
		log.Println(volumeOpts)
		ChangePendingVol(job.Data["server_id"])
		return err
	}
	log.Println("Volume created with ID:", newVolume.ID)
	log.Println(newVolume)

	createopts := volumeattach.CreateOpts{
		Device:   "/dev/vdc",
		VolumeID: newVolume.ID,
	}

	res, err := volumeattach.Create(context.TODO(), models.ComputeClient, job.Data["server_id"], createopts).Extract()
	if err != nil {
		log.Println("error :", err)
		ChangePendingVol(job.Data["server_id"])
		return err
	}

	log.Println(res)

	result := config.Database.Model(&models.Server{}).
		Where("id = ?", job.Data["server_id"]).
		Update("attach_volume_id", newVolume.ID).Error
	if result != nil {
		return result
	}

	log.Printf("Worker %d completed the job of creating and attaching a volume\n", workerID)

	return nil

}
