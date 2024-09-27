
# kayveedb Go Package Documentation

## Current version: **v1.2.0**

### Overview

`kayveedb` is a robust Go package that implements a B-Tree-based key-value store with advanced features such as caching support, transaction management, publish-subscribe mechanisms, user authentication, and support for various data structures like lists, sets, hashes, and sorted sets. The package leverages an LRU (Least Recently Used) eviction policy for caching and ensures data security through ChaCha20 encryption and HMAC for key hashing.

### Highlights of Version v1.2.0

* **Protocol Management:** Enhanced client-server communication protocols.
* **Advanced Caching:** Extended cache operations with concurrency support.
* **Transaction Management:** Support for complex transactions including lists, sets, hashes, and sorted sets.
* **Client Management:** Efficient tracking and management of active clients.
* **Database Management:** Ability to handle multiple databases with ease.
* **Publish-Subscribe System:** Real-time message broadcasting and subscription mechanisms.
* **Extended Data Structures:** Support for lists, sets, hashes, and sorted sets.
* **User Authentication:** Comprehensive user management and authentication system.

### Table of Contents
- [Installation](#installation)
- [Package Contents](#package-contents)
  - [Constants](#constants)
  - [Functions](#functions)
- [Cache](#cache)
- [BTree](#btree)
- [Protocol](#protocol)
- [Transactions](#transactions)
- [Clients](#clients)
- [Database Management](#database-management)
- [Publish-Subscribe](#publish-subscribe)
- [Data Structures](#data-structures)
- [Authentication](#authentication)
- [Encryption and HMAC](#encryption-and-hmac)
- [Usage Examples](#usage-examples)
- [Error Handling](#error-handling)
- [License](#license)

----

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

## Cache

### CacheEntry

The `CacheEntry` struct stores the node in the cache and its access order.

**Fields:**

- `offset int64`: The node's offset in the file.
- `node *Node`: The actual node.
- `element *list.Element`: The position in the access order list.
- `dirty bool`: Whether the node has unsaved changes.

### CacheManager

`CacheManager` extends the cache operations, providing thread-safe methods to manage cache entries.

**Fields:**

- `cache *Cache`: The underlying cache instance.
- `mu sync.Mutex`: Mutex to protect cache operations.

**Methods:**

- `SetCache(key string, value []byte)`: Adds a key-value pair to the cache.
- `GetCache(key string) (*Node, error)`: Retrieves a value from the cache.
- `DeleteCache(key string) error`: Deletes a key from the cache.
- `FlushCache()`: Removes all entries from the cache.
- `SetCacheSize(size int)`: Adjusts the cache size.
- `GetCacheSize() int`: Returns the current cache size.
- `SetCachePolicy(policy string) error`: Sets the cache eviction policy (e.g., LRU, LFU).

**Example:**
```go
cacheManager := kayveedb.NewCacheManager(100, flushFn)
cacheManager.SetCache("mykey", []byte("myvalue"))
node, err := cacheManager.GetCache("mykey")
if err != nil {
    log.Fatal(err)
}
fmt.Println("Cached Node:", node)
```

## BTree

The `BTree` struct implements a B-Tree with caching and transaction support. It uses the cache for efficient node storage and retrieval.

**Fields:**

- `root *Node`: The root node of the B-Tree.
- `t int`: The minimum degree of the B-Tree.
- `dbPath string`: Path to the database file.
- `dbName string`: Name of the database file.
- `logName string`: Path to the operation log file.
- `dbFile *os.File`: Database file handle.
- `logFile *os.File`: Operation log file handle.
- `hmacKey []byte`: Key for HMAC hashing.
- `mu sync.RWMutex`: Read-write mutex for safe concurrent access.
- `cache *Cache`: LRU cache for storing nodes.
- `clients *ClientManager`: Manages active clients.

**Methods:**

### `NewBTree`

Creates a new B-Tree with a configurable cache size.
The database files are either the names provided or will default to `$CWD/kayvee.db` and `$CWD/kayvee.log`.

**Signature:**
```go
func NewBTree(t int, dbPath, dbName, logName string, hmacKey, encryptionKey, nonce []byte, cacheSize int) (*BTree, error)
```
**Parameters:**

- `t int`: Minimum degree of the B-Tree.
- `dbPath string`: Path to the database file.
- `dbName string`: Name of the database file.
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

### `Insert`

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

### `Update`

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

### `Delete`

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

### `Read`

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

### `Close`

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
## Protocol

The protocol package manages client-server communication, defining command types, status codes, and packet serialization/deserialization mechanisms.

**Types**
- **CommandType**: Represents the type of command.
- **StatusCode**: Represents the status of the response.
- **Packet**: Represents a protocol packet.
- **Response**: Represents a response packet.
- **Subscriber**: Channel type for Pub/Sub.

**Functions**
- `InitBTree(...)`: Initializes the global BTree instance.
- `HandleClientConnect(clientID uint32)`: Handles client connections.
- `HandleClientDisconnect(clientID uint32)`: Handles client disconnections.
- `SetMaxPayloadSize(size uint32)`: Sets the maximum payload size.
- `GetMaxPayloadSize() uint32`: Retrieves the current maximum payload size.
- `SerializePacket(p Packet) ([]byte, error)`: Serializes a Packet into bytes.
- `DeserializeResponse(reader io.Reader) (Response, error)`: Deserializes bytes into a Response.

**Command Types**
```go
type CommandType byte

const (
    CommandAuth        CommandType = 0x01
    CommandInsert      CommandType = 0x02
    CommandUpdate      CommandType = 0x03
    CommandDelete      CommandType = 0x04
    CommandRead        CommandType = 0x05
    CommandBeginTx     CommandType = 0x06
    CommandCommitTx    CommandType = 0x07
    CommandRollbackTx  CommandType = 0x08
    CommandSetCache    CommandType = 0x09
    CommandGetCache    CommandType = 0x0A
    CommandDeleteCache CommandType = 0x0B
    CommandFlushCache  CommandType = 0x0C
    CommandPublish     CommandType = 0x0D
    CommandSubscribe   CommandType = 0x0E
    CommandConnect     CommandType = 0x0F
    CommandDisconnect  CommandType = 0x10
)
```

**Status Codes**
```go
type StatusCode uint32

const (
    StatusSuccess        StatusCode = 0x00
    StatusError          StatusCode = 0x01
    StatusTxBegin        StatusCode = 0x02
    StatusTxCommit       StatusCode = 0x03
    StatusTxRollback     StatusCode = 0x04
    StatusClientAdded    StatusCode = 0x05
    StatusClientRemoved  StatusCode = 0x06
)
```

**Packet Structure**
```go
type Packet struct {
    CommandID   uint32
    CommandType CommandType
    Payload     []byte
}

type Response struct {
    CommandID uint32
    Status    StatusCode
    Data      string
}
```

### Serialization and Deserialization

- **SerializePacket**: Converts a `Packet` struct into a byte slice for transmission.
- **DeserializeResponse**: Converts a byte stream into a `Response` struct, ensuring payload size does not exceed the maximum allowed.

**Example:**
```go
packet := kayveedb.Packet{
    CommandID:   1,
    CommandType: kayveedb.CommandInsert,
    Payload:     []byte("sample payload"),
}

serialized, err := kayveedb.SerializePacket(packet)
if err != nil {
    log.Fatal(err)
}

response, err := kayveedb.DeserializeResponse(bytes.NewReader(serialized))
if err != nil {
    log.Fatal(err)
}

fmt.Println("Response:", response)
```

---

## Transactions

The transactions package provides comprehensive transaction management, supporting operations like lists, sets, hashes, and sorted sets within transactions.

**Types**
- **Transaction**: Represents a single transaction.
- **TransactionManager**: Manages multiple transactions concurrently.

**Functions**
- `NewTransactionManager() *TransactionManager`: Initializes a new TransactionManager.
- `Begin(txID uint32)`: Begins a new transaction.
- `AddOperation(txID uint32, operation func() error) error`: Adds a generic operation to a transaction.
- `AddListOperation(txID uint32, listOp func() error) error`: Adds a list-specific operation.
- `AddSetOperation(txID uint32, setOp func() error) error`: Adds a set-specific operation.
- `Commit(txID uint32) error`: Commits a transaction, executing all operations.
- `Rollback(txID uint32) error`: Rolls back a transaction, discarding all operations.

**Example:**
```go
tm := kayveedb.NewTransactionManager()
txID := uint32(1)

tm.Begin(txID)

tm.AddOperation(txID, func() error {
    return tree.Insert("key1", []byte("value1"), encryptionKey, nonce)
})

tm.AddListOperation(txID, func() error {
    return listManager.LPush("mylist", "value1")
})

if err := tm.Commit(txID); err != nil {
    log.Fatalf("Transaction failed: %v", err)
}
```

---

## Clients

The clients package manages active client connections, allowing for efficient tracking and management.

**Types**
- **ClientManager**: Manages active clients.

**Functions**
- `NewClientManager() *ClientManager`: Initializes a new ClientManager.
- `AddClient(clientID uint32)`: Adds a new client to the active client list.
- `RemoveClient(clientID uint32)`: Removes a client from the active client list.
- `GetActiveClientCount() int`: Retrieves the count of active clients.

**Example:**
```go
clientManager := kayveedb.NewClientManager()
clientManager.AddClient(1001)
clientManager.AddClient(1002)

fmt.Println("Active Clients:", clientManager.GetActiveClientCount())

clientManager.RemoveClient(1001)
fmt.Println("Active Clients after removal:", clientManager.GetActiveClientCount())
```

---

## Database Management

The manage package provides tools to handle multiple databases, allowing for creation, deletion, and switching between databases.

**Types**
- **DatabaseManager**: Manages multiple databases.

**Functions**
- `NewDatabaseManager(basePath string) *DatabaseManager`: Initializes a DatabaseManager with a base path.
- `CreateDatabase(dbName string) error`: Creates a new database directory and files.
- `DropDatabase(dbName string) error`: Removes a database directory and files.
- `UseDatabase(dbName string) error`: Sets the current database to be used.
- `ShowDatabases() ([]string, error)`: Lists all databases in the base path.
- `CurrentDatabase() string`: Returns the name of the currently used database.
- `GetDatabasePath() string`: Returns the full path to the current database.

**Example:**
```go
dbManager := kayveedb.NewDatabaseManager("/path/to/databases")

// Create a new database
err := dbManager.CreateDatabase("newdb")
if err != nil {
    log.Fatal(err)
}

// Switch to the new database
err = dbManager.UseDatabase("newdb")
if err != nil {
    log.Fatal(err)
}

// List all databases
databases, err := dbManager.ShowDatabases()
if err != nil {
    log.Fatal(err)
}
fmt.Println("Databases:", databases)
```

---

## Publish-Subscribe

The pubsub package implements a publish-subscribe system, enabling real-time message broadcasting and subscription.

**Types**
- **Subscriber**: A channel type for receiving messages.
- **PubSub**: Manages channels and subscribers.

**Functions**
- `NewPubSub() *PubSub`: Initializes a new PubSub instance.
- `Publish(channel, message string)`: Publishes a message to a specific channel.
- `Subscribe(channel string) Subscriber`: Subscribes to a specific channel.
- `Unsubscribe(channel string, sub Subscriber)`: Unsubscribes from a specific channel.

**Example:**
```go
pubsub := kayveedb.NewPubSub()

// Subscriber 1
sub1 := pubsub.Subscribe("updates")
go func() {
    for msg := range sub1 {
        fmt.Println("Subscriber 1 received:", msg)
    }
}()

// Subscriber 2
sub2 := pubsub.Subscribe("updates")
go func() {
    for msg := range sub2 {
        fmt.Println("Subscriber 2 received:", msg)
    }
}()

// Publish messages
pubsub.Publish("updates", "First Update")
pubsub.Publish("updates", "Second Update")

// Unsubscribe Subscriber 1
pubsub.Unsubscribe("updates", sub1)

// Publish another message
pubsub.Publish("updates", "Third Update")
```
## Data Structures

The `datastructures` package provides support for various data structures, enabling complex data manipulations within the B-Tree.

### ListManager

Manages list data structures with operations like push and range retrieval.

**Fields:**
- `lists map[string][]string`: Stores the list data.
- `mu sync.Mutex`: Mutex for thread-safe operations.

**Methods:**
- `NewListManager() *ListManager`: Initializes a new `ListManager`.
- `LPush(key, value string)`: Pushes a value to the left of the list.
- `RPush(key, value string)`: Pushes a value to the right of the list.
- `LRange(key string, start, stop int) ([]string, error)`: Retrieves a range of values from the list.

**Example:**
```go
listManager := kayveedb.NewListManager()
listManager.LPush("mylist", "value1")
listManager.RPush("mylist", "value2")

values, err := listManager.LRange("mylist", 0, 2)
if err != nil {
    log.Fatal(err)
}
fmt.Println("List values:", values)
```

---

### SetManager

Manages set data structures with operations like adding members and retrieving all members.

**Fields:**
- `sets map[string]map[string]bool`: Stores the set data.
- `mu sync.Mutex`: Mutex for thread-safe operations.

**Methods:**
- `NewSetManager() *SetManager`: Initializes a new `SetManager`.
- `SAdd(key, member string)`: Adds a member to the set.
- `SMembers(key string) ([]string, error)`: Retrieves all members of the set.

**Example:**
```go
setManager := kayveedb.NewSetManager()
setManager.SAdd("myset", "member1")
setManager.SAdd("myset", "member2")

members, err := setManager.SMembers("myset")
if err != nil {
    log.Fatal(err)
}
fmt.Println("Set members:", members)
```

---

### HashManager

Manages hash data structures with operations like setting and getting field values.

**Fields:**
- `hashes map[string]map[string]string`: Stores the hash data.
- `mu sync.Mutex`: Mutex for thread-safe operations.

**Methods:**
- `NewHashManager() *HashManager`: Initializes a new `HashManager`.
- `HSet(key, field, value string)`: Sets a field in the hash.
- `HGet(key, field string) (string, error)`: Retrieves a field value from the hash.

**Example:**
```go
hashManager := kayveedb.NewHashManager()
hashManager.HSet("myhash", "field1", "value1")

value, err := hashManager.HGet("myhash", "field1")
if err != nil {
    log.Fatal(err)
}
fmt.Println("Hash field value:", value)
```

---

### ZSetManager

Manages sorted set (zset) data structures with operations like adding members with scores and retrieving ranges based on scores.

**Fields:**
- `zsets map[string]map[string]float64`: Stores the zset data.
- `mu sync.Mutex`: Mutex for thread-safe operations.

**Methods:**
- `NewZSetManager() *ZSetManager`: Initializes a new `ZSetManager`.
- `ZAdd(key, member string, score float64)`: Adds a member with a score to the zset.
- `ZRange(key string, start, stop int) ([]string, error)`: Retrieves a range of members from the zset based on scores.

**Example:**
```go
zsetManager := kayveedb.NewZSetManager()
zsetManager.ZAdd("myzset", "member1", 1.0)
zsetManager.ZAdd("myzset", "member2", 2.0)

members, err := zsetManager.ZRange("myzset", 0, 2)
if err != nil {
    log.Fatal(err)
}
fmt.Println("ZSet members:", members)
```

---

## Authentication

The `auth` package handles user authentication and session management, ensuring secure access to the database.

**Types**
- **User**: Represents a user in the system.
- **AuthManager**: Handles user authentication and session management.

**Functions**
- `NewAuthManager() *AuthManager`: Initializes an `AuthManager`.
- `CreateUser(username, password string) error`: Adds a new user to the system.
- `AlterUser(username, newPassword string) error`: Changes a user's password.
- `DropUser(username string) error`: Removes a user from the system.
- `Grant(username, role string) error`: Grants a role/privilege to a user.
- `Revoke(username, role string) error`: Revokes a role/privilege from a user.
- `Connect(username, password string) error`: Verifies user credentials and establishes a session.
- `Disconnect(username string)`: Ends a user's session.

**Example:**
```go
authManager := kayveedb.NewAuthManager()

// Create a new user
err := authManager.CreateUser("john_doe", "securepassword")
if err != nil {
    log.Fatal(err)
}

// Grant a role to the user
err = authManager.Grant("john_doe", "admin")
if err != nil {
    log.Fatal(err)
}

// Authenticate and connect the user
err = authManager.Connect("john_doe", "securepassword")
if err != nil {
    log.Fatal(err)
}

// Disconnect the user
authManager.Disconnect("john_doe")
```

---

## Encryption and HMAC

The `kayveedb` package ensures data security through encryption and hashing mechanisms.

**Encryption**: Utilizes `chacha20poly1305` from the `golang.org/x/crypto` package for encrypting stored values.
**HMAC**: Employs HMAC with SHA-256 for hashing keys, ensuring data integrity and security.

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

---

### HMAC Functions

#### `hashKey`

Hashes a key using HMAC with SHA-256.

**Signature:**
```go
func (b *BTree) hashKey(key string) string
```
## Usage Examples

### Example 1: Listing Keys in the BTree

This example demonstrates how to initialize a B-Tree and list its keys.

```go
package main

import (
    "fmt"
    "log"
    "kayveedb" // Assuming your package is called kayveedb
)

func main() {
    // Initialize your B-tree with a minimum degree
    bt, err := kayveedb.NewBTree(3, "./", "testdb", "testlog", hmacKey, encryptionKey, nonce, 100)
    if err != nil {
        log.Fatalf("Failed to initialize B-tree: %v", err)
    }

    // Insert some keys into the BTree
    bt.Insert("key1", []byte("value1"), encryptionKey, nonce)
    bt.Insert("key2", []byte("value2"), encryptionKey, nonce)
    bt.Insert("key3", []byte("value3"), encryptionKey, nonce)

    // List all keys in the BTree
    keys, err := bt.ListKeys()
    if err != nil {
        log.Fatalf("Failed to list keys: %v", err)
    }

    // Print the keys
    fmt.Println("Keys in the B-tree:", keys)
}
```

---

### Example 2: Handling Empty Trees

This example shows how to handle listing keys from an empty B-Tree.

```go
package main

import (
    "fmt"
    "log"
    "kayveedb"
)

func main() {
    // Initialize an empty B-tree
    bt, err := kayveedb.NewBTree(3, "./", "testdb", "testlog", hmacKey, encryptionKey, nonce, 100)
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

---

### Example 3: Using Publish-Subscribe

A demonstration of how to set up a publish-subscribe system with multiple subscribers.

```go
package main

import (
    "fmt"
    "kayveedb"
)

func main() {
    pubsub := kayveedb.NewPubSub()

    // Subscriber 1
    sub1 := pubsub.Subscribe("news")
    go func() {
        for msg := range sub1 {
            fmt.Println("Subscriber 1 received:", msg)
        }
    }()

    // Subscriber 2
    sub2 := pubsub.Subscribe("news")
    go func() {
        for msg := range sub2 {
            fmt.Println("Subscriber 2 received:", msg)
        }
    }()

    // Publish messages
    pubsub.Publish("news", "Breaking News!")
    pubsub.Publish("news", "Latest Updates")

    // Unsubscribe Subscriber 1
    pubsub.Unsubscribe("news", sub1)

    // Publish another message
    pubsub.Publish("news", "More News")
}
```

---

### Example 4: Managing Users and Authentication

A guide on how to create, manage, and authenticate users within `kayveedb`.

```go
package main

import (
    "fmt"
    "kayveedb"
    "log"
)

func main() {
    authManager := kayveedb.NewAuthManager()

    // Create a new user
    err := authManager.CreateUser("alice", "password123")
    if err != nil {
        log.Fatal(err)
    }

    // Grant admin role to Alice
    err = authManager.Grant("alice", "admin")
    if err != nil {
        log.Fatal(err)
    }

    // Authenticate and connect Alice
    err = authManager.Connect("alice", "password123")
    if err != nil {
        log.Fatal(err)
    }

    // Disconnect Alice
    authManager.Disconnect("alice")
}
```

---

## Error Handling

The `kayveedb` package employs robust error handling across all its functionalities. It is essential to handle errors appropriately to ensure the reliability and stability of your application.

**Example Error Handling:**
```go
keys, err := tree.ListKeys()
if err != nil {
    log.Fatalf("Error while listing keys: %v", err)
}
```

**Best Practices:**
- **Check Errors Immediately:** Always check for errors immediately after a function call that returns an error.
- **Handle Specific Errors:** Where possible, handle specific error types to provide more granular control.
- **Logging:** Log errors with sufficient context to aid in debugging.
- **Graceful Degradation:** Design your application to handle errors gracefully without crashing unexpectedly.

---

## License

This package is licensed under the MIT License.

---

## Additional Information

For more detailed information, including advanced configurations, optimization tips, and contribution guidelines, please refer to the GitHub repository or contact the maintainers directly.
