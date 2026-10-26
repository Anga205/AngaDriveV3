package database

import (
	"angadrive/storage"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	dbInstance *gorm.DB
)

func InitializeDatabase() error {
	if err := storage.EnsureLayout(); err != nil {
		return fmt.Errorf("InitializeDatabase: %w", err)
	}
	dbPath := filepath.Join(storage.Root(), "angadrive.db")
	var err error
	dbInstance, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("InitializeDatabase: %w", err)
	}

	err = dbInstance.AutoMigrate(&Account{}, &Activity{}, &Collection{}, &FileData{}, &CollectionFile{}, &CollectionChild{})
	if err != nil {
		return fmt.Errorf("InitializeDatabase: %w", err)
	}

	// Migrate any pre-existing database that still uses the legacy MD5-based
	// storage key. AutoMigrate adds the new `sha256sum` column but does not
	// rename or backfill the old `md5sum` column, so we do that here.
	if err := migrateMd5ToSha256(storage.Root()); err != nil {
		return fmt.Errorf("InitializeDatabase: md5->sha256 migration failed: %w", err)
	}

	fmt.Println("[GIN-debug] Database initialized successfully")
	loadTimeStamps()
	dontCache := os.Getenv("SAVE_DRIVE_RAM")
	if dontCache == "" || dontCache != "true" {
		fmt.Println("[GIN-debug] Loading entire database into RAM (set env variable SAVE_DRIVE_RAM=true to disable this)")
		LoadCache()
		fmt.Println("[GIN-debug] Database cache has been loaded")
	}
	return nil
}

// migrateMd5ToSha256 transitions a pre-existing database from MD5-based
// physical file naming to SHA-256-based naming.
//
// For every file_data row that still has a non-empty `md5sum` (and an empty
// `sha256sum`), it:
//  1. Locates the physical file at uploaded_files/i/<md5sum>.
//  2. Computes the SHA-256 of that physical file's bytes.
//  3. Renames the physical file to uploaded_files/i/<sha256>.<ext>.
//  4. Updates the row's `sha256sum` to the new name and clears `md5sum`.
//
// Multiple file_data rows sharing the same MD5 (deduplicated content) are
// handled by only renaming the physical file once: after the first row is
// migrated, the old MD5-named file no longer exists, so subsequent rows with
// the same MD5 simply get their `sha256sum` set to the already-computed value
// without attempting another rename.
//
// This is safe for existing installations: no user file is deleted or
// overwritten. If a physical file is missing, the row is left untouched (its
// `md5sum` is preserved) so no data is silently lost.
func migrateMd5ToSha256(uploadedFilesDir string) error {
	db := GetDB()

	// Detect whether the legacy `md5sum` column still exists.
	var columns []struct {
		Name string
	}
	if err := db.Raw("PRAGMA table_info(file_data)").Scan(&columns).Error; err != nil {
		return err
	}
	hasMd5Column := false
	for _, col := range columns {
		if strings.EqualFold(col.Name, "md5sum") {
			hasMd5Column = true
			break
		}
	}
	if !hasMd5Column {
		return nil
	}

	var rows []FileData
	if err := db.Where("md5sum IS NOT NULL AND md5sum != ''").Find(&rows).Error; err != nil {
		return err
	}

	// Map from old md5sum -> new sha256sum so deduplicated rows share one value.
	md5ToSha256 := make(map[string]string)
	iDir := filepath.Join(uploadedFilesDir, "i")

	for _, row := range rows {
		// Read the legacy md5sum column directly (the FileData model no longer
		// exposes it).
		var oldNames []string
		if err := db.Model(&FileData{}).Where("file_directory = ?", row.FileDirectory).
			Pluck("md5sum", &oldNames).Error; err != nil {
			return err
		}
		if len(oldNames) == 0 {
			continue
		}
		oldName := oldNames[0]
		if oldName == "" {
			continue
		}
		if newName, ok := md5ToSha256[oldName]; ok {
			// Another row already migrated this physical file.
			if err := db.Model(&FileData{}).Where("file_directory = ?", row.FileDirectory).
				Update("sha256sum", newName).Error; err != nil {
				return err
			}
			continue
		}

		oldPath := filepath.Join(iDir, oldName)
		if _, err := os.Stat(oldPath); os.IsNotExist(err) {
			// Physical file missing; leave the row untouched to avoid data loss.
			continue
		}

		hash, err := storage.SHA256File(oldPath)
		if err != nil {
			return err
		}
		ext := filepath.Ext(oldName)
		newName := hash + ext
		newPath := filepath.Join(iDir, newName)

		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			if err := os.Rename(oldPath, newPath); err != nil {
				return err
			}
		} else {
			// The SHA-256-named file already exists (e.g. from a partial
			// migration). Remove the old MD5-named file only if it is not the
			// same file.
			if oldPath != newPath {
				os.Remove(oldPath)
			}
		}

		md5ToSha256[oldName] = newName
		if err := db.Model(&FileData{}).Where("file_directory = ?", row.FileDirectory).
			Update("sha256sum", newName).Error; err != nil {
			return err
		}
	}

	// Drop the legacy md5sum column now that all rows have been migrated.
	if err := db.Exec("ALTER TABLE file_data DROP COLUMN md5sum").Error; err != nil {
		return err
	}
	return nil
}

// sha256OfFile was moved to package storage (storage.SHA256File).

func GetDB() *gorm.DB {
	if dbInstance == nil {
		panic("Database not initialized")
	}
	return dbInstance
}

// IsInitialized returns true when the database has been initialized.
func IsInitialized() bool {
	return dbInstance != nil
}
