package commons

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestExecutionContext(t *testing.T) {
	nonceLimit := int64(1000000000000)
	target := getTarget(uint32(20))
	leadingZeros := "00000"
	data := []byte("my dummy data")

	verify := func(d interface{}) bool {
		return isBelowTarget(d.([]byte), target)
	}
	exec := func(i int, ec ExecutionContext) error {
		var nonce int64
		for nonce < nonceLimit && !ec.Ended() {
			raw := bytes.Join([][]byte{
				data[:],
				[]byte(strconv.FormatInt(nonce, 10)),
				[]byte(strconv.FormatInt(time.Now().Unix(), 10)),
			}, []byte{})
			hashed := hash(raw)
			ec.End(hashed)
			nonce++
		}
		return nil
	}

	resChan, errChan := Run(exec, verify, 3)
	defer func() {
		close(resChan)
		close(errChan)
	}()

	select {
	case err := <- errChan:
		t.Fatal("ended with error:", err)
	case res := <- resChan:
		if !verify(res.([]byte)) {
			t.Fatal("should end only once verified")
		}
		str := fmt.Sprintf("%x", res)
		if strings.Index(str, leadingZeros) != 0 {
			t.Fatal("should have a minimum of leading zeros")
		}
	}
}

func TestExecutionContext_ExecError(t *testing.T) {
	verify := func(d interface{}) bool {
		return true
	}
	exec := func(i int, ec ExecutionContext) error {
		for i := 0; i < 10000000; i++ {}
		return errors.New("dummy error")
	}

	resChan, errChan := Run(exec, verify, 3)
	defer func() {
		close(resChan)
		close(errChan)
	}()

	select {
	case err := <- errChan:
		if err == nil {
			t.Fatal("should end with some error")
		}
	case <- resChan:
		t.Fatal("should have ended with error")
	}
}

func isBelowTarget(hash []byte, t *big.Int) bool {
	var hashInt big.Int
	hashInt.SetBytes(hash[:])
	return hashInt.Cmp(t) == -1
}

func getTarget(targetBits uint32) *big.Int {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))
	return target
}

func hash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}