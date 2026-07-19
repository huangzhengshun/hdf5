package main

import (
	"fmt"

	"github.com/huangzhengshun/hdf5"
)

func main() {
	// Create test file with GZIP compression
	fmt.Println("Creating test_gzip.h5...")
	f, err := hdf5.CreateForWrite("test_gzip.h5", hdf5.CreateTruncate)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer f.Close()

	data := make([]float64, 1000)
	for i := range data {
		data[i] = float64(i)
	}

	dw, err := f.CreateDataset("/data", hdf5.Float64, []uint64{1000},
		hdf5.WithChunkDims([]uint64{100}),
		hdf5.WithGZIPCompression(6),
	)
	if err != nil {
		fmt.Printf("Failed to create dataset: %v\n", err)
		return
	}
	defer dw.Close()

	if err := dw.Write(data); err != nil {
		fmt.Printf("Failed to write data: %v\n", err)
		return
	}
	fmt.Println("Created test_gzip.h5 successfully")

	// Create test file with Shuffle+GZIP
	fmt.Println("\nCreating test_shuffle_gzip.h5...")
	f2, err := hdf5.CreateForWrite("test_shuffle_gzip.h5", hdf5.CreateTruncate)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer f2.Close()

	dw2, err := f2.CreateDataset("/data", hdf5.Float64, []uint64{1000},
		hdf5.WithChunkDims([]uint64{100}),
		hdf5.WithShuffle(),
		hdf5.WithGZIPCompression(6),
	)
	if err != nil {
		fmt.Printf("Failed to create dataset: %v\n", err)
		return
	}
	defer dw2.Close()

	if err := dw2.Write(data); err != nil {
		fmt.Printf("Failed to write data: %v\n", err)
		return
	}
	fmt.Println("Created test_shuffle_gzip.h5 successfully")

	// Create test file with Fletcher32
	fmt.Println("\nCreating test_fletcher32.h5...")
	f3, err := hdf5.CreateForWrite("test_fletcher32.h5", hdf5.CreateTruncate)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer f3.Close()

	dw3, err := f3.CreateDataset("/data", hdf5.Float64, []uint64{1000},
		hdf5.WithChunkDims([]uint64{100}),
		hdf5.WithFletcher32(),
	)
	if err != nil {
		fmt.Printf("Failed to create dataset: %v\n", err)
		return
	}
	defer dw3.Close()

	if err := dw3.Write(data); err != nil {
		fmt.Printf("Failed to write data: %v\n", err)
		return
	}
	fmt.Println("Created test_fletcher32.h5 successfully")

	fmt.Println("\nAll test files created!")
}
