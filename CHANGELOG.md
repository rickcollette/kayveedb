# Changelog: kayveedb 
## Current Version: v1.0.3

### NOTE: Please open a github issue if you run into problems. Also - I am happy to review/accept merge requests.

### Overview:

These changes improve the overall performance, especially in environments where frequent reads and writes to the B-tree are required, and offer better customization and flexibility in resource management.

* Introduced LRU (Least Recently Used) cache with configurable size.
* Added support for dirty node handling with automatic flushing to disk.
* Implemented concurrent cache access using sync.Map for thread safety.
* Optimized B-tree operations to check the cache before reading or writing nodes to disk.
* Added a tunable cache size parameter when initializing the B-tree.
* Added LRU eviction strategy for cache management.
* Integrated cache into all B-tree operations (Insert, Update, Delete, Read).
* Improved logging and error handling for cache-related operations.
* Enhanced thread safety for both B-tree and cache operations.

## Differences between Version 1.0.2 and Version 1.0.3

### 1. Cache Implementation
- **Version 1.0.2**: No cache mechanism implemented.
- **Version 1.0.3**: Introduced an LRU (Least Recently Used) cache with configurable cache size.
    - `CacheEntry` struct created to represent cached nodes, including an `offset`, `node`, `element`, and `dirty` flag.
    - `Cache` struct added with an LRU eviction policy and a `flushFn` for persisting dirty nodes.
    - Methods added:
        - `NewCache(size int, flushFn func(offset int64, node *Node) error)`
        - `Get(offset int64) (*Node, bool)`
        - `Put(offset int64, node *Node, dirty bool)`
        - `evict()`
    - **Impact**: The cache improves performance by keeping frequently accessed nodes in memory and supports a configurable size with dirty node handling.

### 2. Cache Size as a Tunable Parameter
- **Version 1.0.2**: No cache, so no cache size parameter.
- **Version 1.0.3**: The cache size is now passed as a parameter when initializing the B-tree.
    - `NewBTree` now accepts an additional `cacheSize` parameter to dynamically set the cache size.
    - **Impact**: The cache size is customizable based on system memory and workload, making the system more flexible.

### 3. Dirty Node Handling in Cache
- **Version 1.0.2**: No cache, so no concept of dirty nodes.
- **Version 1.0.3**: Introduced the `dirty` flag in `CacheEntry` to mark whether a node has unsaved changes, and dirty nodes are flushed to disk before eviction.
    - `CacheEntry` now contains a `dirty` boolean flag.
    - Eviction calls `flushFn` to persist any modified nodes before removal.
    - **Impact**: Improved data consistency by ensuring that modified nodes are persisted to disk before eviction from the cache.

### 4. Concurrent Cache Access
- **Version 1.0.2**: No cache, so no concurrency control for cache.
- **Version 1.0.3**: The cache uses `sync.Map` for concurrent access to the cached entries, with additional `sync.Mutex` to manage the access order list.
    - `Cache.store` is implemented as a `sync.Map` to support concurrent read/write operations.
    - `Cache.mu` is used to synchronize access to the access order list.
    - **Impact**: Enhanced performance in multithreaded environments by allowing concurrent access to cache entries.

### 5. Node Caching in B-tree Operations
- **Version 1.0.2**: Nodes were always read from and written to disk directly.
- **Version 1.0.3**: Before reading a node from disk, the B-tree checks the cache; likewise, written nodes are cached.
    - Modified `readNode` and `writeNode` methods to interact with the cache:
        - `readNode(offset int64) (*Node, error)` checks the cache before reading from disk.
        - `writeNode(node *Node) (int64, error)` adds the written node to the cache, marking it as dirty.
    - **Impact**: Reduced disk I/O by caching nodes in memory and retrieving frequently accessed nodes from the cache.

### 6. Improved Node Flushing Mechanism
- **Version 1.0.2**: No mechanism for explicitly flushing dirty nodes since no cache was implemented.
- **Version 1.0.3**: Introduced `flushFn`, which is invoked when evicting dirty nodes to ensure they are persisted to disk.
    - `flushFn` writes the dirty node to disk before eviction.
    - **Impact**: Ensures data integrity by guaranteeing that modified nodes are not lost during eviction.

### 7. LRU Eviction Strategy
- **Version 1.0.2**: No eviction strategy since no cache.
- **Version 1.0.3**: Implemented an LRU eviction strategy where the least recently used nodes are evicted first when the cache is full.
    - The cache tracks access order using a doubly linked list (`Cache.order`), moving accessed nodes to the front and evicting nodes from the back.
    - **Impact**: Efficient memory usage by keeping recently accessed nodes in memory and evicting less frequently accessed nodes.

### 8. Functionality Changes
- **Version 1.0.2**: No caching, and all B-tree operations involved direct disk reads and writes.
- **Version 1.0.3**: All B-tree operations now utilize the cache to optimize performance.
    - `Insert`, `Update`, `Delete`, `Read`, and internal B-tree functions (`splitChild`, `insertNonFull`, `delete`) now work with the cache, improving performance by reducing disk I/O.

### 9. Logging and Error Handling
- **Version 1.0.2**: Basic error handling and operation logging via `logOperation`.
- **Version 1.0.3**: Logging remains the same, but with added cache-related error handling, especially during node eviction (e.g., handling failures when flushing dirty nodes).
    - Additional logging is introduced during eviction failures.
    - **Impact**: Better transparency and error reporting during node eviction and cache operations.

### 10. Thread-Safe B-tree and Cache
- **Version 1.0.2**: The B-tree operations were protected by `sync.RWMutex` but no cache interactions.
- **Version 1.0.3**: Both B-tree and cache operations are thread-safe, with `sync.RWMutex` protecting B-tree operations and `sync.Mutex`/`sync.Map` managing cache concurrency.
    - **Impact**: The Version 1.0.3 is more robust in multithreaded environments, ensuring safe concurrent access to both B-tree and cache.
