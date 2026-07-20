package main

import (
	"fmt"

	hdf5 "github.com/huangzhengshun/hdf5"
)

func main() {
	testCases := []int{32, 33, 64, 100, 500}
	
	for _, count := range testCases {
		fmt.Printf("\n=== Testing with %d datasets ===\n", count)
		filename := fmt.Sprintf("test_btree_%d.h5", count)
		
		f, err := hdf5.CreateForWrite(filename, hdf5.CreateTruncate)
		if err != nil {
			fmt.Printf("Failed to create file: %v\n", err)
			return
		}
		
		for i := 0; i < count; i++ {
			datasetPath := fmt.Sprintf("/dataset_%04d", i)
			_, err := f.CreateDataset(datasetPath, hdf5.Float64, []uint64{2})
			if err != nil {
				fmt.Printf("Failed to create dataset %d/%d: %v\n", i, count, err)
				f.Close()
				return
			}
		}
		
		if err := f.Close(); err != nil {
			fmt.Printf("Failed to close file: %v\n", err)
			return
		}
		
		fmt.Printf("Created file %s with %d datasets successfully\n", filename, count)
	}
	
	fmt.Println("\nAll test files created!")
}
