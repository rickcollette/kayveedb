package lib

import (
	"container/list"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/gob"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/chacha20poly1305"
)

const Version string = "v1.2.0"

// CacheEntry holds the node, its position in the access order list, and its dirty state
type CacheEntry struct {
	offset  int64
	node    *Node
	element *list.Element
	dirty   bool // Mark whether the node has unsaved changes
}

// Cache struct with an LRU eviction policy, using sync.Map for concurrency
type Cache struct {
	store   sync.Map   // Using sync.Map for concurrent read/write
	order   *list.List // Doubly linked list to track access order
	size    int
	mu      sync.Mutex                           // Mutex for managing the linked list
	flushFn func(offset int64, node *Node) error // Callback to flush dirty nodes to disk
}

type LogEntry struct {
	Operation string
	Key       string
	Value     []byte
}

type KeyValue struct {
	Key   string
	Value []byte
}

// BTree structure with a node cache and client manager
type BTree struct {
	root    *Node
	t       int
	dbPath  string
	dbName  string
	logName string
	dbFile  *os.File
	logFile *os.File
	hmacKey []byte
	mu      sync.RWMutex
	cache   *Cache // Cache with configurable size
	clients *ClientManager // ClientManager for tracking active clients
}

// Add trailing slash to dbPath if not present
func ensureTrailingSlash(path string) string {
	if path != "" && path[len(path)-1] != '/' {
		return path + "/"
	}
	return path
}

// Node structure remains unchanged
type Node struct {
	keys     []*KeyValue
	children []int64
	isLeaf   bool
	numKeys  int
	offset   int64
}

// NewCache creates a new LRU cache with a given size
func NewCache(size int, flushFn func(offset int64, node *Node) error) *Cache {
	return &Cache{
		order:   list.New(), // Initialize the doubly linked list
		size:    size,
		flushFn: flushFn, // Set the function to flush dirty nodes to disk
	}
}

// Get retrieves a node from the cache and moves it to the front (most recently used)
func (c *Cache) Get(offset int64) (*Node, bool) {
	entry, ok := c.store.Load(offset)
	if !ok {
		return nil, false
	}
	cacheEntry := entry.(*CacheEntry)

	// Move the accessed node to the front of the order list (most recently used)
	c.mu.Lock()
	c.order.MoveToFront(cacheEntry.element)
	c.mu.Unlock()

	return cacheEntry.node, true
}

// Put adds a node to the cache and evicts the least recently used node if necessary
func (c *Cache) Put(offset int64, node *Node, dirty bool) {
	// Check if the node is already in the cache
	if entry, ok := c.store.Load(offset); ok {
		cacheEntry := entry.(*CacheEntry)
		// Update the node and move it to the front of the list
		cacheEntry.node = node
		cacheEntry.dirty = dirty
		c.mu.Lock()
		c.order.MoveToFront(cacheEntry.element)
		c.mu.Unlock()
		return
	}

	// If the cache is full, evict the least recently used node (the tail of the list)
	if c.size > 0 && c.order.Len() >= c.size {
		c.evict()
	}

	// Add the new node to the front of the list (most recently used)
	c.mu.Lock()
	element := c.order.PushFront(offset)
	c.mu.Unlock()
	cacheEntry := &CacheEntry{
		offset:  offset,
		node:    node,
		element: element,
		dirty:   dirty,
	}
	c.store.Store(offset, cacheEntry)
}

// evict removes the least recently used node from the cache
func (c *Cache) evict() {
	// Get the least recently used node (the tail of the list)
	c.mu.Lock()
	tail := c.order.Back()
	c.mu.Unlock()

	if tail == nil {
		return
	}

	offset := tail.Value.(int64)

	// Load the cache entry for the node to be evicted
	entry, ok := c.store.Load(offset)
	if !ok {
		return
	}
	cacheEntry := entry.(*CacheEntry)

	// Flush the dirty node to disk before eviction
	if cacheEntry.dirty {
		if err := c.flushFn(offset, cacheEntry.node); err != nil {
			fmt.Printf("Failed to flush dirty node to disk: %v\n", err)
		}
	}

	// Remove it from both the map and the list
	c.mu.Lock()
	c.order.Remove(tail)
	c.mu.Unlock()
	c.store.Delete(offset)
}

// NewBTree initializes the B-tree and adds a cache with configurable size
func NewBTree(t int, dbPath, dbName, logName string, hmacKey, encryptionKey, nonce []byte, cacheSize int) (*BTree, error) {
	// Ensure the dbPath has a trailing slash
	dbPath = ensureTrailingSlash(dbPath)

	// Default to "kayvee.db" if no dbName is provided
	if dbName == "" {
		dbName = "kayvee.db"
	}

	// Default to "kayvee.log" if no logName is provided
	if logName == "" {
		logName = "kayvee.log"
	}

	// Build full paths for db and log files
	dbFilePath := filepath.Join(dbPath, dbName)
	logFilePath := filepath.Join(dbPath, logName)

	flushFn := func(offset int64, node *Node) error {
		// Flush the node to disk before eviction
		file, err := os.OpenFile(dbFilePath, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return fmt.Errorf("failed to open database file: %w", err)
		}
		defer file.Close()

		// Seek to the node's offset
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			return fmt.Errorf("failed seeking node at offset %d: %w", offset, err)
		}

		// Encode and write the node to disk
		encoder := gob.NewEncoder(file)
		if err := encoder.Encode(node); err != nil {
			return fmt.Errorf("failed to encode node at offset %d: %w", offset, err)
		}
		return nil
	}

	// Initialize the client manager
	clientManager := NewClientManager()

	b := &BTree{
		t:       t,
		dbPath:  dbPath,
		dbName:  dbName,
		logName: logName,
		hmacKey: hmacKey,
		cache:   NewCache(cacheSize, flushFn), // Initialize a cache with configurable size
		clients: clientManager,                // Initialize ClientManager
	}

	// Open database file
	var err error
	b.dbFile, err = os.OpenFile(dbFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	// Open log file
	b.logFile, err = os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	if err := b.LoadDB(); err != nil {
		return nil, err
	}

	if err := b.LoadLog(encryptionKey, nonce); err != nil {
		return nil, err
	}

	return b, nil
}

// Insert a key-value pair and write to the log.
func (b *BTree) Insert(key string, value, encryptionKey, nonce []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	encValue, err := b.encrypt(value, encryptionKey, nonce)
	if err != nil {
		return err
	}

	kv := &KeyValue{Key: b.hashKey(key), Value: encValue}

	root := b.root
	if root == nil {
		b.root = &Node{
			isLeaf: true,
			keys:   []*KeyValue{kv},
		}
		b.logOperation("CREATE", key, encValue, false) // Log normally
		return nil
	}

	if root.numKeys == 2*b.t-1 {
		newRoot := &Node{children: []int64{root.offset}}
		b.splitChild(newRoot, 0, root)
		b.root = newRoot
		b.insertNonFull(newRoot, kv)

		// Write the new root to disk
		if err := b.writeRoot(); err != nil {
			return err
		}
	}

	b.insertNonFull(root, kv)
	b.logOperation("CREATE", key, encValue, false) // Log normally
	return nil
}

// Update an existing key-value pair and log the operation.
func (b *BTree) Update(key string, newValue, encryptionKey, nonce []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	hKey := b.hashKey(key)
	node := b.search(b.root, hKey)
	if node == nil {
		return errors.New("key not found")
	}

	encValue, err := b.encrypt(newValue, encryptionKey, nonce)
	if err != nil {
		return err
	}

	node.Value = encValue
	b.logOperation("UPDATE", key, encValue, false)
	return nil
}

// Delete removes a key from the B-tree, starting from the given node.
// It ensures that after deletion, nodes have the correct number of keys, merging or borrowing from siblings if necessary.
// Nodes and their children are written back to disk as necessary.
func (b *BTree) Delete(node *Node, key string) error {
	i := 0
	for i < node.numKeys && key > node.keys[i].Key {
		i++
	}

	if i < node.numKeys && key == node.keys[i].Key {
		if node.isLeaf {
			node.keys = append(node.keys[:i], node.keys[i+1:]...)
			node.numKeys--

			// Write the modified node back to disk
			if _, err := b.writeNode(node); err != nil {
				return err
			}
		} else {
			if err := b.deleteInternalNode(node, i); err != nil {
				return err
			}
		}
	} else if !node.isLeaf {
		// Load child and recurse
		child, err := b.readNode(node.children[i]) // Fix: Handle the error return
		if err != nil {
			return fmt.Errorf("failed to load child node: %w", err)
		}

		if child.numKeys < b.t {
			if err := b.fill(node, i); err != nil {
				return err
			}
		}

		if err := b.Delete(child, key); err != nil {
			return err
		}
	}

	// Write the modified node back to disk
	if _, err := b.writeNode(node); err != nil {
		return err
	}

	b.logOperation("DELETE", key, nil, false)
	return nil
}

// Read retrieves and decrypts a value.
func (b *BTree) Read(key string, encryptionKey, nonce []byte) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	hKey := b.hashKey(key)
	item := b.search(b.root, hKey)
	if item == nil {
		return nil, errors.New("key not found")
	}

	decValue, err := b.decrypt(item.Value, encryptionKey, nonce)
	if err != nil {
		return nil, err
	}
	return decValue, nil
}

// LoadDB loads the B-tree structure from the database file.
func (b *BTree) LoadDB() error {
	file, err := os.Open(b.dbName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	// Only load the root node's metadata, and defer loading other nodes on access.
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&b.root); err != nil {
		return err
	}

	// Initialize an empty root if it's a new database
	if b.root == nil {
		b.root = &Node{isLeaf: true}
	}
	return nil
}

// LoadLog replays the operation log to restore the latest state.
func (b *BTree) LoadLog(encryptionKey, nonce []byte) error {
	file, err := os.Open(b.logName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	for {
		var entry LogEntry
		if err := decoder.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// Replay the log but skip writing new logs during replay
		switch entry.Operation {
		case "CREATE":
			b.Insert(entry.Key, entry.Value, encryptionKey, nonce)
			b.logOperation("CREATE", entry.Key, entry.Value, true) // Skip logging
		case "DELETE":
			b.Delete(b.root, entry.Key)
			b.logOperation("DELETE", entry.Key, nil, true) // Skip logging
		}
	}
	return nil
}

// logOperation logs an operation (CREATE/UPDATE/DELETE) to the log file.
// If skipLog is true, the operation will not be logged, typically used during log replay.
func (b *BTree) logOperation(op, key string, value []byte, skipLog bool) error {
	if skipLog {
		return nil // Skip logging if we're replaying logs
	}
	entry := LogEntry{
		Operation: op,
		Key:       key,
		Value:     value,
	}
	encoder := gob.NewEncoder(b.logFile)
	if err := encoder.Encode(entry); err != nil {
		return err
	}
	return b.logFile.Sync() // Sync the log file to disk after writing
}

// encrypt encrypts the provided data using XChaCha20 and returns the encrypted result.
// It uses the encryptionKey and nonce to perform the encryption.
func (b *BTree) encrypt(data, encryptionKey, nonce []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(encryptionKey)
	if err != nil {
		return nil, err
	}
	return aead.Seal(nil, nonce, data, nil), nil
}

// decrypt decrypts the provided encrypted data using XChaCha20.
// It uses the encryptionKey and nonce to perform the decryption and returns the decrypted result.
func (b *BTree) decrypt(data, encryptionKey, nonce []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(encryptionKey)
	if err != nil {
		return nil, err
	}
	return aead.Open(nil, nonce, data, nil)
}

// hashKey hashes the provided key using HMAC with SHA-256.
// It returns the hashed key as a hexadecimal string.
func (b *BTree) hashKey(key string) string {
	mac := hmac.New(func() hash.Hash { return sha256.New() }, b.hmacKey)
	mac.Write([]byte(key))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

// writeNode writes the given node to the database file and returns its offset.
func (b *BTree) writeNode(node *Node) (int64, error) {
	dbFilePath := filepath.Join(b.dbPath, b.dbName)

	file, err := os.OpenFile(dbFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to open dbName file: %w", err)
	}
	defer file.Close()

	// Move to the end of the file to append the new node
	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, fmt.Errorf("failed to append new node: %w", err)
	}

	// Encode and write the node to disk
	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(node); err != nil {
		return 0, fmt.Errorf("failed to encode node: %w", err)
	}

	// Add the node to the cache and mark it as dirty
	b.cache.Put(offset, node, true)

	return offset, nil
}

// readNode reads a node from the database file at the given offset.
func (b *BTree) readNode(offset int64) (*Node, error) {
	dbFilePath := filepath.Join(b.dbPath, b.dbName)

	// Check cache first
	if node, ok := b.cache.Get(offset); ok {
		return node, nil
	}

	// If not found in cache, read from disk
	file, err := os.OpenFile(dbFilePath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open dbName file: %w", err)
	}
	defer file.Close()

	// Seek to the node's position in the file
	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed seeking node at offset %d: %w", offset, err)
	}

	// Decode the node from the file
	var node Node
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&node); err != nil {
		return nil, fmt.Errorf("failed to decode node at offset %d: %w", offset, err)
	}

	// Add the node to the cache as not dirty (it's fresh from disk)
	b.cache.Put(offset, &node, false)

	return &node, nil
}
// splitChild splits a full child node into two and adjusts the parent accordingly.
// The node and its children are written back to disk after the split.
func (b *BTree) splitChild(parent *Node, i int, fullChild *Node) error {
	t := b.t

	// Create a new node that will be the sibling of the full child
	newChild := &Node{
		isLeaf:   fullChild.isLeaf,
		keys:     append([]*KeyValue{}, fullChild.keys[t:]...), // Copy the second half of the keys
		children: append([]int64{}, fullChild.children[t:]...),  // Copy the second half of the children
		numKeys:  t - 1,
	}

	// Write the new child to disk and get its offset
	newChildOffset, err := b.writeNode(newChild)
	if err != nil {
		return err
	}

	// Update the full child
	fullChild.keys = fullChild.keys[:t-1]
	fullChild.children = fullChild.children[:t]
	fullChild.numKeys = t - 1

	// Write the updated full child back to disk
	if _, err := b.writeNode(fullChild); err != nil {
		return err
	}

	// Update the parent node with the new child
	parent.children = append(parent.children[:i+1], append([]int64{newChildOffset}, parent.children[i+1:]...)...)
	parent.keys = append(parent.keys[:i], append([]*KeyValue{fullChild.keys[t-1]}, parent.keys[i:]...)...)
	parent.numKeys++

	// Write the parent node back to disk
	if _, err := b.writeNode(parent); err != nil {
		return err
	}

	return nil
}

// insertNonFull inserts a key into a node that is not full.
// If the node is a leaf, it inserts the key directly. Otherwise, it recurses into the appropriate child.
// The node and its children are written back to disk after the insertion.
func (b *BTree) insertNonFull(node *Node, kv *KeyValue) {
	i := node.numKeys - 1

	if node.isLeaf {
		// Insert directly into the leaf node
		node.keys = append(node.keys, nil)
		for i >= 0 && kv.Key < node.keys[i].Key {
			node.keys[i+1] = node.keys[i]
			i--
		}
		node.keys[i+1] = kv
		node.numKeys++

		// Write the updated node back to disk
		b.writeNode(node)
	} else {
		// Traverse the tree only when necessary
		for i >= 0 && kv.Key < node.keys[i].Key {
			i--
		}
		i++

		// Load the child node only when required
		child, err := b.readNode(node.children[i])
		if err != nil {
			fmt.Printf("Error reading child node: %v\n", err)
			return
		}

		if child.numKeys == 2*b.t-1 {
			// If the child is full, split it
			b.splitChild(node, i, child)
			if kv.Key > node.keys[i].Key {
				i++
			}
		}

		// Re-read the child node after the split
		child, err = b.readNode(node.children[i])
		if err != nil {
			fmt.Printf("Error re-reading child node: %v\n", err)
			return
		}

		b.insertNonFull(child, kv)
	}
}

// deleteInternalNode handles deletion of a key in an internal node.
// Depending on the number of keys in the child nodes, it borrows keys or merges nodes.
func (b *BTree) deleteInternalNode(node *Node, idx int) error {
	t := b.t
	key := node.keys[idx]

	// Case 1: Predecessor child has at least t keys
	predChild, err := b.readNode(node.children[idx])
	if err != nil {
		return fmt.Errorf("failed to read predecessor child: %w", err)
	}
	if predChild.numKeys >= t {
		pred := b.getPredecessor(node, idx)
		node.keys[idx] = pred
		if err := b.Delete(predChild, pred.Key); err != nil {
			return err
		}
		if _, err := b.writeNode(node); err != nil {
			return err
		}
		return nil
	}

	// Case 2: Successor child has at least t keys
	succChild, err := b.readNode(node.children[idx+1])
	if err != nil {
		return fmt.Errorf("failed to read successor child: %w", err)
	}
	if succChild.numKeys >= t {
		succ := b.getSuccessor(node, idx)
		node.keys[idx] = succ
		if err := b.Delete(succChild, succ.Key); err != nil {
			return err
		}
		if _, err := b.writeNode(node); err != nil {
			return err
		}
		return nil
	}

	// Case 3: Both children have fewer than t keys, merge them
	if err := b.merge(node, idx); err != nil {
		return err
	}
	child, err := b.readNode(node.children[idx])
	if err != nil {
		return fmt.Errorf("failed to read child node: %w", err)
	}
	if err := b.Delete(child, key.Key); err != nil {
		return err
	}
	if _, err := b.writeNode(node); err != nil {
		return err
	}

	return nil
}
// merge merges the child at index idx with its sibling.
func (b *BTree) merge(node *Node, idx int) error {
	child, err := b.readNode(node.children[idx])
	if err != nil {
		return fmt.Errorf("failed to read child node: %w", err)
	}
	sibling, err := b.readNode(node.children[idx+1])
	if err != nil {
		return fmt.Errorf("failed to read sibling node: %w", err)
	}

	// Pull the key from the current node down into the child
	child.keys = append(child.keys, node.keys[idx])

	// Append sibling's keys and children to the child
	child.keys = append(child.keys, sibling.keys...)
	if !child.isLeaf {
		child.children = append(child.children, sibling.children...)
	}

	// Remove the key from the current node and the sibling
	node.keys = append(node.keys[:idx], node.keys[idx+1:]...)
	node.children = append(node.children[:idx+1], node.children[idx+2:]...)

	child.numKeys += sibling.numKeys + 1
	node.numKeys--

	// Write the modified nodes back to disk
	if _, err := b.writeNode(child); err != nil {
		return err
	}
	if _, err := b.writeNode(node); err != nil {
		return err
	}

	return nil
}

// fill ensures that the child node has at least t keys by borrowing or merging from/to its siblings.
func (b *BTree) fill(node *Node, idx int) error {
	// If the previous sibling has more than t-1 keys, borrow from it
	if idx != 0 {
		prevSibling, err := b.readNode(node.children[idx-1])
		if err != nil {
			return fmt.Errorf("failed to read previous sibling: %w", err)
		}
		if prevSibling.numKeys >= b.t {
			if err := b.borrowFromPrev(node, idx); err != nil {
				return fmt.Errorf("failed to borrow from previous sibling: %w", err)
			}
			return nil
		}
	}

	// If the next sibling has more than t-1 keys, borrow from it
	if idx != node.numKeys {
		nextSibling, err := b.readNode(node.children[idx+1])
		if err != nil {
			return fmt.Errorf("failed to read next sibling: %w", err)
		}
		if nextSibling.numKeys >= b.t {
			if err := b.borrowFromNext(node, idx); err != nil {
				return fmt.Errorf("failed to borrow from next sibling: %w", err)
			}
			return nil
		}
	}

	// Merge the child with either its previous or next sibling
	if idx != node.numKeys {
		if err := b.merge(node, idx); err != nil {
			return fmt.Errorf("failed to merge with previous sibling: %w", err)
		}
	} else {
		if err := b.merge(node, idx-1); err != nil {
			return fmt.Errorf("failed to merge with next sibling: %w", err)
		}
	}
	return nil
}

// borrowFromPrev borrows a key from the previous sibling and inserts it into the child.
func (b *BTree) borrowFromPrev(node *Node, idx int) error {
	child, err := b.readNode(node.children[idx])
	if err != nil {
		return fmt.Errorf("failed to read child node: %w", err)
	}
	sibling, err := b.readNode(node.children[idx-1])
	if err != nil {
		return fmt.Errorf("failed to read previous sibling: %w", err)
	}

	// Move the key from the parent down to the child
	child.keys = append([]*KeyValue{node.keys[idx-1]}, child.keys...)
	node.keys[idx-1] = sibling.keys[sibling.numKeys-1]

	// Move the sibling's last child to the child
	if !child.isLeaf {
		child.children = append([]int64{sibling.children[sibling.numKeys]}, child.children...)
	}

	sibling.numKeys--
	child.numKeys++

	// Write the modified nodes back to disk
	if _, err := b.writeNode(child); err != nil {
		return err
	}
	if _, err := b.writeNode(sibling); err != nil {
		return err
	}
	if _, err := b.writeNode(node); err != nil {
		return err
	}

	return nil
}

// borrowFromNext borrows a key from the next sibling and inserts it into the child.
func (b *BTree) borrowFromNext(node *Node, idx int) error {
	child, err := b.readNode(node.children[idx])
	if err != nil {
		return fmt.Errorf("failed to read child node: %w", err)
	}
	sibling, err := b.readNode(node.children[idx+1])
	if err != nil {
		return fmt.Errorf("failed to read next sibling: %w", err)
	}

	// Move the key from the parent down to the child
	child.keys = append(child.keys, node.keys[idx])
	node.keys[idx] = sibling.keys[0]

	// Move the sibling's first child to the child
	if !child.isLeaf {
		child.children = append(child.children, sibling.children[0])
		sibling.children = sibling.children[1:]
	}

	sibling.keys = sibling.keys[1:]
	sibling.numKeys--
	child.numKeys++

	// Write the modified nodes back to disk
	if _, err := b.writeNode(child); err != nil {
		return err
	}
	if _, err := b.writeNode(sibling); err != nil {
		return err
	}
	if _, err := b.writeNode(node); err != nil {
		return err
	}

	return nil
}
// writeRoot writes the root node of the BTree to disk.
func (b *BTree) writeRoot() error {
	// Write the root node and get its offset
	offset, err := b.writeNode(b.root)
	if err != nil {
		return err
	}

	// Move to the start of the file to write the root offset
	_, err = b.dbFile.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	// Encode and write the offset
	encoder := gob.NewEncoder(b.dbFile)
	return encoder.Encode(offset)
}

// search looks for a key in the BTree, starting from the given node.
// It returns the KeyValue pair if found or nil if not found.
func (b *BTree) search(node *Node, key string) *KeyValue {
	if node == nil {
		return nil
	}

	// Find the index where the key would be in the current node
	i := 0
	for i < node.numKeys && key > node.keys[i].Key {
		i++
	}

	// If the key is found, return the key-value pair
	if i < node.numKeys && key == node.keys[i].Key {
		return node.keys[i]
	}

	// If the node is a leaf, stop the search
	if node.isLeaf {
		return nil
	}

	// Recursively search the child node
	child, err := b.readNode(node.children[i])
	if err != nil {
		fmt.Printf("failed to load child node: %v\n", err)
		return nil
	}

	return b.search(child, key)
}

// getPredecessor finds the predecessor of a key in the BTree.
func (b *BTree) getPredecessor(node *Node, idx int) *KeyValue {
	current, err := b.readNode(node.children[idx])
	if err != nil {
		fmt.Printf("failed to read predecessor node: %v\n", err)
		return nil
	}

	// Traverse to the rightmost leaf
	for !current.isLeaf {
		current, err = b.readNode(current.children[current.numKeys])
		if err != nil {
			fmt.Printf("failed to read child node: %v\n", err)
			return nil
		}
	}

	// Return the last key of the rightmost node
	return current.keys[current.numKeys-1]
}

// getSuccessor finds the successor of a key in the BTree.
func (b *BTree) getSuccessor(node *Node, idx int) *KeyValue {
	current, err := b.readNode(node.children[idx+1])
	if err != nil {
		fmt.Printf("failed to get successor node: %v\n", err)
		return nil
	}

	// Traverse to the leftmost leaf
	for !current.isLeaf {
		current, err = b.readNode(current.children[0])
		if err != nil {
			fmt.Printf("failed to read child node: %v\n", err)
			return nil
		}
	}

	// Return the first key of the leftmost node
	return current.keys[0]
}
