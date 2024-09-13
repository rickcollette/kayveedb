
# kayveedb Documentation

## NOTE: Please open a github issue if you run into problems.  Also - I am happy to review/accept merge requests.

## Overview

`kayveedb` is a Go package that provides a B-tree-based key-value database with XChaCha20 encryption for both in-memory and at-rest data, as well as AES-256 for secure key hashing. It implements log-based persistence, where only changes are logged and applied later for efficiency.

This package is suitable for environments requiring both security and performance, ensuring that both in-memory and at-rest data are encrypted. Functions in this package require the encryption key and nonce to be provided by the calling application, allowing for flexible key management.

## Features

- **XChaCha20 encryption**: For in-memory and at-rest data.
- **AES-256 HMAC hashing**: For secure key management.
- **B-tree data structure**: Efficient data storage and retrieval.
- **Log-based persistence**: Logs changes and applies them to the tree later.

## Package Import

```go
import "github.com/rickcollette/kayveedb"
```

## Usage

### Initialization

To create a new B-tree:

```go
// Create a new B-tree with a minimum degree `t`, snapshot file, operation log file,
// HMAC key, encryption key, and nonce for encryption.
btree, err := kayveedb.NewBTree(t int, snapshot string, logPath string, hmacKey []byte, encryptionKey []byte, nonce []byte)
if err != nil {
    log.Fatalf("Failed to create B-tree: %v", err)
}
```

### Inserting Key-Value Pairs

To insert a key-value pair into the B-tree:

```go
key := "myKey"
value := []byte("myValue")
encryptionKey := []byte("32-byte-long-encryption-key")
nonce := []byte("24-byte-nonce")

err := btree.Insert(key, value, encryptionKey, nonce)
if err != nil {
    log.Fatalf("Failed to insert key-value: %v", err)
}
```

### Reading Values

To read a value from the B-tree:

```go
decryptedValue, err := btree.Read(key, encryptionKey, nonce)
if err != nil {
    log.Fatalf("Failed to read key: %v", err)
}
fmt.Printf("Decrypted value: %s", decryptedValue)
```

### Updating Values

To update a key-value pair in the B-tree:

```go
newValue := []byte("updatedValue")

err := btree.Update(key, newValue, encryptionKey, nonce)
if err != nil {
    log.Fatalf("Failed to update key-value: %v", err)
}
```

### Deleting Key-Value Pairs

To delete a key-value pair from the B-tree:

```go
err := btree.Delete(key)
if err != nil {
    log.Fatalf("Failed to delete key: %v", err)
}
```

### Snapshot and Persistence

To create a snapshot of the current B-tree state:

```go
err := btree.Snapshot()
if err != nil {
    log.Fatalf("Failed to create snapshot: %v", err)
}
```

## API Documentation

### func NewBTree

```go
func NewBTree(t int, snapshot string, logPath string, hmacKey []byte, encryptionKey []byte, nonce []byte) (*BTree, error)
```

Creates a new B-tree with the specified parameters. It requires the minimum degree (`t`), snapshot file, operation log path, and the encryption key and nonce.

#### Example:

```go
btree, err := kayveedb.NewBTree(3, "snapshot.btree", "operation.log", hmacKey, encryptionKey, nonce)
if err != nil {
    log.Fatalf("Failed to initialize B-tree: %v", err)
}
```

### func (b *BTree) Insert

```go
func (b *BTree) Insert(key string, value, encryptionKey, nonce []byte) error
```

Inserts a new key-value pair into the B-tree. The value is encrypted using the provided `encryptionKey` and `nonce`.

#### Example:

```go
err := btree.Insert("myKey", []byte("myValue"), encryptionKey, nonce)
if err != nil {
    log.Fatalf("Failed to insert key-value pair: %v", err)
}
```

### func (b *BTree) Read

```go
func (b *BTree) Read(key string, encryptionKey, nonce []byte) ([]byte, error)
```

Reads and decrypts the value associated with the given `key` using the provided `encryptionKey` and `nonce`.

#### Example:

```go
decryptedValue, err := btree.Read("myKey", encryptionKey, nonce)
if err != nil {
    log.Fatalf("Failed to read key: %v", err)
}
fmt.Printf("Decrypted value: %s", decryptedValue)
```

### func (b *BTree) Update

```go
func (b *BTree) Update(key string, value, encryptionKey, nonce []byte) error
```

Updates an existing key-value pair in the B-tree. The value is encrypted using the provided `encryptionKey` and `nonce`.

#### Example:

```go
err := btree.Update("myKey", []byte("newValue"), encryptionKey, nonce)
if err != nil {
    log.Fatalf("Failed to update key-value pair: %v", err)
}
```

### func (b *BTree) Delete

```go
func (b *BTree) Delete(key string) error
```

Deletes the key-value pair associated with the given `key`.

#### Example:

```go
err := btree.Delete("myKey")
if err != nil {
    log.Fatalf("Failed to delete key: %v", err)
}
```

### func (b *BTree) Snapshot

```go
func (b *BTree) Snapshot() error
```

Creates a snapshot of the current B-tree structure, saving it to disk.

#### Example:

```go
err := btree.Snapshot()
if err != nil {
    log.Fatalf("Failed to create snapshot: %v", err)
}
```

### func (b *BTree) logOperation

```go
func (b *BTree) logOperation(op, key string, value []byte)
```

Logs an operation (`CREATE`, `UPDATE`, or `DELETE`) for persistence.

### func (b *BTree) loadLog

```go
func (b *BTree) loadLog(encryptionKey, nonce []byte) error
```

Loads the operation log from disk and replays the operations to restore the tree to its most recent state.

### func (b *BTree) loadSnapshot

```go
func (b *BTree) loadSnapshot() error
```

Loads the B-tree from the snapshot file if it exists.

## Internal Functions

These are internal functions used to manage B-tree nodes and encryption:

- `encrypt(data, encryptionKey, nonce []byte) ([]byte, error)`
- `decrypt(data, encryptionKey, nonce []byte) ([]byte, error)`
- `hashKey(key string) string`
- `splitChild(parent *Node, i int, fullChild *Node)`
- `insertNonFull(node *Node, kv *KeyValue)`
- `search(node *Node, key string) *KeyValue`
- `delete(node *Node, key string)`
- `merge(node *Node, idx int)`
- `fill(node *Node, idx int)`
- `borrowFromPrev(node *Node, idx int)`
- `borrowFromNext(node *Node, idx int)`

## License

This package is licensed under the MIT License.
