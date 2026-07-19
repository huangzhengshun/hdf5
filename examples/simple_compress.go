package main

import (
	"fmt"
	"os"

	"github.com/huangzhengshun/hdf5"
)

func main() {
	testFile := "simple_compress_test.h5"
	os.Remove(testFile)

	fw, err := hdf5.CreateForWrite(testFile, hdf5.CreateTruncate)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer fw.Close()

	fmt.Println("Creating compressed dataset...")
	ds, err := fw.CreateDataset("/data", hdf5.Int32, []uint64{1000}, hdf5.WithChunkDims([]uint64{100}), hdf5.WithGZIPCompression(6))
	if err != nil {
		fmt.Printf("Failed to create compressed dataset: %v\n", err)
		return
	}

	data := make([]int32, 1000)
	for i := range data {
		data[i] = int32(i)
	}
	err = ds.Write(data)
	if err != nil {
		fmt.Printf("Failed to write data: %v\n", err)
		return
	}

	fmt.Printf("File created: %s\n", testFile)
}