package database

import (
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Status represents the status of a file in the database.
type Status string

const (
	// StatusPending indicates that the file is awaiting processing.
	StatusPending Status = "pending"
	// StatusCompleted indicates that the file has been successfully processed.
	StatusCompleted Status = "completed"
	// StatusFailed indicates that the file processing has failed.
	StatusFailed Status = "failed"
)

// FileInfo represents the structure of a file record in the database.
type FileInfo struct {
	ID          uint   `gorm:"primaryKey"`
	UserAddress string `gorm:"index;not null"`
	FileName    string `gorm:"uniqueIndex:unique_user_file;not null"`
	Size        uint64 `gorm:"not null"`
	ProofSetID  int
	CIDs        string `gorm:"column:cids"`
	Root        string
	Status      Status    `gorm:"default:'pending'"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// InitDB initializes the database connection and migrates the schema.
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

// InsertData inserts a new file record into the database.
func InsertData(db *gorm.DB, userAddress, fileName string, size uint64, proofSetID int, root string, cids []string) error {
	fileInfo := FileInfo{
		UserAddress: userAddress,
		FileName:    fileName,
		Size:        size,
		ProofSetID:  proofSetID,
		Root:        root,
		CIDs:        strings.Join(cids, " "),
		Status:      StatusPending,
	}
	return db.Create(&fileInfo).Error
}

// UpdateFileStatus updates the status of a file record in the database.
func UpdateFileStatus(db *gorm.DB, id uint, status Status) error {
	return db.Model(&FileInfo{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// QueryCIDs retrieves a file record by user address and file name.
func QueryCIDs(db *gorm.DB, userAddress, fileName string, status Status) ([]string, error) {
	cids := ""
	if err := db.Model(&FileInfo{}).Where("user_address = ? AND file_name = ? AND status = ?", userAddress, fileName, status).
		Pluck("cids", &cids).Error; err != nil {
		return nil, err
	}

	return strings.Split(cids, " "), nil
}

// QueryCIDsByRoot retrieves CIDs associated with a specific root and status.
func QueryCIDsByRoot(db *gorm.DB, root string, status Status) ([]string, error) {
	cids := ""
	if err := db.Model(&FileInfo{}).Where("root = ? AND status = ?", root, status).
		Pluck("cids", &cids).Error; err != nil {
		return nil, err
	}

	return strings.Split(cids, " "), nil
}

// QueryPendingInfo retrieves all file records with a status of "pending".
func QueryPendingInfo(db *gorm.DB) ([]FileInfo, error) {
	var fileInfos []FileInfo
	if err := db.Where("status = ?", StatusPending).Find(&fileInfos).Error; err != nil {
		return nil, err
	}

	return fileInfos, nil
}

// ListFiles retrieves all files for a specific user address.
func ListFiles(db *gorm.DB, userAddress string) ([]FileInfo, error) {
	var fileInfos []FileInfo
	if err := db.Where("user_address = ?", userAddress).Find(&fileInfos).Error; err != nil {
		return nil, err
	}

	return fileInfos, nil
}
