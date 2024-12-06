package avl

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/K4Mobility/stream/quantile/order"
)

// Node represents a node in an AVL tree.
type Node struct {
	left   *Node
	right  *Node
	val    float64
	height int
	size   int
}

func max(x int, y int) int {
	if x > y {
		return x
	}
	return y
}

// NewNode instantiates a Node struct with a a provided value.
func NewNode(val float64) *Node {
	return &Node{
		val:    val,
		height: 0,
		size:   1,
	}
}

// Left returns the left child of the node.
func (n *Node) Left() (order.Node, error) {
	if n == nil {
		return nil, errors.New("tried to retrieve child of nil node")
	}
	return n.left, nil
}

// Right returns the right child of the node.
func (n *Node) Right() (order.Node, error) {
	if n == nil {
		return nil, errors.New("tried to retrieve child of nil node")
	}
	return n.right, nil
}

// Height returns the height of the subtree rooted at the node.
func (n *Node) Height() int {
	if n == nil {
		return -1
	}
	return n.height
}

// Size returns the size of the subtree rooted at the node.
func (n *Node) Size() int {
	if n == nil {
		return 0
	}
	return n.size
}

// Value returns the value stored at the node.
func (n *Node) Value() float64 {
	return n.val
}

// TreeString returns the string representation of the subtree rooted at the node.
func (n *Node) TreeString() string {
	if n == nil {
		return ""
	}
	return n.treeString("", "", true)
}

func (n *Node) add(val float64) *Node {
	if n == nil {
		return NewNode(val)
	} else if val <= n.val {
		n.left = n.left.add(val)
	} else {
		n.right = n.right.add(val)
	}

	n.size = n.left.Size() + n.right.Size() + 1
	n.height = max(n.left.Height(), n.right.Height()) + 1
	return n.balance()
}

func (n *Node) remove(val float64) *Node {
	// this case occurs if we attempt to remove a value
	// that does not exist in the subtree; this will
	// result in remove() being a no-op
	if n == nil {
		return n
	}

	root := n
	if val < root.val {
		root.left = root.left.remove(val)
	} else if val > root.val {
		root.right = root.right.remove(val)
	} else {
		if root.left == nil {
			return root.right
		} else if root.right == nil {
			return root.left
		}
		root = n.right.min()
		root.right = n.right.removeMin()
		root.left = n.left
	}

	root.size = root.left.Size() + root.right.Size() + 1
	root.height = max(root.left.Height(), root.right.Height()) + 1
	return root.balance()
}

func (n *Node) min() *Node {
	if n.left == nil {
		return n
	}

	return n.left.min()
}

func (n *Node) removeMin() *Node {
	if n.left == nil {
		return n.right
	}

	n.left = n.left.removeMin()
	n.size = n.left.Size() + n.right.Size() + 1
	n.height = max(n.left.Height(), n.right.Height()) + 1
	return n.balance()
}

/*****************
 * Rotations
 *****************/

func (n *Node) balance() *Node {
	if n.heightDiff() < -1 {
		// Since we've entered this block, we already
		// know that the right child is not nil
		if n.right.heightDiff() > 0 {
			n.right = n.right.rotateRight()
		}
		return n.rotateLeft()
	} else if n.heightDiff() > 1 {
		// Since we've entered this block, we already
		// know that the left child is not nil
		if n.left.heightDiff() < 0 {
			n.left = n.left.rotateLeft()
		}
		return n.rotateRight()
	}

	return n
}

func (n *Node) heightDiff() int {
	return n.left.Height() - n.right.Height()
}

func (n *Node) rotateLeft() *Node {
	m := n.right
	n.right = m.left
	m.left = n

	// No need to call size() here; we already know that n is not nil, since
	// rotations are only called for non-leaf nodes
	m.size = n.size
	n.size = n.left.Size() + n.right.Size() + 1

	n.height = max(n.left.Height(), n.right.Height()) + 1
	m.height = max(m.left.Height(), m.right.Height()) + 1

	return m
}

func (n *Node) rotateRight() *Node {
	m := n.left
	n.left = m.right
	m.right = n

	// No need to call size() here; we already know that n is not nil, since
	// rotations are only called for non-leaf nodes
	m.size = n.size
	n.size = n.left.Size() + n.right.Size() + 1

	n.height = max(n.left.Height(), n.right.Height()) + 1
	m.height = max(m.left.Height(), m.right.Height()) + 1

	return m
}

/*******************
 * Order Statistics
 *******************/

// Select returns the node with the kth smallest value in the
// subtree rooted at the node..
func (n *Node) Select(k int) order.Node {
	if n == nil {
		return nil
	}

	size := n.left.Size()
	if k < size {
		return n.left.Select(k)
	} else if k > size {
		return n.right.Select(k - size - 1)
	}

	return n
}

// Rank returns the number of nodes strictly less than the value that
// are contained in the subtree rooted at the node.
func (n *Node) Rank(val float64) int {
	if n == nil {
		return 0
	} else if val < n.val {
		return n.left.Rank(val)
	} else if val > n.val {
		return 1 + n.left.Size() + n.right.Rank(val)
	}
	return n.left.Size()
}

/*******************
 * Pretty-printing
 *******************/

// treeString recursively prints out a subtree rooted at the node in a sideways format, as below:
// │       ┌── 7.000000
// │   ┌── 6.000000
// │   │   └── 5.000000
// └── 4.000000
//
//	│   ┌── 3.000000
//	└── 2.000000
//	    └── 1.000000
//	        └── 1.000000
func (n *Node) treeString(prefix string, result string, isTail bool) string {
	// isTail indicates whether or not the current node's parent branch needs to be represented
	// as a "tail", i.e. its branch needs to hang in the string representation, rather than branch upwards.
	if isTail {
		// If true, then we need to print the subtree like this:
		// │   ┌── [n.right.treeString()]
		// └── [n.val]
		//     └── [n.left.treeString()]
		if n.right != nil {
			result = n.right.treeString(fmt.Sprintf("%s│   ", prefix), result, false)
		}
		result = fmt.Sprintf("%s%s└── %f\n", result, prefix, n.val)
		if n.left != nil {
			result = n.left.treeString(fmt.Sprintf("%s    ", prefix), result, true)
		}
	} else {
		// If false, then we need to print the subtree like this:
		//     ┌── [n.right.treeString()]
		// ┌── [n.val]
		// │   └── [n.left.treeString()]
		if n.right != nil {
			result = n.right.treeString(fmt.Sprintf("%s    ", prefix), result, false)
		}
		result = fmt.Sprintf("%s%s┌── %f\n", result, prefix, n.val)
		if n.left != nil {
			result = n.left.treeString(fmt.Sprintf("%s│   ", prefix), result, true)
		}
	}

	return result
}
