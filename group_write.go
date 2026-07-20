package hdf5

import (
	"encoding/binary"
	"fmt"
	"sort"
	"strings"

	"github.com/huangzhengshun/hdf5/internal/core"
	"github.com/huangzhengshun/hdf5/internal/structures"
	"github.com/huangzhengshun/hdf5/internal/writer"
)

type btreeEntry struct {
	key       uint64
	childAddr uint64
}

// GroupWriter represents an HDF5 group opened for writing.
// Groups organize datasets and other groups in a hierarchical structure.
//
// This type enables writing attributes to groups, similar to datasets.
// It provides a clean, object-oriented API consistent with DatasetWriter.
//
// Example:
//
//	fw, _ := hdf5.CreateForWrite("data.h5", hdf5.CreateTruncate)
//	defer fw.Close()
//
//	// Create group
//	group, _ := fw.CreateGroup("/mygroup")
//
//	// Write attributes to group
//	group.WriteAttribute("description", "My data group")
//	group.WriteAttribute("version", int32(1))
//
// Note: This is a write-only handle. For reading group contents, use
// the file-level Walk() or Group() methods after reopening the file.
type GroupWriter struct {
	// path is the full path of this group (e.g., "/mygroup" or "/data/experiments")
	path string

	// headerAddr is the address of the group's object header in the HDF5 file.
	// This is used for writing attributes and linking to this group.
	headerAddr uint64

	// file is a reference to the parent FileWriter.
	// This is needed for attribute operations and accessing file-level structures.
	file *FileWriter
}

// WriteAttribute writes an attribute to this group.
//
// Storage strategy (automatic):
//   - 0-7 attributes: Compact storage (object header messages)
//   - 8+ attributes: Dense storage (Fractal Heap + B-tree v2)
//
// Supported value types:
//   - Scalars: int8, int16, int32, int64, uint8, uint16, uint32, uint64, float32, float64
//   - Arrays: []int32, []float64, etc. (1D arrays only)
//   - Strings: string (fixed-length, converted to byte array)
//   - String arrays: []string (variable-length strings via Global Heap)
//
// Parameters:
//   - name: Attribute name (ASCII, no null bytes)
//   - value: Attribute value (Go scalar, slice, or string)
//
// Returns:
//   - error: If attribute cannot be written
//
// Example:
//
//	group, _ := fw.CreateGroup("/mygroup")
//	group.WriteAttribute("MATLAB_class", "double")
//	group.WriteAttribute("MATLAB_complex", uint8(1))
//	group.WriteAttribute("description", "Temperature measurements")
//	group.WriteAttribute("topics", []string{"camera", "lidar", "imu"})
//
// Limitations:
//   - No compound types
//   - Attributes cannot be modified after creation (write-once)
//   - No attribute deletion
func (g *GroupWriter) WriteAttribute(name string, value interface{}) error {
	// Delegate to existing attribute writing infrastructure
	// This reuses the same code path as DatasetWriter.WriteAttribute
	return writeAttribute(g.file, g.headerAddr, name, value)
}

// DeleteAttribute removes an attribute by name from this group.
//
// This method supports both compact and dense attribute storage:
//   - Compact storage (0-7 attributes): Removes message from object header
//   - Dense storage (8+ attributes): Removes from B-tree and fractal heap
//
// Parameters:
//   - name: Attribute name to delete
//
// Returns:
//   - error: If attribute not found or deletion fails
//
// Example:
//
//	group, _ := fw.CreateGroup("/mygroup")
//	group.WriteAttribute("temp", int32(42))
//	group.DeleteAttribute("temp") // Remove attribute
//
// Reference: H5Adelete.c - H5A__delete().
func (g *GroupWriter) DeleteAttribute(name string) error {
	return deleteAttribute(g.file, g.headerAddr, name)
}

// Path returns the full path of this group.
//
// This can be used to display the group's location in the file hierarchy
// or for debugging purposes.
//
// Returns:
//   - string: The group's path (e.g., "/mygroup" or "/data/experiments")
//
// Example:
//
//	group, _ := fw.CreateGroup("/mygroup")
//	fmt.Println(group.Path()) // Output: /mygroup
func (g *GroupWriter) Path() string {
	return g.path
}

// validateGroupPath validates group path is not empty, starts with '/', and is not root.
func validateGroupPath(path string) error {
	if path == "" {
		return fmt.Errorf("group path cannot be empty")
	}
	if path[0] != '/' {
		return fmt.Errorf("group path must start with '/' (got %q)", path)
	}
	if path == "/" {
		return fmt.Errorf("root group already exists")
	}
	return nil
}

// createGroupStructures creates and writes the local heap, symbol table node, and B-tree for a group.
// Returns (heapAddr, stNodeAddr, btreeAddr, error).
func (fw *FileWriter) createGroupStructures() (uint64, uint64, uint64, error) {
	offsetSize := int(fw.file.sb.OffsetSize)

	// Create local heap (4096 bytes supports ~300+ typical names).
	heap := structures.NewLocalHeap(4096)
	heapAddr, err := fw.writer.Allocate(heap.Size())
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to allocate heap: %w", err)
	}

	// Create symbol table node with capacity = 2*K where K=4 (GroupLeafNodeK).
	stNode := structures.NewSymbolTableNode(snodCapacity)
	stNodeAddr, err := fw.writer.Allocate(snodTotalSize)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to allocate symbol table node: %w", err)
	}

	if err := stNode.WriteAt(fw.writer, stNodeAddr, uint8(offsetSize), snodCapacity, fw.file.sb.Endianness); err != nil { //nolint:gosec // Safe: offsetSize is 8
		return 0, 0, 0, fmt.Errorf("failed to write symbol table node: %w", err)
	}

	// Create B-tree
	btree := structures.NewBTreeNodeV1(0, 16)
	if err := btree.AddKey(0, stNodeAddr); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to add B-tree key: %w", err)
	}

	btreeSize := uint64(24 + (2*16+1)*offsetSize + 2*16*offsetSize)
	btreeAddr, err := fw.writer.Allocate(btreeSize)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to allocate B-tree: %w", err)
	}

	if err := btree.WriteAt(fw.writer, btreeAddr, uint8(offsetSize), 16, fw.file.sb.Endianness); err != nil { //nolint:gosec // Safe: offsetSize is 8
		return 0, 0, 0, fmt.Errorf("failed to write B-tree: %w", err)
	}

	// Write heap
	if err := heap.WriteTo(fw.writer, heapAddr, uint8(offsetSize), uint8(offsetSize)); err != nil { //nolint:gosec // Safe: offsetSize is 8
		return 0, 0, 0, fmt.Errorf("failed to write local heap: %w", err)
	}

	return heapAddr, stNodeAddr, btreeAddr, nil
}

// CreateGroup creates a new empty group in the HDF5 file.
// Groups organize datasets and other groups in a hierarchical structure.
//
// This method creates an empty group using symbol table format (old HDF5 format).
// For groups with many links, consider using CreateDenseGroup() or CreateGroupWithLinks().
//
// Parameters:
//   - path: Group path (must start with "/", e.g., "/data" or "/data/experiments")
//
// Returns:
//   - *GroupWriter: Handle for writing attributes to the group
//   - error: If creation fails
//
// Example:
//
//	fw, _ := hdf5.CreateForWrite("data.h5", hdf5.CreateTruncate)
//	defer fw.Close()
//
//	// Create root-level group
//	group, _ := fw.CreateGroup("/data")
//	group.WriteAttribute("description", "My data group")
//
//	// Create nested group
//	nested, _ := fw.CreateGroup("/data/experiments")
//	nested.WriteAttribute("MATLAB_class", "double")
//
// Limitations for MVP (v0.11.0-beta):
//   - Only symbol table structure (no indexed groups)
//   - No link creation time tracking
//   - Maximum 32 entries per group (symbol table node capacity)
//   - Parent group must exist (create parents first)
func (fw *FileWriter) CreateGroup(path string) (*GroupWriter, error) {
	// Validate path
	if err := validateGroupPath(path); err != nil {
		return nil, err
	}

	// Parse path into parent and name
	parent, name := parsePath(path)

	// Validate parent exists (if not root)
	if parent != "" && parent != "/" {
		if _, exists := fw.groups[parent]; !exists {
			return nil, fmt.Errorf("parent group %q does not exist (create it first)", parent)
		}
	}

	// Create group structures (heap, symbol table, B-tree)
	heapAddr, stNodeAddr, btreeAddr, err := fw.createGroupStructures()
	if err != nil {
		return nil, err
	}

	// Create object header for the group
	// Message 1: Symbol Table Message (type 0x11)
	stMsg := core.EncodeSymbolTableMessage(btreeAddr, heapAddr, int(fw.file.sb.OffsetSize), int(fw.file.sb.LengthSize))

	ohw := &core.ObjectHeaderWriter{
		Version: 2,
		Flags:   0,
		Messages: []core.MessageWriter{
			{Type: core.MsgSymbolTable, Data: stMsg},
		},
	}

	// Pre-allocate OHDR with padding to accommodate future attributes.
	// This prevents corruption when attributes are added later.
	ohw.PadToSize(core.MinOHDRAllocSize)
	headerSize := ohw.Size()

	headerAddr, err := fw.writer.Allocate(headerSize)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate object header: %w", err)
	}

	// Write object header
	writtenSize, err := ohw.WriteTo(fw.writer, headerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to write object header: %w", err)
	}

	if writtenSize != headerSize {
		return nil, fmt.Errorf("header size mismatch: expected %d, wrote %d", headerSize, writtenSize)
	}

	// Store group metadata for nested dataset linking
	fw.groups[path] = &GroupMetadata{
		heapAddr:      heapAddr,
		stNodeAddr:    stNodeAddr,
		btreeAddr:     btreeAddr,
		headerAddr:    headerAddr,
		headerAllocSz: headerSize,
	}

	// Link to parent group
	if err := fw.linkToParent(parent, name, headerAddr); err != nil {
		return nil, fmt.Errorf("failed to link to parent: %w", err)
	}

	// Return GroupWriter handle
	return &GroupWriter{
		path:       path,
		headerAddr: headerAddr,
		file:       fw,
	}, nil
}

// parsePath splits a path into parent directory and name.
// Examples:
//   - "/group1" → ("", "group1")
//   - "/data/experiments" → ("/data", "experiments")
//   - "/" → ("", "")
func parsePath(path string) (parent, name string) {
	if path == "/" {
		return "", ""
	}

	// Remove trailing slash if present
	path = strings.TrimSuffix(path, "/")

	// Find last slash
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash == 0 {
		// Root-level path like "/group1"
		return "", path[1:] // Return ("", "group1")
	}

	// Nested path like "/data/experiments"
	return path[:lastSlash], path[lastSlash+1:]
}

// linkToParent links a child object to its parent group.
// Links the child by adding an entry to the parent's symbol table.
// When the SNOD is full (8 entries for K=4), it splits per the C library algorithm
// (H5Gnode.c:598-637). When the local heap is full, it reallocates a larger one.
//
// Parameters:
//   - parentPath: Path to parent group ("" or "/" for root)
//   - childName: Name of the child object
//   - childAddr: Address of the child's object header
//
// Returns:
//   - error: If linking fails
//
//nolint:gocognit,gocyclo,cyclop,funlen // Complex but necessary: SNOD split + heap expansion + B-tree update
func (fw *FileWriter) linkToParent(parentPath, childName string, childAddr uint64) error {
	// Get parent group metadata.
	var heapAddr, btreeAddr uint64
	if parentPath == "" || parentPath == "/" {
		heapAddr = fw.rootHeapAddr
		btreeAddr = fw.rootBTreeAddr
	} else {
		meta, exists := fw.groups[parentPath]
		if !exists {
			return fmt.Errorf("parent group %q not found (create it first)", parentPath)
		}
		heapAddr = meta.heapAddr
		btreeAddr = meta.btreeAddr
	}

	// Step 1: Read existing local heap.
	heap, err := fw.readLocalHeap(heapAddr)
	if err != nil {
		return fmt.Errorf("read local heap: %w", err)
	}

	// Step 2: Add child name to heap. If full, expand.
	nameOffset, err := heap.AddString(childName)
	if err != nil {
		heap, heapAddr, nameOffset, err = fw.expandHeapAndAdd(heap, heapAddr, parentPath, childName)
		if err != nil {
			return err
		}
	}

	// Step 3: Read ALL SNODs in this group (the B-tree may have multiple children after splits).
	btreeNode, snodAddrs, err := fw.readGroupBTree(btreeAddr)
	if err != nil {
		return fmt.Errorf("read group B-tree: %w", err)
	}

	// Collect all entries from all SNODs, plus the new entry.
	allEntries := make([]structures.SymbolTableEntry, 0, snodCapacity)
	for _, addr := range snodAddrs {
		sn, readErr := fw.readSymbolTableNode(addr)
		if readErr != nil {
			return fmt.Errorf("read SNOD at 0x%X: %w", addr, readErr)
		}
		allEntries = append(allEntries, sn.Entries...)
	}

	// Add the new entry.
	newEntry := structures.SymbolTableEntry{
		LinkNameOffset: nameOffset,
		ObjectAddress:  childAddr,
		CacheType:      0,
		Reserved:       0,
	}
	allEntries = append(allEntries, newEntry)

	// Sort all entries by name (HDF5 format requirement).
	fw.sortEntriesByName(allEntries, heap, nameOffset, childName)

	// Step 4: Distribute entries across SNODs.
	// Each SNOD holds at most snodCapacity (8) entries.
	// Per C reference (H5Gnode.c:613): split at K (4), each half gets K entries.
	numSNODs := (len(allEntries) + snodCapacity - 1) / snodCapacity
	if numSNODs < 1 {
		numSNODs = 1
	}

	// Ensure we have enough SNOD addresses (allocate new ones if needed).
	for len(snodAddrs) < numSNODs {
		newAddr, allocErr := fw.writer.Allocate(snodTotalSize)
		if allocErr != nil {
			return fmt.Errorf("allocate new SNOD: %w", allocErr)
		}
		snodAddrs = append(snodAddrs, newAddr)
	}

	offsetSize := fw.file.sb.OffsetSize

	// Step 4: Rebuild and write B-tree FIRST (before SNODs).
	// For v0 format with fixed addresses, the B-tree write must complete before SNOD writes
	// to avoid overwriting SNOD data with B-tree zero padding.
	const groupBTreeK = 16 // B-tree internal node K (separate from GroupLeafNodeK).

	// Build B-tree keys for SNODs
	snodEntries := make([]btreeEntry, numSNODs)
	for i := 0; i < numSNODs; i++ {
		var leftKey uint64
		if i > 0 {
			prevEnd := i * snodCapacity
			if prevEnd > len(allEntries) {
				prevEnd = len(allEntries)
			}
			leftKey = allEntries[prevEnd-1].LinkNameOffset
		}
		snodEntries[i] = btreeEntry{key: leftKey, childAddr: snodAddrs[i]}
	}
	if len(allEntries) > 0 {
		snodEntries = append(snodEntries, btreeEntry{key: allEntries[len(allEntries)-1].LinkNameOffset, childAddr: 0})
	}

	// Build multi-level B-tree
	newBTreeAddr, err := fw.buildGroupBTree(snodEntries, snodAddrs, groupBTreeK, offsetSize, btreeAddr)
	if err != nil {
		return fmt.Errorf("build group B-tree: %w", err)
	}

	// Update parent's stNodeAddr to the first SNOD (it may have moved).
	if len(btreeNode.ChildPointers) == 0 || snodAddrs[0] != btreeNode.ChildPointers[0] {
		fw.updateGroupStNodeAddr(parentPath, snodAddrs[0])
	}

	// Update B-tree address if it changed (multi-level B-tree)
	if newBTreeAddr != btreeAddr {
		var headerAddr uint64
		if parentPath == "" || parentPath == "/" {
			headerAddr = fw.rootGroupAddr
			fw.rootBTreeAddr = newBTreeAddr
		} else {
			meta, exists := fw.groups[parentPath]
			if !exists {
				return fmt.Errorf("group %q not found", parentPath)
			}
			headerAddr = meta.headerAddr
			meta.btreeAddr = newBTreeAddr
		}
		if err := fw.rewriteSymbolTableMessage(headerAddr, btreeAddr, newBTreeAddr, heapAddr); err != nil {
			return fmt.Errorf("rewrite symbol table message: %w", err)
		}
	}

	// Step 5: Write entries to SNODs (after B-tree to avoid overlap in v0 fixed layout).
	pos := 0
	for i := 0; i < numSNODs; i++ {
		end := pos + snodCapacity
		if end > len(allEntries) {
			end = len(allEntries)
		}
		chunk := allEntries[pos:end]
		pos = end

		sn := structures.NewSymbolTableNode(snodCapacity)
		for _, e := range chunk {
			if addErr := sn.AddEntry(e); addErr != nil {
				return fmt.Errorf("add entry to SNOD %d: %w", i, addErr)
			}
		}
		if writeErr := sn.WriteAt(fw.writer, snodAddrs[i], offsetSize, snodCapacity, fw.file.sb.Endianness); writeErr != nil {
			return fmt.Errorf("write SNOD %d: %w", i, writeErr)
		}
	}

	// Step 6: Write updated heap.
	if err := heap.WriteTo(fw.writer, heapAddr, uint8(offsetSize), uint8(offsetSize)); err != nil { //nolint:gosec // Safe: offsetSize is 8
		return fmt.Errorf("write heap: %w", err)
	}

	return nil
}

// sortEntriesByName sorts symbol table entries by their name from the local heap.
// The new entry (at nameOffset) uses childName directly since the heap data
// may not have been flushed yet.
func (fw *FileWriter) sortEntriesByName(entries []structures.SymbolTableEntry, heap *structures.LocalHeap, nameOffset uint64, childName string) {
	sort.Slice(entries, func(i, j int) bool {
		si := fw.resolveEntryName(entries[i], heap, nameOffset, childName)
		sj := fw.resolveEntryName(entries[j], heap, nameOffset, childName)
		return si < sj
	})
}

// resolveEntryName gets the string name for a symbol table entry from the heap.
func (fw *FileWriter) resolveEntryName(entry structures.SymbolTableEntry, heap *structures.LocalHeap, nameOffset uint64, childName string) string {
	if entry.LinkNameOffset == nameOffset {
		return childName
	}
	name, err := heap.GetString(entry.LinkNameOffset)
	if err != nil {
		return ""
	}
	return name
}

// readGroupBTree reads the B-tree v1 node at the given address and extracts child SNOD addresses.
// Returns the B-tree node, the list of SNOD addresses, and any error.
func (fw *FileWriter) readGroupBTree(btreeAddr uint64) (*structures.BTreeNodeV1, []uint64, error) {
	offsetSize := fw.file.sb.OffsetSize
	endianness := fw.file.sb.Endianness

	// Read B-tree header: 4 (sig) + 1 (type) + 1 (level) + 2 (entries) + 2*offsetSize (siblings).
	headerSize := 8 + 2*int(offsetSize)
	header := make([]byte, headerSize)
	//nolint:gosec // G115: HDF5 addresses fit in int64 for io.ReaderAt interface.
	if _, err := fw.writer.ReadAt(header, int64(btreeAddr)); err != nil {
		return nil, nil, fmt.Errorf("read B-tree header: %w", err)
	}

	sig := string(header[0:4])
	if sig != "TREE" { //nolint:goconst // HDF5 B-tree signature used across multiple packages
		return nil, nil, fmt.Errorf("invalid B-tree signature: %q", sig)
	}

	nodeLevel := header[5]
	entriesUsed := endianness.Uint16(header[6:8])

	// Read keys and children (interleaved).
	// Layout after header: Key[0], Child[0], Key[1], Child[1], ..., Key[N].
	// Total data: (entriesUsed+1) keys + entriesUsed children.
	dataSize := (int(entriesUsed)+1)*int(offsetSize) + int(entriesUsed)*int(offsetSize)
	data := make([]byte, dataSize)
	//nolint:gosec // G115: HDF5 addresses fit in int64 for io.ReaderAt interface.
	if _, err := fw.writer.ReadAt(data, int64(btreeAddr)+int64(headerSize)); err != nil {
		return nil, nil, fmt.Errorf("read B-tree data: %w", err)
	}

	node := &structures.BTreeNodeV1{
		Signature:     [4]byte{'T', 'R', 'E', 'E'},
		NodeType:      header[4],
		NodeLevel:     nodeLevel,
		EntriesUsed:   entriesUsed,
		LeftSibling:   0xFFFFFFFFFFFFFFFF,
		RightSibling:  0xFFFFFFFFFFFFFFFF,
		Keys:          make([]uint64, 0, entriesUsed+1),
		ChildPointers: make([]uint64, 0, entriesUsed),
	}

	pos := 0
	var snodAddrs []uint64
	for i := uint16(0); i < entriesUsed; i++ {
		key := readAddrFromBuf(data[pos:], int(offsetSize), endianness)
		pos += int(offsetSize)
		child := readAddrFromBuf(data[pos:], int(offsetSize), endianness)
		pos += int(offsetSize)

		node.Keys = append(node.Keys, key)
		node.ChildPointers = append(node.ChildPointers, child)

		if child != 0 && child != 0xFFFFFFFFFFFFFFFF {
			if nodeLevel == 0 {
				snodAddrs = append(snodAddrs, child)
			} else {
				_, childSNODs, readErr := fw.readGroupBTree(child)
				if readErr != nil {
					return nil, nil, fmt.Errorf("read B-tree child node: %w", readErr)
				}
				snodAddrs = append(snodAddrs, childSNODs...)
			}
		}
	}
	// Read final key.
	if pos+int(offsetSize) <= len(data) {
		finalKey := readAddrFromBuf(data[pos:], int(offsetSize), endianness)
		node.Keys = append(node.Keys, finalKey)
	}

	return node, snodAddrs, nil
}

// buildGroupBTree builds a multi-level B-tree for group symbol tables.
// entries: list of (key, childAddr) pairs for leaf level
// btreeAddr: existing B-tree address (for single-level case)
// Returns the root B-tree node address.
func (fw *FileWriter) buildGroupBTree(entries []btreeEntry, snodAddrs []uint64, k uint16, offsetSize uint8, btreeAddr uint64) (uint64, error) {
	maxChildren := 2 * int(k)
	maxKeys := maxChildren + 1

	if len(snodAddrs) <= maxChildren {
		btree := structures.NewBTreeNodeV1(0, k)
		btree.NodeLevel = 0
		for _, entry := range entries {
			if entry.childAddr != 0 {
				if err := btree.AddKey(entry.key, entry.childAddr); err != nil {
					return 0, err
				}
			}
		}
		if len(entries) > 0 && entries[len(entries)-1].childAddr == 0 {
			btree.Keys = append(btree.Keys, entries[len(entries)-1].key)
		}
		if err := btree.WriteAt(fw.writer, btreeAddr, offsetSize, k, fw.file.sb.Endianness); err != nil {
			return 0, err
		}
		return btreeAddr, nil
	}

	currentLevelAddrs := snodAddrs
	currentLevelEntries := entries
	currentLevel := uint8(0)

	for len(currentLevelAddrs) > maxChildren {
		numNodes := (len(currentLevelAddrs) + maxChildren - 1) / maxChildren
		nextLevelAddrs := make([]uint64, 0, numNodes)
		nextLevelEntries := make([]btreeEntry, 0, numNodes+1)

		for i := 0; i < numNodes; i++ {
			start := i * maxChildren
			end := start + maxChildren
			if end > len(currentLevelAddrs) {
				end = len(currentLevelAddrs)
			}

			node := structures.NewBTreeNodeV1(0, k)
			node.NodeLevel = currentLevel

			for j := start; j < end; j++ {
				keyIdx := j
				if keyIdx >= len(currentLevelEntries) {
					keyIdx = len(currentLevelEntries) - 1
				}
				node.AddKey(currentLevelEntries[keyIdx].key, currentLevelAddrs[j])
			}

			if end < len(currentLevelEntries) {
				node.Keys = append(node.Keys, currentLevelEntries[end].key)
			}

			nodeAddr, err := fw.writer.Allocate(uint64(8 + 2*int(offsetSize) + maxKeys*int(offsetSize) + maxChildren*int(offsetSize)))
			if err != nil {
				return 0, err
			}

			if err := node.WriteAt(fw.writer, nodeAddr, offsetSize, k, fw.file.sb.Endianness); err != nil {
				return 0, err
			}

			nextLevelAddrs = append(nextLevelAddrs, nodeAddr)
			if i > 0 && start < len(currentLevelEntries) {
				nextLevelEntries = append(nextLevelEntries, btreeEntry{key: currentLevelEntries[start].key, childAddr: nodeAddr})
			} else if i == 0 {
				nextLevelEntries = append(nextLevelEntries, btreeEntry{key: 0, childAddr: nodeAddr})
			}
		}

		if len(currentLevelEntries) > 0 {
			nextLevelEntries = append(nextLevelEntries, btreeEntry{key: currentLevelEntries[len(currentLevelEntries)-1].key, childAddr: 0})
		}

		currentLevelAddrs = nextLevelAddrs
		currentLevelEntries = nextLevelEntries
		currentLevel++
	}

	if len(currentLevelAddrs) == 1 {
		return currentLevelAddrs[0], nil
	}

	node := structures.NewBTreeNodeV1(0, k)
	node.NodeLevel = currentLevel
	for _, entry := range currentLevelEntries {
		if entry.childAddr != 0 {
			node.AddKey(entry.key, entry.childAddr)
		}
	}
	if len(currentLevelEntries) > 0 && currentLevelEntries[len(currentLevelEntries)-1].childAddr == 0 {
		node.Keys = append(node.Keys, currentLevelEntries[len(currentLevelEntries)-1].key)
	}

	nodeAddr, err := fw.writer.Allocate(uint64(8 + 2*int(offsetSize) + maxKeys*int(offsetSize) + maxChildren*int(offsetSize)))
	if err != nil {
		return 0, err
	}
	if err := node.WriteAt(fw.writer, nodeAddr, offsetSize, k, fw.file.sb.Endianness); err != nil {
		return 0, err
	}
	return nodeAddr, nil
}

// readAddrFromBuf reads a variable-sized address from a byte buffer.
func readAddrFromBuf(data []byte, size int, endianness binary.ByteOrder) uint64 {
	if size > len(data) {
		size = len(data)
	}
	switch size {
	case 2:
		return uint64(endianness.Uint16(data[:2]))
	case 4:
		return uint64(endianness.Uint32(data[:4]))
	case 8:
		return endianness.Uint64(data[:8])
	default:
		return uint64(data[0])
	}
}

// updateGroupHeapAddr updates the heap address for a group.
// This also rewrites the group's object header symbol table message to point to the new heap.
func (fw *FileWriter) updateGroupHeapAddr(parentPath string, newHeapAddr uint64) error {
	if parentPath == "" || parentPath == "/" {
		fw.rootHeapAddr = newHeapAddr
		return fw.rewriteSymbolTableMessage(fw.rootGroupAddr, fw.rootBTreeAddr, fw.rootBTreeAddr, newHeapAddr)
	}
	meta, exists := fw.groups[parentPath]
	if !exists {
		return fmt.Errorf("group %q not found", parentPath)
	}
	meta.heapAddr = newHeapAddr
	return fw.rewriteSymbolTableMessage(meta.headerAddr, meta.btreeAddr, meta.btreeAddr, newHeapAddr)
}

// expandHeapAndAdd expands the local heap (doubles its size) and adds a string.
// Returns the new heap, new address, string offset, and any error.
func (fw *FileWriter) expandHeapAndAdd(heap *structures.LocalHeap, _ uint64, parentPath, childName string) (*structures.LocalHeap, uint64, uint64, error) {
	newSize := heap.DataSegmentSize * 2
	newHeap := structures.NewLocalHeap(newSize)
	if err := newHeap.CopyStringsFrom(heap); err != nil {
		return nil, 0, 0, fmt.Errorf("copy strings to new heap: %w", err)
	}
	nameOffset, err := newHeap.AddString(childName)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("add string to expanded heap: %w", err)
	}
	newHeapAddr, err := fw.writer.Allocate(newHeap.Size())
	if err != nil {
		return nil, 0, 0, fmt.Errorf("allocate expanded heap: %w", err)
	}
	if err := fw.updateGroupHeapAddr(parentPath, newHeapAddr); err != nil {
		return nil, 0, 0, fmt.Errorf("update heap address: %w", err)
	}
	return newHeap, newHeapAddr, nameOffset, nil
}

// updateGroupStNodeAddr updates the primary SNOD address for a group.
func (fw *FileWriter) updateGroupStNodeAddr(parentPath string, newStNodeAddr uint64) {
	if parentPath == "" || parentPath == "/" {
		fw.rootStNodeAddr = newStNodeAddr
	} else if meta, exists := fw.groups[parentPath]; exists {
		meta.stNodeAddr = newStNodeAddr
	}
}

// rewriteSymbolTableMessage rewrites the symbol table message in an object header.
// This is needed when the heap or B-tree address changes.
// oldBTreeAddr: the old B-tree address to search for
// newBTreeAddr: the new B-tree address to write
// newHeapAddr: the new heap address to write
func (fw *FileWriter) rewriteSymbolTableMessage(headerAddr, _, newBTreeAddr, newHeapAddr uint64) error {
	stMsg := core.EncodeSymbolTableMessage(newBTreeAddr, newHeapAddr, int(fw.file.sb.OffsetSize), int(fw.file.sb.LengthSize))

	header, err := core.ReadObjectHeader(fw.writer, headerAddr, fw.file.sb)
	if err != nil {
		return fmt.Errorf("read object header: %w", err)
	}

	for _, msg := range header.Messages {
		if msg.Type == core.MsgSymbolTable {
			msgDataOffset := msg.Offset + 4
			if _, writeErr := fw.writer.WriteAt(stMsg, int64(msgDataOffset)); writeErr != nil {
				return fmt.Errorf("rewrite symbol table message: %w", writeErr)
			}

			var headerAllocSz uint64
			if headerAddr == fw.rootGroupAddr {
				headerAllocSz = fw.rootHeaderAllocSz
			} else {
				for _, meta := range fw.groups {
					if meta.headerAddr == headerAddr {
						headerAllocSz = meta.headerAllocSz
						break
					}
				}
			}
			if headerAllocSz == 0 {
				headerAllocSz = uint64(512)
			}

			checksumSize := uint64(4)
			dataLen := headerAllocSz - checksumSize
			ohdrBuf := make([]byte, dataLen)
			if _, readErr := fw.writer.ReadAt(ohdrBuf, int64(headerAddr)); readErr != nil {
				return fmt.Errorf("failed to read object header for checksum: %w", readErr)
			}
			newChecksum := core.JenkinsChecksum(ohdrBuf)
			var csumBuf [4]byte
			binary.LittleEndian.PutUint32(csumBuf[:], newChecksum)
			if _, writeErr := fw.writer.WriteAt(csumBuf[:], int64(headerAddr)+int64(dataLen)); writeErr != nil {
				return fmt.Errorf("failed to write object header checksum: %w", writeErr)
			}

			return nil
		}
	}

	return fmt.Errorf("symbol table message not found in object header at 0x%X", headerAddr)
}

// readLocalHeap reads a local heap from the file at the specified address.
// This is used to modify the heap by adding new strings for linking.
//
// Parameters:
//   - addr: Address of the local heap in the file
//
// Returns:
//   - *structures.LocalHeap: The heap structure (writable)
//   - error: If read fails
func (fw *FileWriter) readLocalHeap(addr uint64) (*structures.LocalHeap, error) {
	// Load existing heap from disk
	heap, err := structures.LoadLocalHeap(fw.writer, addr, fw.file.sb)
	if err != nil {
		return nil, fmt.Errorf("load local heap: %w", err)
	}

	// Convert to writable mode (copies Data to internal strings buffer)
	if err := heap.PrepareForModification(); err != nil {
		return nil, fmt.Errorf("prepare heap for modification: %w", err)
	}

	// Set write-mode fields
	// Note: DataSegmentAddress is set by WriteTo(), not here
	heap.OffsetToHeadFreeList = 1 // MVP: no free list (1 = H5HL_FREE_NULL)

	return heap, nil
}

// readSymbolTableNode reads a symbol table node from the file at the specified address.
// This is used to modify the node by adding new entries for linking.
//
// Parameters:
//   - addr: Address of the symbol table node in the file
//
// Returns:
//   - *structures.SymbolTableNode: The node structure (writable)
//   - error: If read fails
func (fw *FileWriter) readSymbolTableNode(addr uint64) (*structures.SymbolTableNode, error) {
	// Use the existing ParseSymbolTableNode function from structures package
	return structures.ParseSymbolTableNode(fw.writer, addr, fw.file.sb)
}

// CreateDenseGroup creates new dense group (HDF5 1.8+ format).
//
// Dense groups are more efficient for large numbers of links (>8).
// They use fractal heap + B-tree v2 instead of symbol table.
//
// Parameters:
//   - name: Group name (must start with "/")
//   - links: Map of link_name → target_path
//
// Returns:
//   - error: Non-nil if creation fails
//
// Example:
//
//	err := fw.CreateDenseGroup("/large_group", map[string]string{
//	    "dataset1": "/data/dataset1",
//	    "dataset2": "/data/dataset2",
//	    // ... many links
//	})
//
// Reference: H5Gcreate.c - H5Gcreate2().
func (fw *FileWriter) CreateDenseGroup(name string, links map[string]string) error {
	// Validate name
	if !strings.HasPrefix(name, "/") {
		return fmt.Errorf("group name must start with /: %s", name)
	}

	// Create DenseGroupWriter
	dgw := writer.NewDenseGroupWriter(name)

	// Add all links
	for linkName, targetPath := range links {
		// Resolve target path to object header address
		targetAddr, err := fw.resolveObjectAddress(targetPath)
		if err != nil {
			return fmt.Errorf("failed to resolve target %s: %w", targetPath, err)
		}

		err = dgw.AddLink(linkName, targetAddr)
		if err != nil {
			return fmt.Errorf("failed to add link %s: %w", linkName, err)
		}
	}

	// Write dense group
	ohAddr, err := dgw.WriteToFile(fw.writer, fw.writer.Allocator(), fw.file.sb)
	if err != nil {
		return fmt.Errorf("failed to write dense group: %w", err)
	}

	// Link to parent
	parent, childName := parsePath(name)

	// Validate parent exists (if not root)
	if parent != "" && parent != "/" {
		if _, exists := fw.groups[parent]; !exists {
			return fmt.Errorf("parent group %q does not exist (create it first)", parent)
		}
	}

	if err := fw.linkToParent(parent, childName, ohAddr); err != nil {
		return fmt.Errorf("failed to link to parent: %w", err)
	}

	return nil
}

// resolveObjectAddress resolves object path to file address.
// Searches all SNODs in the parent group's B-tree to find the named object.
//
// Parameters:
//   - path: Object path (e.g., "/data/dataset1" or "/dataset1")
//
// Returns:
//   - uint64: File address of object header
//   - error: Non-nil if object not found or parent doesn't exist
func (fw *FileWriter) resolveObjectAddress(path string) (uint64, error) {
	if path == "/" {
		return fw.rootGroupAddr, nil
	}
	if !strings.HasPrefix(path, "/") {
		return 0, fmt.Errorf("path must start with /: %s", path)
	}

	parent, name := parsePath(path)

	// Get parent B-tree and heap addresses.
	var btreeAddr, heapAddr uint64
	if parent == "" || parent == "/" {
		btreeAddr = fw.rootBTreeAddr
		heapAddr = fw.rootHeapAddr
	} else {
		meta, exists := fw.groups[parent]
		if !exists {
			return 0, fmt.Errorf("parent group %q not found", parent)
		}
		btreeAddr = meta.btreeAddr
		heapAddr = meta.heapAddr
	}

	// Read all SNODs from B-tree.
	_, snodAddrs, err := fw.readGroupBTree(btreeAddr)
	if err != nil {
		return 0, fmt.Errorf("read group B-tree: %w", err)
	}

	// Read heap.
	heap, err := fw.readLocalHeap(heapAddr)
	if err != nil {
		return 0, fmt.Errorf("read local heap: %w", err)
	}

	// Search all SNODs for the named object.
	for _, snodAddr := range snodAddrs {
		stNode, readErr := fw.readSymbolTableNode(snodAddr)
		if readErr != nil {
			continue
		}
		for _, entry := range stNode.Entries {
			linkName, nameErr := heap.GetString(entry.LinkNameOffset)
			if nameErr != nil {
				continue
			}
			if linkName == name {
				return entry.ObjectAddress, nil
			}
		}
	}

	return 0, fmt.Errorf("object not found: %s", path)
}

// unlinkFromParent removes a named child entry from its parent group's symbol table.
// This is the reverse of linkToParent. It reads the parent's B-tree + SNODs, finds
// the named entry, removes it, and rewrites the B-tree + SNODs.
//
// Parameters:
//   - parentPath: Path to parent group ("" or "/" for root)
//   - childName: Name of the child object to unlink
//
// Returns:
//   - uint64: The ObjectAddress of the removed entry (for cascade delete)
//   - error: If unlinking fails or child not found
//
// Reference: H5Gobj.c - H5G_obj_remove(), H5Gnode.c - H5G__node_remove().
//
//nolint:gocognit,gocyclo,cyclop,funlen,nestif // Complex but necessary: mirror of linkToParent with entry removal
func (fw *FileWriter) unlinkFromParent(parentPath, childName string) (uint64, error) {
	// Get parent group metadata (same as linkToParent lines 308-319).
	var heapAddr, btreeAddr uint64
	if parentPath == "" || parentPath == "/" {
		heapAddr = fw.rootHeapAddr
		btreeAddr = fw.rootBTreeAddr
	} else {
		meta, exists := fw.groups[parentPath]
		if !exists {
			return 0, fmt.Errorf("parent group %q not found", parentPath)
		}
		heapAddr = meta.heapAddr
		btreeAddr = meta.btreeAddr
	}

	// Step 1: Read existing local heap.
	heap, err := fw.readLocalHeap(heapAddr)
	if err != nil {
		return 0, fmt.Errorf("read local heap: %w", err)
	}

	// Step 2: Read ALL SNODs from B-tree.
	_, snodAddrs, err := fw.readGroupBTree(btreeAddr)
	if err != nil {
		return 0, fmt.Errorf("read group B-tree: %w", err)
	}

	// Step 3: Collect all entries from all SNODs.
	allEntries := make([]structures.SymbolTableEntry, 0, snodCapacity)
	for _, addr := range snodAddrs {
		sn, readErr := fw.readSymbolTableNode(addr)
		if readErr != nil {
			return 0, fmt.Errorf("read SNOD at 0x%X: %w", addr, readErr)
		}
		allEntries = append(allEntries, sn.Entries...)
	}

	// Step 4: Find and remove the entry by name.
	foundIdx := -1
	var removedAddr uint64
	for i, entry := range allEntries {
		entryName, nameErr := heap.GetString(entry.LinkNameOffset)
		if nameErr != nil {
			continue
		}
		if entryName == childName {
			foundIdx = i
			removedAddr = entry.ObjectAddress
			break
		}
	}

	if foundIdx == -1 {
		return 0, fmt.Errorf("child %q not found in parent group", childName)
	}

	// Remove the entry from the slice.
	allEntries = append(allEntries[:foundIdx], allEntries[foundIdx+1:]...)

	// Step 5: Rebuild and write B-tree + SNODs with the remaining entries.
	// If no entries remain, still write an empty SNOD (HDF5 groups always have at least one SNOD).
	numSNODs := 1
	if len(allEntries) > 0 {
		numSNODs = (len(allEntries) + snodCapacity - 1) / snodCapacity
	}

	// Reuse existing SNOD addresses where possible.
	for len(snodAddrs) < numSNODs {
		newAddr, allocErr := fw.writer.Allocate(snodTotalSize)
		if allocErr != nil {
			return 0, fmt.Errorf("allocate new SNOD: %w", allocErr)
		}
		snodAddrs = append(snodAddrs, newAddr)
	}
	// Only use as many as needed.
	snodAddrs = snodAddrs[:numSNODs]

	offsetSize := fw.file.sb.OffsetSize

	// Rebuild B-tree.
	const groupBTreeK = 16
	newBTree := structures.NewBTreeNodeV1(0, groupBTreeK)

	if len(allEntries) > 0 {
		for i := 0; i < numSNODs; i++ {
			var leftKey uint64
			if i > 0 {
				prevEnd := i * snodCapacity
				if prevEnd > len(allEntries) {
					prevEnd = len(allEntries)
				}
				leftKey = allEntries[prevEnd-1].LinkNameOffset
			}
			if addErr := newBTree.AddKey(leftKey, snodAddrs[i]); addErr != nil {
				return 0, fmt.Errorf("add B-tree key for SNOD %d: %w", i, addErr)
			}
		}
		// Final right key = last entry's name offset.
		lastEntry := allEntries[len(allEntries)-1]
		newBTree.Keys = append(newBTree.Keys, lastEntry.LinkNameOffset)
	} else {
		// Empty group: single SNOD with key 0.
		if addErr := newBTree.AddKey(0, snodAddrs[0]); addErr != nil {
			return 0, fmt.Errorf("add B-tree key for empty SNOD: %w", addErr)
		}
		newBTree.Keys = append(newBTree.Keys, uint64(0))
	}

	// Write B-tree.
	if err := newBTree.WriteAt(fw.writer, btreeAddr, offsetSize, groupBTreeK, fw.file.sb.Endianness); err != nil {
		return 0, fmt.Errorf("write B-tree: %w", err)
	}

	// Write entries to SNODs.
	pos := 0
	for i := 0; i < numSNODs; i++ {
		end := pos + snodCapacity
		if end > len(allEntries) {
			end = len(allEntries)
		}
		chunk := allEntries[pos:end]
		pos = end

		sn := structures.NewSymbolTableNode(snodCapacity)
		for _, e := range chunk {
			if addErr := sn.AddEntry(e); addErr != nil {
				return 0, fmt.Errorf("add entry to SNOD %d: %w", i, addErr)
			}
		}
		if writeErr := sn.WriteAt(fw.writer, snodAddrs[i], offsetSize, snodCapacity, fw.file.sb.Endianness); writeErr != nil {
			return 0, fmt.Errorf("write SNOD %d: %w", i, writeErr)
		}
	}

	// Write updated heap.
	if err := heap.WriteTo(fw.writer, heapAddr, offsetSize, offsetSize); err != nil {
		return 0, fmt.Errorf("write heap: %w", err)
	}

	return removedAddr, nil
}

// Dense group threshold (HDF5 default: switch to dense when >8 links).
const denseGroupThreshold = 8

// CreateGroupWithLinks creates group with automatic format selection.
//
// This method automatically chooses the most efficient storage format:
//   - Symbol table (old format) for ≤8 links (compact)
//   - Dense format (new format) for >8 links (scalable)
//
// This matches HDF5 1.8+ behavior: start compact, use dense when needed.
//
// Parameters:
//   - name: Group name (must start with "/")
//   - links: Map of link_name → target_path (can be empty)
//
// Returns:
//   - error: Non-nil if creation fails
//
// Example:
//
//	// Small group (will use symbol table)
//	fw.CreateGroupWithLinks("/small", map[string]string{
//	    "data1": "/dataset1",
//	    "data2": "/dataset2",
//	})
//
//	// Large group (will use dense format)
//	largeLinks := make(map[string]string)
//	for i := 0; i < 100; i++ {
//	    largeLinks[fmt.Sprintf("link%d", i)] = fmt.Sprintf("/dataset%d", i)
//	}
//	fw.CreateGroupWithLinks("/large", largeLinks)
//
// Reference: H5Gint.c - H5G_convert_to_dense().
func (fw *FileWriter) CreateGroupWithLinks(name string, links map[string]string) error {
	if len(links) > denseGroupThreshold {
		// Use dense format for large groups
		return fw.CreateDenseGroup(name, links)
	}

	// Use symbol table format for small groups
	// Create empty group first
	_, err := fw.CreateGroup(name)
	if err != nil {
		return err
	}

	// For MVP: linking is handled by CreateDenseGroup for dense groups
	// For symbol table groups, links would need to be added via linkToParent
	// This is a limitation of the MVP - symbol table groups can be created empty,
	// but adding links after creation requires manual linkToParent calls

	// Future: implement addLinkToGroup() to add links to existing symbol table groups

	if len(links) > 0 {
		return fmt.Errorf("adding links to symbol table groups not yet supported in MVP (group %s has %d links)", name, len(links))
	}

	return nil
}
