package models

type Student struct {
	ID     uint `gorm:"primaryKey;autoIncrement"`
	ListId uint `gorm:"index;"`
	Name   string
	SshKey string
	IP     string
	// Identité Moodle (renseignée à l'import depuis un cours Moodle).
	// MoodleEmail sert de clé de jointure : login Moodle ↔ ligne student ↔ id nbgrader.
	MoodleEmail  string `gorm:"index"`
	MoodleUserID int
}

type ListStudents struct {
	ID       uint      `gorm:"primaryKey;autoIncrement"`
	PoolId   uint      `gorm:"uniqueIndex"`
	Students []Student `gorm:"foreignKey:ListId;references:ID;constraint:OnDelete:CASCADE"`
}
