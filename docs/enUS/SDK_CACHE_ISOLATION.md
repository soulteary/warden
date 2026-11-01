# SDK user-cache isolation

Cache.Set and Cache.Get detach each user's Scope slice as well as the outer user list. Mutating caller-owned input after Set or modifying a returned Scope cannot change the stored cache or another reader's result.

Record order, nil versus empty Scope, TTL, zero-TTL behavior and Clear remain unchanged. Set(nil) retains the existing non-nil empty cached-list behavior. Callers must still synchronize concurrent mutations of their own input while Set is reading it.

Hits capture a valid immutable view under a shared lock and copy rows and Scope after unlocking. A hit captured before Clear may finish returning; in-flight network requests started before invalidation remain fenced by the epoch. Misses recheck under the write lock, and response commits retain epoch/sequence protection. Default TTL expires strictly after its boundary; a source-bound deadline expires at equality.
