import h5py
import os

print("Final Test: Reading go-hdf5 created file with 500 groups")
print("=" * 60)

# Test 1: Basic group reading
print("\nTest 1: Basic group reading with 500 groups")
try:
    with h5py.File('test_btree_split.h5', 'r') as f:
        root_keys = list(f.keys())
        print(f"  Root group keys: {len(root_keys)}")
        
        missing_groups = []
        for i in range(500):
            group_path = f"/dataset_{i:04d}"
            if group_path not in f:
                missing_groups.append(group_path)
        
        if len(missing_groups) == 0:
            print(f"  All 500 groups found: PASSED")
        else:
            print(f"  Missing {len(missing_groups)} groups: FAILED")
            print(f"  Missing: {missing_groups[:10]}")
except Exception as e:
    print(f"  FAILED: {e}")

# Test 2: Nested groups
print("\nTest 2: Nested groups")
try:
    with h5py.File('test_groups.h5', 'r') as f:
        root_keys = list(f.keys())
        print(f"  Root group keys: {len(root_keys)}")
        
        if '/nested' in f:
            nested_keys = list(f['/nested'].keys())
            print(f"  Nested group keys: {len(nested_keys)}")
            
            if len(nested_keys) == 50:
                print(f"  All 50 nested groups found: PASSED")
            else:
                print(f"  Expected 50 nested groups, found {len(nested_keys)}: FAILED")
        else:
            print(f"  /nested group not found: FAILED")
except Exception as e:
    print(f"  FAILED: {e}")

# Test 3: Verify no checksum errors
print("\nTest 3: Verify no checksum errors")
try:
    with h5py.File('test_btree_split.h5', 'r') as f:
        # Try to iterate through all groups to trigger potential checksum errors
        group_count = [0]
        def count_groups(name, obj):
            if isinstance(obj, h5py.Group):
                group_count[0] += 1
        
        f.visititems(count_groups)
        print(f"  Total groups visited: {group_count}")
        print(f"  No checksum errors: PASSED")
except Exception as e:
    print(f"  FAILED: {e}")

# Cleanup
if os.path.exists('test_btree_split.h5'):
    os.remove('test_btree_split.h5')
if os.path.exists('test_groups.h5'):
    os.remove('test_groups.h5')

print("\n" + "=" * 60)
print("All tests completed!")
