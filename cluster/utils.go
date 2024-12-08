package cluster

import (
	"crypto/sha1"
	"encoding/hex"
	"math/big"
	"strings"
)

// use hex encoding for smaller storage size
func Hash(input string) string {
	hasher := sha1.New()
	hasher.Write([]byte(input))
	return hex.EncodeToString(hasher.Sum(nil))
}

func getXORDistance(id1, id2 string) *big.Int {
	id1Int := new(big.Int)
	id1Int.SetString(id1, 16)

	id2Int := new(big.Int)
	id2Int.SetString(id2, 16)

	distance := new(big.Int)
	distance.Xor(id1Int, id2Int)
	return distance
}

func getBucketIndex(distance *big.Int) int {
	bitLength := distance.BitLen()
	return bits - 1 - bitLength
}

func AddressToHostnamePort(address string) (string, string) {
	addr := strings.Split(address, ":")
	return addr[0], addr[1]
}

func HostnamePortToAddress(hostname, port string) string {
	return hostname + ":" + port
}

func CalculateWeight(key, nodeId string) *big.Int {
	combined := key + nodeId

	hashedCombined := Hash(combined)

	weight := new(big.Int)
	weight.SetString(hashedCombined, 16)
	return weight
}
