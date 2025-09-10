package models

type ServerPool struct {
	ID           uint   `gorm:"primaryKey:autoIncrement"`
	ServerpoolID string `gorm:"index:idx_pool_user, unique"`
	UserID       string `gorm:"index:idx_pool_user, unique"`
	PendingJobs  int
	MinVM        int
	MaxVM        int
	ListServ     []Server `gorm:"foreignKey:PoolID"`
}
