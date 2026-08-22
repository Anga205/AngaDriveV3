package database

import (
	"os"
	"sync"
	"testing"
)

// resetState clears all package-level RAM caches and re-initializes the
// database against a fresh temporary directory (a brand-new, empty database).
func resetState(t *testing.T) {
	t.Helper()

	// Reset global RAM structures.
	UserFiles = make(map[string]*FileSet)
	UserCollections = make(map[string]*CollectionSet)
	CollectionFiles = make(map[string]*FileSet)
	CollectionFolders = make(map[string]*CollectionSet)
	TimeStamps = []int64{}
	UserAccountsByEmail = make(map[string]Account)
	UserAccountsByToken = make(map[string]Account)
	FileCache = make(map[string]FileData)
	CollectionCache = make(map[string]Collection)

	// Skip automatic cache loading during init so each test controls when
	// LoadCache runs (simulating startup).
	os.Setenv("SAVE_DRIVE_RAM", "true")
	defer os.Unsetenv("SAVE_DRIVE_RAM")

	if err := InitializeDatabase(t.TempDir()); err != nil {
		t.Fatalf("InitializeDatabase failed: %v", err)
	}
}

// forceLoad reloads all caches into RAM (simulating startup).
func forceLoad(t *testing.T) {
	t.Helper()
	LoadCache()
}

func insertTestFile(t *testing.T, dir string) FileData {
	t.Helper()
	f := FileData{
		OriginalFileName: "test.txt",
		FileDirectory:    dir,
		AccountToken:     "anon-token",
		FileSize:         100,
		Timestamp:        1,
		Md5sum:           "md5-" + dir,
	}
	if err := f.Insert(); err != nil {
		t.Fatalf("insert file failed: %v", err)
	}
	return f
}

func insertTestCollection(t *testing.T, name string) Collection {
	t.Helper()
	c := Collection{Name: name, Editors: "anon-token"}
	if err := c.Insert(); err != nil {
		t.Fatalf("insert collection failed: %v", err)
	}
	return c
}

// countCollectionFiles returns the number of collection_files rows.
func countCollectionFiles(t *testing.T, collectionID string) int64 {
	t.Helper()
	var count int64
	if err := GetDB().Model(&CollectionFile{}).Where("collection_id = ?", collectionID).Count(&count).Error; err != nil {
		t.Fatalf("count collection_files: %v", err)
	}
	return count
}

// countCollectionChildren returns the number of collection_children rows.
func countCollectionChildren(t *testing.T, parentID string) int64 {
	t.Helper()
	var count int64
	if err := GetDB().Model(&CollectionChild{}).Where("parent_collection_id = ?", parentID).Count(&count).Error; err != nil {
		t.Fatalf("count collection_children: %v", err)
	}
	return count
}

func TestFreshDatabaseSchema(t *testing.T) {
	resetState(t)

	var tables []string
	if err := GetDB().Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tables).Error; err != nil {
		t.Fatalf("list tables: %v", err)
	}
	found := map[string]bool{}
	for _, name := range tables {
		found[name] = true
	}
	for _, want := range []string{"accounts", "activity", "collections", "file_data", "collection_files", "collection_children"} {
		if !found[want] {
			t.Fatalf("expected table %q to exist, got tables %v", want, tables)
		}
	}
}

// TestCollectionFileRelationships covers the collection → file relationship.
func TestCollectionFileRelationships(t *testing.T) {
	resetState(t)
	forceLoad(t)

	f1 := insertTestFile(t, "fileA")
	f2 := insertTestFile(t, "fileB")
	f3 := insertTestFile(t, "fileC")

	c1 := insertTestCollection(t, "C1")
	c2 := insertTestCollection(t, "C2")

	// 1. one collection → one file
	if err := c1.AddFile(f1.FileDirectory); err != nil {
		t.Fatalf("add file failed: %v", err)
	}
	if !CollectionFiles[c1.ID].Contains(f1.FileDirectory) {
		t.Fatal("RAM missing file after AddFile")
	}
	if countCollectionFiles(t, c1.ID) != 1 {
		t.Fatalf("expected 1 collection_file edge, got %d", countCollectionFiles(t, c1.ID))
	}

	// 2. one collection → multiple files
	if err := c1.AddFile(f2.FileDirectory); err != nil {
		t.Fatalf("add file failed: %v", err)
	}
	if err := c1.AddFile(f3.FileDirectory); err != nil {
		t.Fatalf("add file failed: %v", err)
	}
	if countCollectionFiles(t, c1.ID) != 3 {
		t.Fatalf("expected 3 collection_file edges, got %d", countCollectionFiles(t, c1.ID))
	}
	if got := len(c1.GetFiles()); got != 3 {
		t.Fatalf("expected 3 files, got %d", got)
	}

	// 3. one file → multiple collections
	if err := c2.AddFile(f1.FileDirectory); err != nil {
		t.Fatalf("add file failed: %v", err)
	}
	var edges int64
	GetDB().Model(&CollectionFile{}).Where("file_id = ?", f1.FileDirectory).Count(&edges)
	if edges != 2 {
		t.Fatalf("expected 2 edges for file, got %d", edges)
	}

	// 4. duplicate mapping ignored
	if err := c1.AddFile(f1.FileDirectory); err != nil {
		t.Fatalf("duplicate add should not error: %v", err)
	}
	GetDB().Model(&CollectionFile{}).Where("collection_id = ? AND file_id = ?", c1.ID, f1.FileDirectory).Count(&edges)
	if edges != 1 {
		t.Fatalf("expected 1 edge (dedup), got %d", edges)
	}

	// 5. removing one mapping doesn't affect others
	if err := c1.RemoveFile(f2.FileDirectory); err != nil {
		t.Fatalf("remove file failed: %v", err)
	}
	if CollectionFiles[c1.ID].Contains(f2.FileDirectory) {
		t.Fatal("RAM still contains removed file")
	}
	if !CollectionFiles[c1.ID].Contains(f1.FileDirectory) {
		t.Fatal("RAM lost unrelated file")
	}
	if !CollectionFiles[c2.ID].Contains(f1.FileDirectory) {
		t.Fatal("other collection lost its mapping")
	}

	// 6. deleting a collection removes its mappings
	if err := c2.Delete(); err != nil {
		t.Fatalf("delete collection failed: %v", err)
	}
	if countCollectionFiles(t, c2.ID) != 0 {
		t.Fatalf("expected 0 edges after collection delete, got %d", countCollectionFiles(t, c2.ID))
	}
	var fileCount int64
	GetDB().Model(&FileData{}).Where("file_directory = ?", f1.FileDirectory).Count(&fileCount)
	if fileCount != 1 {
		t.Fatal("file was deleted when collection relationship removed")
	}

	// 7. deleting a file removes its collection mappings
	if err := DeleteFile(f3, func(Collection) {}); err != nil {
		t.Fatalf("delete file failed: %v", err)
	}
	GetDB().Model(&CollectionFile{}).Where("file_id = ?", f3.FileDirectory).Count(&edges)
	if edges != 0 {
		t.Fatalf("expected 0 edges after file delete, got %d", edges)
	}
}

// TestCollectionChildRelationships covers the collection → collection relationship.
func TestCollectionChildRelationships(t *testing.T) {
	resetState(t)
	forceLoad(t)

	A := insertTestCollection(t, "A")
	B := insertTestCollection(t, "B")
	C := insertTestCollection(t, "C")
	D := insertTestCollection(t, "D")

	// 8. one parent → multiple children
	if err := A.AddFolder(B.ID); err != nil {
		t.Fatalf("add folder failed: %v", err)
	}
	if err := A.AddFolder(C.ID); err != nil {
		t.Fatalf("add folder failed: %v", err)
	}
	if countCollectionChildren(t, A.ID) != 2 {
		t.Fatalf("expected 2 children, got %d", countCollectionChildren(t, A.ID))
	}

	// 9. one child → multiple parents
	if err := D.AddFolder(B.ID); err != nil {
		t.Fatalf("add folder failed: %v", err)
	}
	var count int64
	GetDB().Model(&CollectionChild{}).Where("child_collection_id = ?", B.ID).Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 parents for B, got %d", count)
	}

	// 10. multiple levels of nesting
	if err := B.AddFolder(C.ID); err != nil {
		t.Fatalf("add folder failed: %v", err)
	}
	GetDB().Model(&CollectionChild{}).Where("parent_collection_id = ? AND child_collection_id = ?", B.ID, C.ID).Count(&count)
	if count != 1 {
		t.Fatalf("expected B->C edge, got %d", count)
	}

	// 11. recursive relationships allowed: C -> A
	if err := C.AddFolder(A.ID); err != nil {
		t.Fatalf("recursive add should be allowed: %v", err)
	}
	GetDB().Model(&CollectionChild{}).Where("parent_collection_id = ? AND child_collection_id = ?", C.ID, A.ID).Count(&count)
	if count != 1 {
		t.Fatalf("expected C->A edge, got %d", count)
	}

	// 12. A -> A rejected
	if err := A.AddFolder(A.ID); err == nil {
		t.Fatal("self-reference should be rejected")
	}

	// 13. A -> B and B -> A both allowed
	if err := B.AddFolder(A.ID); err != nil {
		t.Fatalf("B->A should be allowed: %v", err)
	}
	GetDB().Model(&CollectionChild{}).Where("parent_collection_id = ? AND child_collection_id = ?", B.ID, A.ID).Count(&count)
	if count != 1 {
		t.Fatalf("expected B->A edge, got %d", count)
	}

	// 14. duplicate (parent, child) prevented
	if err := A.AddFolder(B.ID); err != nil {
		t.Fatalf("duplicate add should not error: %v", err)
	}
	GetDB().Model(&CollectionChild{}).Where("parent_collection_id = ? AND child_collection_id = ?", A.ID, B.ID).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 A->B edge (dedup), got %d", count)
	}

	// 15. removing one parent relationship does not delete the child
	if err := D.RemoveFolder(B.ID); err != nil {
		t.Fatalf("remove folder failed: %v", err)
	}
	var childCount int64
	GetDB().Model(&Collection{}).Where("id = ?", B.ID).Count(&childCount)
	if childCount != 1 {
		t.Fatal("child collection was deleted when parent relationship removed")
	}
	GetDB().Model(&CollectionChild{}).Where("parent_collection_id = ? AND child_collection_id = ?", A.ID, B.ID).Count(&count)
	if count != 1 {
		t.Fatalf("expected A->B to remain, got %d", count)
	}

	// 16. deleting a collection removes all edges involving it
	if err := B.Delete(); err != nil {
		t.Fatalf("delete collection failed: %v", err)
	}
	GetDB().Model(&CollectionChild{}).Where("parent_collection_id = ? OR child_collection_id = ?", B.ID, B.ID).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 edges involving B, got %d", count)
	}
	var remaining int64
	GetDB().Model(&Collection{}).Count(&remaining)
	if remaining != 3 { // A, C, D remain
		t.Fatalf("expected 3 collections to remain, got %d", remaining)
	}
}

// TestSelfReferenceRejectedAtDBLevel verifies the database CHECK constraint
// independently rejects a direct self-reference.
func TestSelfReferenceRejectedAtDBLevel(t *testing.T) {
	resetState(t)
	forceLoad(t)

	A := insertTestCollection(t, "A")
	err := GetDB().Create(&CollectionChild{
		ParentCollectionID: A.ID,
		ChildCollectionID:  A.ID,
	}).Error
	if err == nil {
		t.Fatal("database should reject parent_collection_id == child_collection_id")
	}
}

// TestStartupLoadingReconstructsRelationships covers RAM reload/startup.
func TestStartupLoadingReconstructsRelationships(t *testing.T) {
	resetState(t)

	f1 := insertTestFile(t, "fileX")
	f2 := insertTestFile(t, "fileY")
	A := insertTestCollection(t, "A")
	B := insertTestCollection(t, "B")
	C := insertTestCollection(t, "C")

	if err := A.AddFile(f1.FileDirectory); err != nil {
		t.Fatal(err)
	}
	if err := A.AddFile(f2.FileDirectory); err != nil {
		t.Fatal(err)
	}
	if err := A.AddFolder(B.ID); err != nil {
		t.Fatal(err)
	}
	if err := A.AddFolder(C.ID); err != nil {
		t.Fatal(err)
	}

	// Simulate a restart by clearing RAM and reloading.
	CollectionFiles = make(map[string]*FileSet)
	CollectionFolders = make(map[string]*CollectionSet)
	FileCache = make(map[string]FileData)
	CollectionCache = make(map[string]Collection)
	UserFiles = make(map[string]*FileSet)
	UserCollections = make(map[string]*CollectionSet)

	forceLoad(t)

	gotFiles := len(CollectionFiles[A.ID].Keys())
	if gotFiles != 2 {
		t.Fatalf("expected 2 files after reload, got %d", gotFiles)
	}
	if !CollectionFiles[A.ID].Contains(f1.FileDirectory) || !CollectionFiles[A.ID].Contains(f2.FileDirectory) {
		t.Fatal("files not accumulated correctly after reload")
	}
	gotFolders := len(CollectionFolders[A.ID].Keys())
	if gotFolders != 2 {
		t.Fatalf("expected 2 children after reload, got %d", gotFolders)
	}
	if !CollectionFolders[A.ID].Contains(B.ID) || !CollectionFolders[A.ID].Contains(C.ID) {
		t.Fatal("children not accumulated correctly after reload")
	}
}

// TestCollectionSizeCalculation verifies size computation and propagation.
func TestCollectionSizeCalculation(t *testing.T) {
	resetState(t)
	forceLoad(t)

	f1 := insertTestFile(t, "sizeF1") // 100 bytes
	f2 := insertTestFile(t, "sizeF2") // 100 bytes

	root := insertTestCollection(t, "root")
	child := insertTestCollection(t, "child")
	grandchild := insertTestCollection(t, "grandchild")

	// root → child → grandchild
	if err := root.AddFolder(child.ID); err != nil {
		t.Fatal(err)
	}
	if err := child.AddFolder(grandchild.ID); err != nil {
		t.Fatal(err)
	}

	if err := root.AddFile(f1.FileDirectory); err != nil {
		t.Fatal(err)
	}
	if err := grandchild.AddFile(f2.FileDirectory); err != nil {
		t.Fatal(err)
	}

	// root = 100 (own file) + 100 (grandchild's file) = 200.
	reloaded, _ := GetCollection(root.ID)
	if reloaded.Size != 200 {
		t.Fatalf("expected root size 200, got %d", reloaded.Size)
	}
	reloadedChild, _ := GetCollection(child.ID)
	if reloadedChild.Size != 100 {
		t.Fatalf("expected child size 100, got %d", reloadedChild.Size)
	}

	// Removing the grandchild's file must propagate up to root.
	if err := grandchild.RemoveFile(f2.FileDirectory); err != nil {
		t.Fatal(err)
	}
	reloaded, _ = GetCollection(root.ID)
	if reloaded.Size != 100 {
		t.Fatalf("expected root size 100 after removal, got %d", reloaded.Size)
	}
}

// TestCyclicCollectionSizeTerminates ensures cyclic graphs don't hang size calc.
func TestCyclicCollectionSizeTerminates(t *testing.T) {
	resetState(t)
	forceLoad(t)

	A := insertTestCollection(t, "A")
	B := insertTestCollection(t, "B")

	// A -> B and B -> A (a 2-cycle).
	if err := A.AddFolder(B.ID); err != nil {
		t.Fatal(err)
	}
	if err := B.AddFolder(A.ID); err != nil {
		t.Fatal(err)
	}

	ra, _ := GetCollection(A.ID)
	rb, _ := GetCollection(B.ID)
	if ra.Size != 0 || rb.Size != 0 {
		t.Fatalf("expected cycle sizes 0/0, got %d/%d", ra.Size, rb.Size)
	}
}

// TestConcurrentAddFile exercises concurrent relationship additions.
func TestConcurrentAddFile(t *testing.T) {
	resetState(t)
	forceLoad(t)

	c := insertTestCollection(t, "C")
	const n = 50
	files := make([]FileData, n)
	for i := 0; i < n; i++ {
		files[i] = insertTestFile(t, "concFile"+string(rune('a'+i%26))+string(rune('A'+i/26)))
	}

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(dir string) {
			defer wg.Done()
			_ = c.AddFile(dir)
		}(files[i].FileDirectory)
	}
	wg.Wait()

	if got := len(c.GetFiles()); got != n {
		t.Fatalf("expected %d files in RAM, got %d", n, got)
	}
	if countCollectionFiles(t, c.ID) != n {
		t.Fatalf("expected %d edges in DB, got %d", n, countCollectionFiles(t, c.ID))
	}
}

// TestConcurrentAddRemoveFile exercises concurrent add/remove of the same file.
func TestConcurrentAddRemoveFile(t *testing.T) {
	resetState(t)
	forceLoad(t)

	c := insertTestCollection(t, "C")
	f := insertTestFile(t, "contended")

	if err := c.AddFile(f.FileDirectory); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = c.AddFile(f.FileDirectory)
		}()
		go func() {
			defer wg.Done()
			_ = c.RemoveFile(f.FileDirectory)
		}()
	}
	wg.Wait()

	// Final state must be consistent: RAM presence matches DB presence.
	dbHas := countCollectionFiles(t, c.ID) > 0
	ramHas := CollectionFiles[c.ID].Contains(f.FileDirectory)
	if dbHas != ramHas {
		t.Fatalf("inconsistent state: DB has=%v RAM has=%v", dbHas, ramHas)
	}
}
