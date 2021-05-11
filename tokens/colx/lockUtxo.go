package colx

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
)

var lockedUtxos map[utxokey]int = make(map[utxokey]int)
var unlockConds map[utxokey](func() bool) = make(map[utxokey](func() bool))
var utxoLock sync.RWMutex

type utxokey struct {
	txhash string
	vout   int
}

var (
	ErrUtxoLocked = fmt.Errorf("[Locked utxo] Utxo is already locked")
	ErrNotLocked  = fmt.Errorf("[Locked utxo] Utxo is not locked")
)

func (b *Bridge) IsUtxoLocked(txhash string, vout int) bool {
	txhash = strings.ToUpper(txhash)
	key := utxokey{txhash: txhash, vout: vout}
	return b.isUtxoLocked(key)
}

func (b *Bridge) isUtxoLocked(key utxokey) bool {
	utxoLock.RLock()
	defer utxoLock.RUnlock()
	if lockedUtxos[key] == 1 {
		return true
	}
	return false
}

// defaultUnlockCond unlock utxo after 5 days
var defaultUnlockCond = after(3600 * 120)

func (b *Bridge) LockUtxo(txhash string, vout int) error {
	// Use default unlock cond
	return b.LockUtxoWithCond(txhash, vout, defaultUnlockCond)
}

func (b *Bridge) LockUtxoWithCond(txhash string, vout int, cond func() bool) error {
	txhash = strings.ToUpper(txhash)
	key := utxokey{txhash: txhash, vout: vout}
	return b.lockUtxo(key, cond)
}

func after(seconds int64) func() bool {
	deadline := time.Now().Unix() + seconds
	return func() bool {
		if time.Now().Unix() >= deadline {
			return true
		}
		return false
	}
}

func (b *Bridge) lockUtxo(key utxokey, cond func() bool) error {
	if b.isUtxoLocked(key) {
		return ErrUtxoLocked
	}
	utxoLock.Lock()
	defer utxoLock.Unlock()
	lockedUtxos[key] = 1
	unlockConds[key] = cond
	return nil
}

func (b *Bridge) UnlockUtxo(txhash string, vout int) {
	txhash = strings.ToUpper(txhash)
	key := utxokey{txhash: txhash, vout: vout}
	b.unlockUtxo(key)
}

func (b *Bridge) unlockUtxo(key utxokey) {
	utxoLock.Lock()
	defer utxoLock.Unlock()
	delete(lockedUtxos, key)
	delete(unlockConds, key)
}

func (b *Bridge) SetUnlockUtxoCond(txhash string, vout int, cond func() bool) error {
	txhash = strings.ToUpper(txhash)
	key := utxokey{txhash: txhash, vout: vout}
	return b.setUnlockUtxoCond(key, cond)
}

func (b *Bridge) setUnlockUtxoCond(key utxokey, cond func() bool) error {
	utxoLock.Lock()
	defer utxoLock.Lock()
	if b.isUtxoLocked(key) == false {
		return ErrNotLocked
	}
	unlockConds[key] = cond
	return nil
}

func (b *Bridge) StartMonitLockedUtxo() {
	log.Info("Start monit locked utxo")
	for {
		log.Debug("Check locked utxos", "number locked", len(lockedUtxos))
		for key, cond := range unlockConds {
			if cond() == true {
				log.Debug("Unlock utxo", "key", key)
				b.unlockUtxo(key)
			}
		}
		time.Sleep(time.Second * 60)
	}
}