package database

import (
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type FileInfo struct {
	ID          uint   `gorm:"primaryKey"`
	UserAddress string `gorm:"index;not null"`
	FileName    string `gorm:"uniqueIndex:unique_user_file;not null"`
	ProofSetID  int
	CIDs        string
	Root        string
	Status      Status `gorm:"default:'pending'"`
}

func InitDB(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&FileInfo{}); err != nil {
		return nil, err
	}

	return db, nil
}

func InsertData(db *gorm.DB, userAddress, fileName string, proofSetID int, root string, cids []string) error {
	fileInfo := FileInfo{
		UserAddress: userAddress,
		FileName:    fileName,
		ProofSetID:  proofSetID,
		Root:        root,
		CIDs:        strings.Join(cids, " "),
		Status:      StatusPending,
	}
	return db.Create(&fileInfo).Error
}

func UpdateFileStatus(db *gorm.DB, id uint, status Status) error {
	return db.Model(&FileInfo{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func QueryFileInfo(db *gorm.DB, userAddress, fileName string, status Status) ([]string, error) {
	var fileInfo FileInfo
	if err := db.Where("user_address = ? AND file_name = ? AND status = ?", userAddress, fileName, status).
		First(&fileInfo).Error; err != nil {
		return nil, err
	}

	return strings.Split(fileInfo.CIDs, " "), nil
}

func QueryPendingInfo(db *gorm.DB) ([]FileInfo, error) {
	var fileInfos []FileInfo
	if err := db.Where("status = ?", StatusPending).Find(&fileInfos).Error; err != nil {
		return nil, err
	}

	return fileInfos, nil
}

func ListFiles(db *gorm.DB, userAddress string) ([]string, error) {
	var fileInfos []FileInfo
	if err := db.Where("user_address = ?", userAddress).Find(&fileInfos).Error; err != nil {
		return nil, err
	}

	var files []string
	for _, d := range fileInfos {
		files = append(files, d.FileName)
	}

	return files, nil
}
