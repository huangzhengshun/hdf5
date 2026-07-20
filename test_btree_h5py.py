import h5py
import os

# Test reading HDF5 file with B-tree splits
print("Test: Reading go-hdf5 created file with B-tree splits")
try:
    with h5py.File('test_btree_split.h5', 'r') as f:
        print(f"  File opened successfully")
        
        # List all keys in root
        root_keys = list(f.keys())
        print(f"  Root group keys: {len(root_keys)}")
        
        # Verify all 500 groups exist
        missing_groups = []
        for i in range(500):
            group_path = f"/dataset_{i:04d}"
            if group_path not in f:
                missing_groups.append(group_path)
        
        if len(missing_groups) == 0:
            print(f"  All 500 groups found")
        else:
            print(f"  Missing {len(missing_groups)} groups:")
            for m in missing_groups[:10]:
                print(f"    {m}")
            if len(missing_groups) > 10:
                print(f"    ... and {len(missing_groups)-10} more")
        
        print("  PASSED")
except Exception as e:
    import traceback
    print(f"  FAILED: {e}")
    traceback.print_exc()

# Cleanup
if os.path.exists('test_btree_split.h5'):
    os.remove('test_btree_split.h5')

print("\nTest completed!")
