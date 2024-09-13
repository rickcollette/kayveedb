/*
Package kayveedb provides a B-tree-based key-value database with XChaCha20 encryption for both in-memory and at-rest data,
as well as AES-256 for secure key hashing. It implements log-based persistence, where only changes are logged and applied
later for efficiency. This package is suitable for environments requiring security and performance, ensuring that both in-memory
and at-rest data are encrypted.

Functions in this package require the encryption key and nonce to be provided by the calling application, allowing for flexible
key management.
*/
package kayveedb

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/gob"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"sync"

	"golang.org/x/crypto/chacha20poly1305"
)

// ShowVersion prints the current version of the kayveedb package.
func ShowVersion() string {
	return Version
}

// LogEntry represents an operation log entry (CREATE/DELETE) for log-based persistence.
type LogEntry struct {
	Operation string // Operation performed (CREATE/DELETE)
	Key       string // Key involved in the operation
	Value     []byte // Encrypted value for CREATE, nil for DELETE
}

// KeyValue represents a key-value pair stored in the B-tree, where the value is encrypted.
type KeyValue struct {
	Key   string // Hashed key
	Value []byte // Encrypted value
}

// BTree defines a B-tree structure with XChaCha20 encryption and log-based persistence.
type BTree struct {
	root        *Node         // Root node of the B-tree
	t           int           // Minimum degree (defines the range for number of keys)
	snapshot    string        // File for storing snapshots
	opLog       string        // File for storing the operation log
	opLogFile   *os.File      // Open file for logging operations
	hmacKey     []byte        // AES-256 HMAC key for key hashing
	mu          sync.RWMutex  // RWMutex for thread-safe operations
}

// Node defines a node within the B-tree.
type Node struct {
	keys      []*KeyValue // List of key-value pairs
	children  []*Node     // Pointers to child nodes
	isLeaf    bool        // Whether the node is a leaf
	numKeys   int         // Number of keys in the node
}

// NewBTree initializes a new B-tree with XChaCha20 encryption and log-based persistence.
// Requires an encryption key and HMAC key for hashing, as well as an encryption nonce.
func NewBTree(t int, snapshot, logPath string, hmacKey, encryptionKey, nonce []byte) (*BTree, error) {
	// Validate encryption key size
	if len(encryptionKey) != chacha20poly1305.KeySize {
		return nil, errors.New("invalid encryption key size")
	}

	b := &BTree{
		t:        t,
		snapshot: snapshot,
		opLog:    logPath,
		hmacKey:  hmacKey,
	}

	// Load snapshot and log file
	if err := b.loadSnapshot(); err != nil {
		return nil, err
	}

	// Pass encryption key and nonce to loadLog
	if err := b.loadLog(encryptionKey, nonce); err != nil {
		return nil, err
	}

	// Open the log file for appending
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	b.opLogFile = file

	return b, nil
}


// Insert adds a new key-value pair into the B-tree. The caller must provide an encryption key and nonce for XChaCha20 encryption.
func (b *BTree) Insert(key string, value, encryptionKey, nonce []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Encrypt the value using the provided key and nonce
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
		b.logOperation("CREATE", key, encValue)
		return nil
	}

	if root.numKeys == 2*b.t-1 {
		newRoot := &Node{children: []*Node{root}}
		b.splitChild(newRoot, 0, root)
		b.root = newRoot
		b.insertNonFull(newRoot, kv)
	} else {
		b.insertNonFull(root, kv)
	}

	// Log the operation
	b.logOperation("CREATE", key, encValue)
	return nil
}

// Update modifies an existing key-value pair in the B-tree.
// It requires the caller to provide an encryption key and nonce for encryption.
func (b *BTree) Update(key string, newValue, encryptionKey, nonce []byte) error {
    b.mu.Lock()
    defer b.mu.Unlock()

    hKey := b.hashKey(key)
    node := b.search(b.root, hKey)
    if node == nil {
        return errors.New("key not found")
    }

    // Encrypt the new value
    encValue, err := b.encrypt(newValue, encryptionKey, nonce)
    if err != nil {
        return err
    }

    // Update the value in the node
    node.Value = encValue

    // Log the operation as an UPDATE (using "CREATE" as we're replacing the value)
    b.logOperation("CREATE", key, encValue)
    return nil
}

// Delete removes a key-value pair from the B-tree. Logs the operation for persistence.
func (b *BTree) Delete(key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	hKey := b.hashKey(key)

	if b.root == nil {
		return nil
	}
	b.delete(b.root, hKey)

	if b.root.numKeys == 0 && !b.root.isLeaf {
		b.root = b.root.children[0]
	}

	// Log the operation
	b.logOperation("DELETE", key, nil)
	return nil
}

// resetLog resets the operation log file after a snapshot is taken.
func (b *BTree) resetLog() error {
	// Close the current operation log file
	if err := b.opLogFile.Close(); err != nil {
		return err
	}

	// Truncate or delete the operation log file and create a new one
	file, err := os.Create(b.opLog)
	if err != nil {
		return err
	}
	b.opLogFile = file
	return nil
}

// loadSnapshot loads the B-tree from the snapshot file if it exists.
func (b *BTree) loadSnapshot() error {
	file, err := os.Open(b.snapshot)
	if err != nil {
		// If the snapshot file does not exist, assume this is a fresh start
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	// Use Gob decoding to load the B-tree structure from the snapshot file
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&b.root); err != nil {
		return err
	}
	return nil
}

// loadLog replays the operation log to restore the tree to the latest state.
// The caller must pass an encryption key and nonce.
func (b *BTree) loadLog(encryptionKey, nonce []byte) error {
	file, err := os.Open(b.opLog)
	if err != nil {
		// If the log file does not exist, assume this is a fresh start
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	for {
		var entry LogEntry
		// Decode the next log entry; if EOF, we're done
		if err := decoder.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// Replay the operation based on the log entry
		switch entry.Operation {
		case "CREATE":
			// Insert the encrypted value directly
			b.Insert(entry.Key, entry.Value, encryptionKey, nonce)
		case "DELETE":
			b.Delete(entry.Key)
		}
	}
	return nil
}


// Read retrieves the value for a given key, decrypting it using the provided encryption key and nonce.
func (b *BTree) Read(key string, encryptionKey, nonce []byte) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	hKey := b.hashKey(key)
	item := b.search(b.root, hKey)
	if item == nil {
		return nil, errors.New("key not found")
	}

	// Decrypt the value before returning
	decValue, err := b.decrypt(item.Value, encryptionKey, nonce)
	if err != nil {
		return nil, err
	}

	return decValue, nil
}

// Snapshot creates a snapshot of the current B-tree state (in-memory structure). The data is encrypted at rest.
func (b *BTree) Snapshot() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	file, err := os.Create(b.snapshot)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(b.root); err != nil {
		return err
	}

	// Reset the log file after taking a snapshot
	return b.resetLog()
}

// logOperation writes an operation (INSERT/DELETE) to the log file for persistence.
func (b *BTree) logOperation(op, key string, value []byte) {
	entry := LogEntry{
		Operation: op,
		Key:       key,
		Value:     value,
	}
	encoder := gob.NewEncoder(b.opLogFile)
	if err := encoder.Encode(entry); err != nil {
		panic(err)
	}
	b.opLogFile.Sync() // Ensure it is flushed to disk
}

// Encryption and decryption functions using the provided XChaCha20 key and nonce.
func (b *BTree) encrypt(data, encryptionKey, nonce []byte) ([]byte, error) {
	if len(encryptionKey) != chacha20poly1305.KeySize {
		return nil, errors.New("invalid encryption key size")
	}
	aead, err := chacha20poly1305.NewX(encryptionKey)
	if err != nil {
		return nil, err
	}

	ciphertext := aead.Seal(nil, nonce, data, nil)
	return ciphertext, nil
}

func (b *BTree) decrypt(data, encryptionKey, nonce []byte) ([]byte, error) {
	if len(encryptionKey) != chacha20poly1305.KeySize {
		return nil, errors.New("invalid encryption key size")
	}
	aead, err := chacha20poly1305.NewX(encryptionKey)
	if err != nil {
		return nil, err
	}

	plaintext, err := aead.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// Hashing function for keys using HMAC with AES-256.
func (b *BTree) hashKey(key string) string {
	mac := hmac.New(func() hash.Hash { return sha256.New() }, b.hmacKey)
	mac.Write([]byte(key))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

// Other B-tree internal functions for managing insertions, deletions, splits, and merges.
func (b *BTree) splitChild(parent *Node, i int, fullChild *Node) {
	t := b.t
	newChild := &Node{
		isLeaf:   fullChild.isLeaf,
		keys:     append([]*KeyValue{}, fullChild.keys[t:]...),
		children: append([]*Node{}, fullChild.children[t:]...),
	}
	fullChild.keys = fullChild.keys[:t-1]
	fullChild.children = fullChild.children[:t]

	parent.children = append(parent.children[:i+1], append([]*Node{newChild}, parent.children[i+1:]...)...)
	parent.keys = append(parent.keys[:i], append([]*KeyValue{fullChild.keys[t-1]}, parent.keys[i:]...)...)
	fullChild.numKeys = t - 1
	newChild.numKeys = t - 1
	parent.numKeys++
}

func (b *BTree) insertNonFull(node *Node, kv *KeyValue) {
	i := node.numKeys - 1

	if node.isLeaf {
		node.keys = append(node.keys, nil)
		for i >= 0 && kv.Key < node.keys[i].Key {
			node.keys[i+1] = node.keys[i]
			i--
		}
		node.keys[i+1] = kv
		node.numKeys++
	} else {
		for i >= 0 && kv.Key < node.keys[i].Key {
			i--
		}
		i++
		if node.children[i].numKeys == 2*b.t-1 {
			b.splitChild(node, i, node.children[i])
			if kv.Key > node.keys[i].Key {
				i++
			}
		}
		b.insertNonFull(node.children[i], kv)
	}
}
// search looks for a key in the B-tree, starting from the given node.
// It returns the KeyValue pair if found or nil if not found.
func (b *BTree) search(node *Node, key string) *KeyValue {
	if node == nil {
		return nil
	}
	i := 0
	for i < node.numKeys && key > node.keys[i].Key {
		i++
	}
	if i < node.numKeys && key == node.keys[i].Key {
		return node.keys[i]
	} else if node.isLeaf {
		return nil
	} else {
		return b.search(node.children[i], key)
	}
}

// delete removes a key from the B-tree, starting from the given node.
func (b *BTree) delete(node *Node, key string) {
	i := 0
	for i < node.numKeys && key > node.keys[i].Key {
		i++
	}

	if i < node.numKeys && key == node.keys[i].Key {
		// Key found in the current node
		if node.isLeaf {
			// Case 1: Key is in a leaf node
			node.keys = append(node.keys[:i], node.keys[i+1:]...)
			node.numKeys--
		} else {
			// Case 2: Key is in an internal node
			b.deleteInternalNode(node, i)
		}
	} else if !node.isLeaf {
		// Key not found in this node, recurse into the appropriate child
		if node.children[i].numKeys < b.t {
			b.fill(node, i)
		}
		if i < node.numKeys && key > node.keys[i].Key {
			i++
		}
		b.delete(node.children[i], key)
	}
}

// deleteInternalNode handles deletion of a key in an internal node.
func (b *BTree) deleteInternalNode(node *Node, idx int) {
	t := b.t
	key := node.keys[idx]

	// Case 1: Predecessor child has at least t keys
	if node.children[idx].numKeys >= t {
		pred := b.getPredecessor(node, idx)
		node.keys[idx] = pred
		b.delete(node.children[idx], pred.Key)

	// Case 2: Successor child has at least t keys
	} else if node.children[idx+1].numKeys >= t {
		succ := b.getSuccessor(node, idx)
		node.keys[idx] = succ
		b.delete(node.children[idx+1], succ.Key)

	// Case 3: Both children have fewer than t keys, merge them
	} else {
		b.merge(node, idx)
		b.delete(node.children[idx], key.Key)
	}
}

// getPredecessor finds the predecessor of a key in the B-tree.
func (b *BTree) getPredecessor(node *Node, idx int) *KeyValue {
	current := node.children[idx]
	for !current.isLeaf {
		current = current.children[current.numKeys]
	}
	return current.keys[current.numKeys-1]
}

// getSuccessor finds the successor of a key in the B-tree.
func (b *BTree) getSuccessor(node *Node, idx int) *KeyValue {
	current := node.children[idx+1]
	for !current.isLeaf {
		current = current.children[0]
	}
	return current.keys[0]
}

// merge merges the child at index idx with its sibling.
func (b *BTree) merge(node *Node, idx int) {
	child := node.children[idx]
	sibling := node.children[idx+1]

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
}

// fill ensures that the child node has at least t keys by borrowing or merging from/to its siblings.
func (b *BTree) fill(node *Node, idx int) {
	// If the previous sibling has more than t-1 keys, borrow from it
	if idx != 0 && node.children[idx-1].numKeys >= b.t {
		b.borrowFromPrev(node, idx)
	} else if idx != node.numKeys && node.children[idx+1].numKeys >= b.t {
		// If the next sibling has more than t-1 keys, borrow from it
		b.borrowFromNext(node, idx)
	} else {
		// Merge the child with either its previous or next sibling
		if idx != node.numKeys {
			b.merge(node, idx)
		} else {
			b.merge(node, idx-1)
		}
	}
}

// borrowFromPrev borrows a key from the previous sibling and inserts it into the child.
func (b *BTree) borrowFromPrev(node *Node, idx int) {
	child := node.children[idx]
	sibling := node.children[idx-1]

	// Move the key from the parent down to the child
	child.keys = append([]*KeyValue{node.keys[idx-1]}, child.keys...)
	node.keys[idx-1] = sibling.keys[sibling.numKeys-1]

	// Move the sibling's last child to the child
	if !child.isLeaf {
		child.children = append([]*Node{sibling.children[sibling.numKeys]}, child.children...)
	}

	sibling.numKeys--
	child.numKeys++
}

// borrowFromNext borrows a key from the next sibling and inserts it into the child.
func (b *BTree) borrowFromNext(node *Node, idx int) {
	child := node.children[idx]
	sibling := node.children[idx+1]

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
}
