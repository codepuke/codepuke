// Go: []int{1, 2, 3}
// A top-level slice carries no struct schema, so name the element type.
const bytes = encode([1n, 2n, 3n], { elemType: GOB_INT });

const numbers = decode<bigint[]>(bytes); // [1n, 2n, 3n]
