package db

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"path"
	"strings"
	"sync"
)

var (
	BKTRetry = []byte("retry")
)

type RetryDB struct {
	rwlock        *sync.RWMutex
	db            *bolt.DB
	dbPath        string
	retryDuration int
	retryTimes    []byte
}

func NewRetryDB(filePath string, times, retryDuration int) (*RetryDB, error) {
	if !strings.Contains(filePath, ".bin") {
		filePath = path.Join(filePath, "retry.bin")
	}
	if times < 0 {
		return nil, fmt.Errorf("retry time must greater than or equal to 0, yours %d", times)
	}
	if retryDuration <= 0 {
		return nil, fmt.Errorf("retry duration must greater than 0, yours %d", retryDuration)
	}

	r := new(RetryDB)
	db, err := bolt.Open(filePath, 0644, &bolt.Options{InitialMmapSize: 500000})
	if err != nil {
		return nil, err
	}

	r.db = db
	r.rwlock = new(sync.RWMutex)
	r.dbPath = filePath
	r.retryDuration = retryDuration
	r.retryTimes = make([]byte, 2)
	binary.LittleEndian.PutUint16(r.retryTimes, uint16(times))

	if err = db.Update(func(btx *bolt.Tx) error {
		_, err := btx.CreateBucketIfNotExists(BKTRetry)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *RetryDB) Put(tx string) error {
	r.rwlock.Lock()
	defer r.rwlock.Unlock()

	txb, err := hex.DecodeString(tx)
	if err != nil {
		return err
	}

	return r.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BKTRetry)
		err := bucket.Put(txb, r.retryTimes)
		if err != nil {
			return err
		}
		return nil
	})
}

func (r *RetryDB) GetAll() ([]string, error) {
	mtxArr := make([]string, 0)
	var err error
	if binary.LittleEndian.Uint16(r.retryTimes) > 0 {
		r.rwlock.Lock()
		defer r.rwlock.Unlock()

		err = r.db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(BKTRetry)
			valArr := make([]uint16, 0)
			err := bucket.ForEach(func(k, v []byte) error {
				mtxArr = append(mtxArr, hex.EncodeToString(k))
				valArr = append(valArr, binary.LittleEndian.Uint16(v)-1)
				return nil
			})
			if err != nil {
				return err
			}
			for i, mtx := range mtxArr {
				k, _ := hex.DecodeString(mtx)
				if valArr[i] <= 0 {
					err := bucket.Delete(k)
					if err != nil {
						return err
					}
				} else {
					val := make([]byte, 2)
					binary.LittleEndian.PutUint16(val, valArr[i])
					err := bucket.Put(k, val)
					if err != nil {
						return err
					}
				}
			}

			return nil
		})
	} else {
		r.rwlock.RLock()
		defer r.rwlock.RUnlock()

		err = r.db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(BKTRetry)
			err := bucket.ForEach(func(k, v []byte) error {
				mtxArr = append(mtxArr, hex.EncodeToString(k))
				return nil
			})
			if err != nil {
				return err
			}
			return nil
		})
	}
	if err != nil {
		return nil, err
	}
	if len(mtxArr) == 0 {
		return nil, errors.New("no tx in db")
	}

	return mtxArr, nil
}

func (r *RetryDB) Del(k string) error {
	r.rwlock.Lock()
	defer r.rwlock.Unlock()

	kb, err := hex.DecodeString(k)
	if err != nil {
		return err
	}
	return r.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BKTRetry)
		err := bucket.Delete(kb)
		if err != nil {
			return err
		}
		return nil
	})
}
