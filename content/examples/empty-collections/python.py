empty_slice = pygob.encode([], elem_type=GOB_INT)                    # []int{}
empty_map = pygob.encode({}, key_type=GOB_STRING, elem_type=GOB_INT)  # map[string]int{}
