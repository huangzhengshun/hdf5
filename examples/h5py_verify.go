package main

import (
	"fmt"
	"os"

	"github.com/huangzhengshun/hdf5"
)

func main() {
	testFile := "h5py_verify_test.h5"
	os.Remove(testFile)

	fw, err := hdf5.CreateForWrite(testFile, hdf5.CreateTruncate)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer fw.Close()

	fmt.Println("Creating test datasets...")

	ds1, err := fw.CreateDataset("/int32_data", hdf5.Int32, []uint64{10})
	if err != nil {
		fmt.Printf("Failed to create int32 dataset: %v\n", err)
		return
	}
	err = ds1.Write([]int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	if err != nil {
		fmt.Printf("Failed to write int32 data: %v\n", err)
		return
	}

	ds2, err := fw.CreateDataset("/float64_data", hdf5.Float64, []uint64{5})
	if err != nil {
		fmt.Printf("Failed to create float64 dataset: %v\n", err)
		return
	}
	err = ds2.Write([]float64{1.1, 2.2, 3.3, 4.4, 5.5})
	if err != nil {
		fmt.Printf("Failed to write float64 data: %v\n", err)
		return
	}

	ds3, err := fw.CreateDataset("/uint8_data", hdf5.Uint8, []uint64{20})
	if err != nil {
		fmt.Printf("Failed to create uint8 dataset: %v\n", err)
		return
	}
	data8 := make([]uint8, 20)
	for i := range data8 {
		data8[i] = uint8(i * 10)
	}
	err = ds3.Write(data8)
	if err != nil {
		fmt.Printf("Failed to write uint8 data: %v\n", err)
		return
	}

	ds4, err := fw.CreateDataset("/string_data", hdf5.String, []uint64{3}, hdf5.WithStringSize(20))
	if err != nil {
		fmt.Printf("Failed to create string dataset: %v\n", err)
		return
	}
	err = ds4.Write([]string{"hello", "world", "test"})
	if err != nil {
		fmt.Printf("Failed to write string data: %v\n", err)
		return
	}

	ds5, err := fw.CreateDataset("/chunked_data", hdf5.Float64, []uint64{100}, hdf5.WithChunkDims([]uint64{10}))
	if err != nil {
		fmt.Printf("Failed to create chunked dataset: %v\n", err)
		return
	}
	chunkData := make([]float64, 100)
	for i := range chunkData {
		chunkData[i] = float64(i) * 0.5
	}
	err = ds5.Write(chunkData)
	if err != nil {
		fmt.Printf("Failed to write chunked data: %v\n", err)
		return
	}

	ds6, err := fw.CreateDataset("/compressed_data", hdf5.Int32, []uint64{1000}, hdf5.WithChunkDims([]uint64{100}), hdf5.WithGZIPCompression(6))
	if err != nil {
		fmt.Printf("Failed to create compressed dataset: %v\n", err)
		return
	}
	compressedData := make([]int32, 1000)
	for i := range compressedData {
		compressedData[i] = int32(i)
	}
	err = ds6.Write(compressedData)
	if err != nil {
		fmt.Printf("Failed to write compressed data: %v\n", err)
		return
	}

	ds8, err := fw.CreateDataset("/shuffled_compressed", hdf5.Int32, []uint64{1000}, hdf5.WithChunkDims([]uint64{100}), hdf5.WithShuffle(), hdf5.WithGZIPCompression(6))
	if err != nil {
		fmt.Printf("Failed to create shuffled+compressed dataset: %v\n", err)
		return
	}
	err = ds8.Write(compressedData)
	if err != nil {
		fmt.Printf("Failed to write shuffled+compressed data: %v\n", err)
		return
	}

	grp, err := fw.CreateGroup("/test_group")
	if err != nil {
		fmt.Printf("Failed to create group: %v\n", err)
		return
	}

	ds7, err := fw.CreateDataset("/test_group/nested_data", hdf5.Int64, []uint64{8})
	if err != nil {
		fmt.Printf("Failed to create nested dataset: %v\n", err)
		return
	}
	err = ds7.Write([]int64{100, 200, 300, 400, 500, 600, 700, 800})
	if err != nil {
		fmt.Printf("Failed to write nested data: %v\n", err)
		return
	}

	err = grp.WriteAttribute("group_attr", "test attribute")
	if err != nil {
		fmt.Printf("Failed to write group attribute: %v\n", err)
		return
	}

	err = ds1.WriteAttribute("dataset_attr", int32(42))
	if err != nil {
		fmt.Printf("Failed to write dataset attribute: %v\n", err)
		return
	}

	fmt.Println("Creating many datasets for B-tree test...")
	_, err = fw.CreateGroup("/many_datasets")
	if err != nil {
		fmt.Printf("Failed to create many_datasets group: %v\n", err)
		return
	}
	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("/many_datasets/dataset_%d", i)
		ds, err := fw.CreateDataset(name, hdf5.Float32, []uint64{10})
		if err != nil {
			fmt.Printf("Failed to create dataset %s: %v\n", name, err)
			return
		}
		data := make([]float32, 10)
		for j := range data {
			data[j] = float32(i*10 + j)
		}
		err = ds.Write(data)
		if err != nil {
			fmt.Printf("Failed to write dataset %s: %v\n", name, err)
			return
		}
	}

	fmt.Printf("File created successfully: %s\n", testFile)
	fmt.Println("Use h5py to verify:")
	fmt.Println("  python -c \"import h5py; f = h5py.File('h5py_verify_test.h5', 'r'); print(list(f.keys())); f.close()\"")
}
