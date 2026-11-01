# SDK user-cache isolation

Cache.Set and Cache.Get detach each user's Scope slice as well as the outer user list. Mutating caller-owned input after Set or modifying a returned Scope cannot change the stored cache or another reader's result.

Record order, nil versus empty Scope, TTL, zero-TTL behavior and Clear remain unchanged. Set(nil) retains the existing non-nil empty cached-list behavior. Callers must still synchronize concurrent mutations of their own input while Set is reading it.
