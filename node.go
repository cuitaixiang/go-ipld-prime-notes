package ipld

// Node represents a value in IPLD.  Any point in a tree of data is a node:
// scalar values (like int, string, etc) are nodes, and
// so are recursive values (like map and list).
//
// Nodes and kinds are described in the IPLD specs at
// https://github.com/ipld/specs/blob/master/data-model-layer/data-model.md .
//
// Methods on the Node interface cover the superset of all possible methods for
// all possible kinds -- but some methods only make sense for particular kinds,
// and thus will only make sense to call on values of the appropriate kind.
// (For example, 'Length' on an int doesn't make sense,
// and 'AsInt' on a map certainly doesn't work either!)
// Use the ReprKind method to find out the kind of value before
// calling kind-specific methods.
// Individual method documentation state which kinds the method is valid for.
// (If you're familiar with the stdlib reflect package, you'll find
// the design of the Node interface very comparable to 'reflect.Value'.)
//
// The Node interface is read-only.  All of the methods on the interface are
// for examining values, and implementations should be immutable.
// The companion interface, NodeBuilder, provides the matching writable
// methods, and should be use to create a (thence immutable) Node.
//
// Keeping Node immutable and separating mutation into NodeBuilder makes
// it possible to perform caching (or rather, memoization, since there's no
// such thing as cache invalidation for immutable systems) of computed
// properties of Node; use copy-on-write algorithms for memory efficiency;
// and to generally build pleasant APIs.
// Many library functions will rely on the immutability of Node (e.g.,
// assuming that pointer-equal nodes do not change in value over time),
// so any user-defined Node implementations should be careful to uphold
// the immutability contract.)
//
// There are many different concrete types which implement Node.
// The primary purpose of various node implementations is to organize
// memory in the program in different ways -- some in-memory layouts may
// be more optimal for some programs than others, and changing the Node
// (and NodeBuilder) implementations lets the programmer choose.
//
// For concrete implementations of Node, check out the "./impl/" folder,
// and the packages within it.
// "impl/free" should probably be your first start; the Node and NodeBuilder
// implementations in that package work for any data.
// Other packages are optimized for specific use-cases.
// Codegen tools can also be used to produce concrete implementations of Node;
// these may be specific to certain data, but still conform to the Node
// interface for interoperability and to support higher-level functions.
//
// Nodes may also be *typed* -- see the 'schema' and 'impl/typed' packages.
// Typed nodes have additional constraints and behaviors (and have a
// `.Type().Kind()` in addition to their `.ReprKind()`!), but still behave
// as a regular Node in all the basic ways.
type Node interface {
	// ReprKind returns a value from the ReprKind enum describing what the
	// essential serializable kind of this node is (map, list, int, etc).
	// Most other handling of a node requires first switching upon the kind.
	ReprKind() ReprKind

	// LookupString looks up a child object in this node and returns it.
	// The returned Node may be any of the ReprKind:
	// a primitive (string, int, etc), a map, a list, or a link.
	//
	// If the Kind of this Node is not ReprKind_Map, a nil node and an error
	// will be returned.
	//
	// If the key does not exist, a nil node and an error will be returned.
	LookupString(key string) (Node, error)

	// Lookup is the equivalent of LookupString, but takes a reified Node
	// as a parameter instead of a plain string.
	// This mechanism is useful if working with typed maps (if the key types
	// have constraints, and you already have a reified `schema.TypedNode` value,
	// using that value can save parsing and validation costs);
	// and may simply be convenient if you already have a Node value in hand.
	//
	// (When writing generic functions over Node, a good rule of thumb is:
	// when handling a map, check for `schema.TypedNode`, and in this case prefer
	// the Lookup(Node) method; otherwise, favor LookupString; typically
	// implementations will have their fastest paths thusly.)
	Lookup(key Node) (Node, error)

	// LookupIndex is the equivalent of LookupString but for indexing into a list.
	// As with LookupString, the returned Node may be any of the ReprKind:
	// a primitive (string, int, etc), a map, a list, or a link.
	//
	// If the Kind of this Node is not ReprKind_List, a nil node and an error
	// will be returned.
	//
	// If idx is out of range, a nil node and an error will be returned.
	LookupIndex(idx int) (Node, error)

	// LookupSegment is will act as either LookupString or LookupIndex,
	// whichever is contextually appropriate.
	//
	// Using LookupSegment may imply an "atoi" conversion if used on a list node,
	// or an "itoa" conversion if used on a map node.  If an "itoa" conversion
	// takes place, it may error, and this method may return that error.
	LookupSegment(seg PathSegment) (Node, error)

	// Note that when using codegenerated types, there may be a fifth variant
	// of lookup method on maps: `Get($GeneratedTypeKey) $GeneratedTypeValue`!

	// MapIterator returns an iterator which yields key-value pairs
	// traversing the node.
	// If the node kind is anything other than a map, nil will be returned.
	//
	// The iterator will yield every entry in the map; that is, it
	// can be expected that itr.Next will be called node.Length times
	// before itr.Done becomes true.
	MapIterator() MapIterator

	// ListIterator returns an iterator which yields key-value pairs
	// traversing the node.
	// If the node kind is anything other than a list, nil will be returned.
	//
	// The iterator will yield every entry in the list; that is, it
	// can be expected that itr.Next will be called node.Length times
	// before itr.Done becomes true.
	ListIterator() ListIterator

	// Length returns the length of a list, or the number of entries in a map,
	// or -1 if the node is not of list nor map kind.
	Length() int

	// Undefined nodes are returned when traversing a struct field that is
	// defined by a schema but unset in the data.  (Undefined nodes are not
	// possible otherwise; you'll only see them from `schema.TypedNode`.)
	// The undefined flag is necessary so iterating over structs can
	// unambiguously make the distinction between values that are
	// present-and-null versus values that are absent.
	IsUndefined() bool

	IsNull() bool
	AsBool() (bool, error)
	AsInt() (int, error)
	AsFloat() (float64, error)
	AsString() (string, error)
	AsBytes() ([]byte, error)
	AsLink() (Link, error)

	// Style returns a NodeStyle which can describe some properties of this node's implementation,
	// and also be used to get a NodeBuilder,
	// which can be use to create new nodes with the same implementation as this one.
	//
	// For typed nodes, the NodeStyle will also implement schema.Type.
	//
	// For Advanced Data Layouts, the NodeStyle will encapsulate any additional
	// parameters and configuration of the ADL, and will also (usually)
	// implement NodeStyleSupportingAmend.
	//
	// Calling this method should not cause an allocation.
	Style() NodeStyle
}

// NodeStyle describes a node implementation (all Node have a NodeStyle),
// and a NodeStyle can always be used to get a NodeBuilder.
//
// A NodeStyle may also provide other information about implementation;
// such information is specific to this library ("style" isn't a concept
// you'll find in the IPLD Specifications), and is usually provided through
// feature-detection interfaces (for example, see NodeStyleSupportingAmend).
//
// Generic algorithms for working with IPLD Nodes make use of NodeStyle
// to get builders for new nodes when creating data, and can also use the
// feature-detection interfaces to help decide what kind of operations
// will be optimal to use on a given node implementation.
//
// Note that NodeStyle is not the same as schema.Type.
// NodeStyle is a (golang-specific!) way to reflect upon the implementation
// and in-memory layout of some IPLD data.
// schema.Type is information about how a group of nodes is related in a schema
// (if they have one!) and the rules that the type mandates the node must follow.
// (Every node must have a style; but schema types are an optional feature.)
type NodeStyle interface {
	// NewBuilder returns a NodeBuilder that can be used to create a new Node.
	//
	// Note that calling NewBuilder often performs an allocation
	// (while in contrast, getting a NodeStyle typically does not!) --
	// this may be consequential when writing high performance code.
	NewBuilder() NodeBuilder
}

// NodeStyleSupportingAmend is a feature-detection interface that can be
// used on a NodeStyle to see if it's possible to build new nodes of this style
// while sharing some internal data in a copy-on-write way.
//
// For example, Nodes using an Advanced Data Layout will typically
// support this behavior, and since ADLs are often used for handling large
// volumes of data, detecting and using this feature can result in significant
// performance savings.
type NodeStyleSupportingAmend interface {
	AmendingBuilder(base Node) NodeBuilder
	// FUTURE: probably also needs a `AmendingWithout(base Node, filter func(k,v) bool) NodeBuilder`, or similar.
	//  ("deletion" based APIs are also possible but both more complicated in interfaces added, and prone to accidentally quadratic usage.)
	// FUTURE: there should be some stdlib `Copy` (?) methods that automatically look for this feature, and fallback if absent.
	//  Might include a wide range of point `Transform`, etc, methods.
	// FUTURE: consider putting this (and others like it) in a `feature` package, if there begin to be enough of them and docs get crowded.
}

// MapIterator is an interface for traversing map nodes.
// Sequential calls to Next() will yield key-value pairs;
// Done() describes whether iteration should continue.
//
// Iteration order is defined to be stable: two separate MapIterator
// created to iterate the same Node will yield the same key-value pairs
// in the same order.
// The order itself may be defined by the Node implementation: some
// Nodes may retain insertion order, and some may return iterators which
// always yield data in sorted order, for example.
// 迭代器
type MapIterator interface {
	// Next returns the next key-value pair.
	//
	// An error value can also be returned at any step: in the case of advanced
	// data structures with incremental loading, it's possible to encounter
	// cancellation or I/O errors at any point in iteration.
	// If an error is returned, the boolean will always be false (so it's
	// correct to check the bool first and short circuit to continuing if true).
	// If an error is returned, the key and value may be nil.
	Next() (key Node, value Node, err error)

	// Done returns false as long as there's at least one more entry to iterate.
	// When Done returns true, iteration can stop.
	//
	// Note when implementing iterators for advanced data layouts (e.g. more than
	// one chunk of backing data, which is loaded incrementally): if your
	// implementation does any I/O during the Done method, and it encounters
	// an error, it must return 'false', so that the following Next call
	// has an opportunity to return the error.
	Done() bool
}

// ListIterator is an interface for traversing list nodes.
// Sequential calls to Next() will yield index-value pairs;
// Done() describes whether iteration should continue.
//
// A loop which iterates from 0 to Node.Length is a valid
// alternative to using a ListIterator.
type ListIterator interface {
	// Next returns the next index and value.
	//
	// An error value can also be returned at any step: in the case of advanced
	// data structures with incremental loading, it's possible to encounter
	// cancellation or I/O errors at any point in iteration.
	// If an error is returned, the boolean will always be false (so it's
	// correct to check the bool first and short circuit to continuing if true).
	// If an error is returned, the key and value may be nil.
	Next() (idx int, value Node, err error)

	// Done returns false as long as there's at least one more entry to iterate.
	// When Done returns false, iteration can stop.
	//
	// Note when implementing iterators for advanced data layouts (e.g. more than
	// one chunk of backing data, which is loaded incrementally): if your
	// implementation does any I/O during the Done method, and it encounters
	// an error, it must return 'false', so that the following Next call
	// has an opportunity to return the error.
	Done() bool
}

// REVIEW: immediate-mode AsBytes() method (as opposed to e.g. returning
// an io.Reader instance) might be problematic, esp. if we introduce
// AdvancedLayouts which support large bytes natively.
//
// Probable solution is having both immediate and iterator return methods.
// Returning a reader for bytes when you know you want a slice already
// is going to be high friction without purpose in many common uses.
//
// Unclear what SetByteStream() would look like for advanced layouts.
// One could try to encapsulate the chunking entirely within the advlay
// node impl... but would it be graceful?  Not sure.  Maybe.  Hopefully!
// Yes?  The advlay impl would still tend to use SetBytes for the raw
// data model layer nodes its composing, so overall, it shakes out nicely.
