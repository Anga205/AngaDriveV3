package database

type Account struct {
	Token          string `gorm:"primaryKey" json:"token"`
	DisplayName    string `json:"display_name"`
	Email          string `json:"email"`
	HashedPassword string `json:"-"`
}

type Activity struct {
	Timestamps int64
}

func (Activity) TableName() string {
	// Override the default table name, which would be "activities", idk why gorm does this
	// but it does, so we need to override it
	// DO NOT REMOVE THIS FUNCTION
	return "activity"
}

type Collection struct {
	ID        string `gorm:"primaryKey"`
	Name      string
	Editors   string // Comma separated list of AccountToken's
	Size      int
	Dependant string // This is for collections birthed from cloning a github repo, if the github repo collection is deleted, this collection will be deleted too
	Timestamp int64
}

// CollectionFile is the M:N mapping between collections and files.
// Each row means "file_id belongs to collection_id".
//
// A file may belong to many collections and a collection may contain many
// files. The composite primary key (collection_id, file_id) prevents duplicate
// edges.
type CollectionFile struct {
	CollectionID string `gorm:"primaryKey"`
	FileID       string `gorm:"primaryKey"`
}

func (CollectionFile) TableName() string {
	return "collection_files"
}

// CollectionChild is the M:N mapping between collections.
// Each row means "child_collection_id is a child of parent_collection_id".
//
// A collection may have multiple parents and multiple children. Recursive
// (cyclic) graphs are intentionally supported. The ONLY forbidden edge is a
// self-reference: parent_collection_id != child_collection_id, enforced both
// by the database CHECK constraint below and by an application-level guard in
// AddFolder.
type CollectionChild struct {
	ParentCollectionID string `gorm:"primaryKey"`
	ChildCollectionID  string `gorm:"primaryKey;check:collection_child_no_self_ref,parent_collection_id <> child_collection_id"`
}

func (CollectionChild) TableName() string {
	return "collection_children"
}

type FileData struct {
	OriginalFileName string `json:"original_file_name"`
	FileDirectory    string `gorm:"primaryKey" json:"file_directory"`
	AccountToken     string `json:"account_token"`
	FileSize         int64  `json:"file_size"`
	Timestamp        int64  `json:"timestamp"`
	Md5sum           string `json:"-"`
}
