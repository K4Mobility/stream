package rb

import "github.com/K4Mobility/stream/quantile/order"

// Tree implements a red-black tree data structure,
// and also satisfies the st.Tree interface,
// as well as the order.Statistic interface.
type Tree struct {
	root *Node
}

// Size returns the size of the tree.
func (t *Tree) Size() int {
	return t.root.Size()
}

// Add inserts a value into the tree.
func (t *Tree) Add(val float64) {
	t.root = t.root.add(val)
}

// Remove deletes a value from the tree.
func (t *Tree) Remove(val float64) {
	t.root = t.root.remove(val)
}

// Select returns the node with the kth smallest value in the tree.
func (t *Tree) Select(k int) order.Node {
	return t.root.Select(k)
}

// Rank returns the number of nodes strictly less than the value.
func (t *Tree) Rank(val float64) int {
	return t.root.Rank(val)
}

// String returns the string representation of the tree.
func (t *Tree) String() string {
	return t.root.TreeString()
}

// Clear resets the tree.
func (t *Tree) Clear() {
	*t = Tree{}
}
