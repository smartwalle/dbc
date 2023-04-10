package dbc

import "math/rand"

func DJBSharding() func(key string) uint32 {
	var seed = rand.Uint32() + 5381
	return func(key string) uint32 {
		var (
			l    = uint32(len(key))
			hash = seed + l
			i    = uint32(0)
		)
		if l >= 4 {
			for i < l-4 {
				hash = (hash * 33) ^ uint32(key[i])
				hash = (hash * 33) ^ uint32(key[i+1])
				hash = (hash * 33) ^ uint32(key[i+2])
				hash = (hash * 33) ^ uint32(key[i+3])
				i += 4
			}
		}
		switch l - i {
		case 1:
		case 2:
			hash = (hash * 33) ^ uint32(key[i])
		case 3:
			hash = (hash * 33) ^ uint32(key[i])
			hash = (hash * 33) ^ uint32(key[i+1])
		case 4:
			hash = (hash * 33) ^ uint32(key[i])
			hash = (hash * 33) ^ uint32(key[i+1])
			hash = (hash * 33) ^ uint32(key[i+2])
		}
		return (hash ^ (hash >> 16)) % ShardCount
	}
}
