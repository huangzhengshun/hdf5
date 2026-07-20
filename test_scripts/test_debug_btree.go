package main

import (
	"fmt"

	hdf5 "github.com/huangzhengshun/hdf5"
)

func main() {
	// Test with exact scenario that might cause the SNOD error
	fmt.Println("=== Debug B-tree test ===")
	
	// Test with 32 datasets (single B-tree node boundary)
	fmt.Println("\n1. Testing with 32 datasets (single B-tree node)")
	testBTree("test_debug_32.h5", 32)
	
	// Test with 33 datasets (triggers multi-level B-tree)
	fmt.Println("\n2. Testing with 33 datasets (triggers multi-level)")
	testBTree("test_debug_33.h5", 33)
	
	// Test with 256 datasets (many SNODs)
	fmt.Println("\n3. Testing with 256 datasets (many SNODs)")
	testBTree("test_debug_256.h5", 256)
	
	// Test with 257 datasets (multi-level B-tree with many SNODs)
	fmt.Println("\n4. Testing with 257 datasets (multi-level with many SNODs)")
	testBTree("test_debug_257.h5", 257)
	
	fmt.Println("\nAll debug tests completed!")
}

func testBTree(filename string, count int) {
	f, err := hdf5.CreateForWrite(filename, hdf5.CreateTruncate)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	
	for i := 0; i < count; i++ {
		datasetPath := fmt.Sprintf("/dataset_%04d", i)
		_, err := f.CreateDataset(datasetPath, hdf5.Float64, []uint64{2})
		if err != nil {
			fmt.Printf("FAILED at dataset %d/%d: %v\n", i, count, err)
			f.Close()
			return
		}
		
		// Print progress at key points
		if i == 0 || i == 31 || i == 32 || i == 63 || i == 64 || i == 127 || i == 128 || i == 255 || i == 256 {
			fmt.Printf("  Created dataset %04d (%d/%d)\n", i, i+1, count)
		}
	}
	
	if err := f.Close(); err != nil {
		fmt.Printf("Failed to close file: %v\n", err)
		return
	}
	
	fmt.Printf("  Created %s with %d datasets successfully\n", filename, count)
}
