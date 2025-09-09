package models

type ServerPool struct {
	IDServerPool    int      `json:"IDServerPool" gorm:"primary_key"`    // clé primaire de ServerPool
	NomServerPool   string   `json:"NomServerPool"`                      // nom du serverpool
	MaxVMServerPool int      `json:"MaxVMServerPool"`                    // nombre max de server du serverpool
	MinVMServerPool int      `json:"MinVMServerPool"`                    // nombre min de server du serverpool
	Servers         []Server `json:"Servers" gorm:"foreignKey:IDServer"` // clé étrangère Server, relation has many
}
