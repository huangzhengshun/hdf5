import h5py
import os

# Test reading HDF5 file with many groups
print("Test: Reading go-hdf5 created file with many groups")
try:
    with h5py.File('test_groups.h5', 'r') as f:
        print(f"  File opened successfully")
        
        # List all keys in root
        root_keys = list(f.keys())
        print(f"  Root group keys: {len(root_keys)}")
        
        # Verify all 100 groups exist
        missing_groups = []
        for i in range(100):
            group_path = f"/group_{i:03d}"
            if group_path not in f:
                missing_groups.append(group_path)
        
        if len(missing_groups) == 0:
            print(f"  All 100 root groups found")
        else:
            print(f"  Missing {len(missing_groups)} groups")
        
        # Verify nested groups
        if '/nested' in f:
            nested_keys = list(f['/nested'].keys())
            print(f"  Nested group keys: {len(nested_keys)}")
            
            missing_nested = []
            for i in range(50):
                group_path = f"/nested/level1_{i:02d}"
                if group_path not in f:
                    missing_nested.append(group_path)
            
            if len(missing_nested) == 0:
                print(f"  All 50 nested level1 groups found")
            else:
                print(f"  Missing {len(missing_nested)} nested groups")
        
        print("  PASSED")
except Exception as e:
    import traceback
    print(f"  FAILED: {e}")
    traceback.print_exc()

# Cleanup
if os.path.exists('test_groups.h5'):
    os.remove('test_groups.h5')

print("\nTest completed!")
