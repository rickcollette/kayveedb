
# kayveedb Go Package Documentation

## Current version: **v1.1.0**

## Overview

`kayveedb` is a Go package that implements a B-Tree-based key-value store with caching support, using an LRU (Least Recently Used) eviction policy. It also features encryption for stored values using ChaCha20 encryption and HMAC for key hashing.

### Version

## Installation

To use `kayveedb`, you can import it as a Go package.

```bash
go get github.com/yourusername/kayveedb
```

## Package Contents

### Constants

- `Version`: The current version of the package.

### Functions

#### `ShowVersion`

Displays the current version of the package.

**Signature:**

```go
func ShowVersion() string
```

**Example:**

```go
fmt.Println(kayveedb.ShowVersion())
```

### Cache

#### CacheEntry

The `CacheEntry` struct stores the node in the cache and its access order.

**Fields:**

- `offset int64`: The node's offset in the file.
- `node *Node`: The actual node.
- `element *list.Element`: The position in the access order list.
- `dirty bool`: Whether the node has unsaved changes.

#### Cache

The `Cache` struct implements an LRU cache with concurrency support, using a `sync.Map` to store nodes and a `list.List` for managing access order.

**Fields:**

- `store sync.Map`: Concurrent map of cached nodes.
- `order *list.List`: Linked list to track access order.
- `size int`: Maximum number of entries in the cache.
- `mu sync.Mutex`: Mutex to protect the cache operations.
- `flushFn func(offset int64, node *Node) error`: A callback function to flush dirty nodes to disk.

**Methods:**

- `Get(offset int64) (*Node, bool)`: Retrieves a node from the cache, moving it to the front of the access order list.
- `Put(offset int64, node *Node, dirty bool)`: Adds a node to the cache and marks it as dirty if it has unsaved changes.
- `evict()`: Evicts the least recently used node and flushes it to disk if it's dirty.

### BTree

The `BTree` struct implements a B-Tree with caching. It uses the cache for efficient node storage and retrieval.

**Fields:**

- `root *Node`: The root node of the B-Tree.
- `t int`: The minimum degree of the B-Tree.
- `dbPath string`: Path to the database file.
- `logName string`: Path to the operation log file.
- `dbFile *os.File`: Database file handle.
- `logFile *os.File`: Operation log file handle.
- `hmacKey []byte`: Key for HMAC hashing.
- `mu sync.RWMutex`: Read-write mutex for safe concurrent access.
- `cache *Cache`: LRU cache for storing nodes.

**Methods:**

#### `NewBTree`

Creates a new B-Tree with a configurable cache size.  
The database files are either the names provided, or will write to `$CWD/kayvee.db` and `$CWD/kayvee.log`.

**Signature:**

```go
func NewBTree(t int, dbName, logName string, hmacKey, encryptionKey, nonce []byte, cacheSize int) (*BTree, error)
```

**Parameters:**

- `t int`: Minimum degree of the B-Tree.
- `dbName string`: Path to the database file.
- `logName string`: Path to the operation log file.
- `hmacKey []byte`: HMAC key for hashing.
- `encryptionKey []byte`: Encryption key for value encryption.
- `nonce []byte`: Nonce for ChaCha20 encryption.
- `cacheSize int`: Size of the cache.

**Example:**

```go
// Using a custom database file
tree, err := kayveedb.NewBTree(3, "/path/to/mydb.db", "/path/to/mylog.log", hmacKey, encryptionKey, nonce, 100)

// Using the default database file ($CWD/kayvee.db)
tree, err := kayveedb.NewBTree(3, "", "", hmacKey, encryptionKey, nonce, 100)
if err != nil {
    log.Fatal(err)
}
```

#### `Insert`

Inserts a new key-value pair into the B-Tree.

**Signature:**

```go
func (b *BTree) Insert(key string, value, encryptionKey, nonce []byte) error
```

**Parameters:**

- `key string`: Key to insert.
- `value []byte`: Value to insert (will be encrypted).
- `encryptionKey []byte`: Encryption key.
- `nonce []byte`: Nonce for encryption.

**Example:**

```go
err := tree.Insert("mykey", []byte("myvalue"), encryptionKey, nonce)
if err != nil {
    log.Fatal(err)
}
```

#### `Update`

Updates an existing key-value pair in the B-Tree.

**Signature:**

```go
func (b *BTree) Update(key string, newValue, encryptionKey, nonce []byte) error
```

**Parameters:**

- `key string`: Key to update.
- `newValue []byte`: New value (will be encrypted).
- `encryptionKey []byte`: Encryption key.
- `nonce []byte`: Nonce for encryption.

**Example:**

```go
err := tree.Update("mykey", []byte("newvalue"), encryptionKey, nonce)
if err != nil {
    log.Fatal(err)
}
```

#### `Delete`

Deletes a key-value pair from the B-Tree.

**Signature:**

```go
func (b *BTree) Delete(node *Node, key string) error
```

**Parameters:**

- `node *Node`: Starting node.
- `key string`: Key to delete.

**Example:**

```go
err := tree.Delete(tree.root, "mykey")
if err != nil {
    log.Fatal(err)
}
```

#### `Read`

Reads and decrypts a value from the B-Tree.

**Signature:**

```go
func (b *BTree) Read(key string, encryptionKey, nonce []byte) ([]byte, error)
```

**Parameters:**

- `key string`: Key to read.
- `encryptionKey []byte`: Encryption key.
- `nonce []byte`: Nonce for decryption.

**Example:**

```go
value, err := tree.Read("mykey", encryptionKey, nonce)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Value:", string(value))
```

#### `Close`

Closes the B-Tree, flushing any unsaved data to disk.

**Signature:**

```go
func (b *BTree) Close() error
```

**Example:**

```go
err := tree.Close()
if err != nil {
    log.Fatal(err)
}
```

### Node

The `Node` struct represents a node in the B-Tree.

**Fields:**

- `keys []*KeyValue`: Slice of key-value pairs stored in the node.
- `children []int64`: Slice of child node offsets.
- `isLeaf bool`: Indicates whether the node is a leaf.
- `numKeys int`: Number of keys in the node.
- `offset int64`: Offset of the node in the file.

## Encryption and HMAC

The `kayveedb` package uses ChaCha20 for encryption and HMAC with SHA-256 for key hashing.

- **Encryption:** Uses `chacha20poly1305` from the `golang.org/x/crypto` package.
- **HMAC:** Uses SHA-256 for hashing keys.

### Encryption Functions

#### `encrypt`

Encrypts data using ChaCha20-Poly1305.

**Signature:**

```go
func (b *BTree) encrypt(data, encryptionKey, nonce []byte) ([]byte, error)
```

#### `decrypt`

Decrypts data using ChaCha20-Poly1305.

**Signature:**

```go
func (b *BTree) decrypt(data, encryptionKey, nonce []byte) ([]byte, error)
```

### HMAC Functions

#### `hashKey`

Hashes a key using HMAC with SHA-256.

**Signature:**

```go
func (b *BTree) hashKey(key string) string
```

## Cache Example

Here’s a simple example of using the cache system to store and retrieve nodes:

```go
// Initialize the cache
cache := kayveedb.NewCache(100, func(offset int64, node *kayveedb.Node) error {
    // Simulate flushing a node to disk
    fmt.Printf("Flushing node at offset %d\n", offset)
    return nil
})

// Create a sample node
node := &kayveedb.Node{
    keys: []*kayveedb.KeyValue{{Key: "samplekey", Value: []byte("samplevalue")}},
}

// Add the node to the cache
cache.Put(1, node, true)

// Retrieve the node from the cache
cachedNode, ok := cache.Get(1)
if ok {
    fmt.Println("Node found in cache:", cachedNode)
}
```

### Function Definition

```go
func (b *BTree) ListKeys() ([]string, error)
```

### Description

The `ListKeys` function traverses the B-tree and returns a slice of all keys stored in the tree.

- **Return Type**: `([]string, error)`
  - A slice of strings containing all the keys in the B-tree.
  - An error if the operation fails during the traversal.

### Traversal Method

This method uses a helper function `traverse` that recursively walks through each node in the B-tree, collecting keys from each node and appending them to the final result.

#### Helper Function

```go
func (b *BTree) traverse(node *Node, keys *[]string) error
```

The `traverse` function is responsible for visiting each node, retrieving the keys, and recursively exploring child nodes (if applicable).

## Usage Example

Here’s how to use the `ListKeys` function:

### Example 1: Listing Keys in the BTree

```go
package main

import (
    "fmt"
    "log"
    "kayveedb" // Assuming your package is called kayveedb
)

func main() {
    // Initialize your B-tree with a minimum degree
    bt, err := kayveedb.NewBTree(3, "./", "testdb", "testlog", nil, nil, nil, 100)
    if err != nil {
        log.Fatalf("Failed to initialize B-tree: %v", err)
    }

    // Insert some keys into the BTree
    bt.Insert("key1", []byte("value1"), nil, nil)
    bt.Insert("key2", []byte("value2"), nil, nil)
    bt.Insert("key3", []byte("value3"), nil, nil)

    // List all keys in the BTree
    keys, err := bt.ListKeys()
    if err != nil {
        log.Fatalf("Failed to list keys: %v", err)
    }

    // Print the keys
    fmt.Println("Keys in the B-tree:", keys)
}
```

### Example 2: Handling Empty Trees

If the B-tree is empty, the function will return an empty slice:

```go
package main

import (
    "fmt"
    "log"
    "kayveedb"
)

func main() {
    // Initialize an empty B-tree
    bt, err := kayveedb.NewBTree(3, "./", "testdb", "testlog", nil, nil, nil, 100)
    if err != nil {
        log.Fatalf("Failed to initialize B-tree: %v", err)
    }

    // List keys in the empty B-tree
    keys, err := bt.ListKeys()
    if err != nil {
        log.Fatalf("Failed to list keys: %v", err)
    }

    // Expecting an empty list
    fmt.Println("Keys in the empty B-tree:", keys)
}
```

## Error Handling

The `ListKeys` function will return an error in case of any failure while traversing the B-tree or reading a node from the disk. Always check the error and handle it appropriately in your code.

### Example Error Handling

```go
keys, err := bt.ListKeys()
if err != nil {
    log.Fatalf("Error while listing keys: %v", err)
}
```

## License

This package is licensed under the MIT License.
