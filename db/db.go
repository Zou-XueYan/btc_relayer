package db

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/ontio/btcrelayer/log"
	"path"
	"strings"
	"sync"
)

var (
	BKTRetry           = []byte("retry")
	BKTBtcLastHeight   = []byte("btclast")
	BKTAlliaLastHeight = []byte("allialast")
	KEYBtcLastHeight   = []byte("btclast")
	KEYAlliaLastHeight = []byte("allialast")
)

type RetryDB struct {
	rwlock        *sync.RWMutex
	db            *bolt.DB
	dbPath        string
	retryDuration int
	retryTimes    []byte
	maxReadSize   uint64
}

func NewRetryDB(filePath string, times, retryDuration int, maxReadSize uint64) (*RetryDB, error) {
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
	r.maxReadSize = maxReadSize
	binary.LittleEndian.PutUint16(r.retryTimes, uint16(times))

	if err = db.Update(func(btx *bolt.Tx) error {
		_, err := btx.CreateBucketIfNotExists(BKTRetry)
		if err != nil {
			return err
		}

		_, err = btx.CreateBucketIfNotExists(BKTBtcLastHeight)
		if err != nil {
			return err
		}

		_, err = btx.CreateBucketIfNotExists(BKTAlliaLastHeight)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *RetryDB) setHeight(height uint32, bucket, key []byte) error {
	r.rwlock.Lock()
	defer r.rwlock.Unlock()
	val := make([]byte, 4)
	binary.LittleEndian.PutUint32(val, height)

	return r.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucket)
		err := bucket.Put(key, val)
		if err != nil {
			return err
		}
		return nil
	})
}

func (r *RetryDB) getHeight(bucket, key []byte) uint32 {
	r.rwlock.RLock()
	defer r.rwlock.RUnlock()
	var height uint32
	r.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucket)
		val := bucket.Get(key)
		if val == nil {
			height = 0
			return nil
		}
		height = binary.LittleEndian.Uint32(val)
		return nil
	})

	return height
}

func (r *RetryDB) SetBtcHeight(height uint32) error {
	return r.setHeight(height, BKTBtcLastHeight, KEYBtcLastHeight)
}

func (r *RetryDB) GetBtcHeight() uint32 {
	return r.getHeight(BKTBtcLastHeight, KEYBtcLastHeight)
}

func (r *RetryDB) SetAlliaHeight(height uint32) error {
	return r.setHeight(height, BKTAlliaLastHeight, KEYAlliaLastHeight)
}

func (r *RetryDB) GetAlliaHeight() uint32 {
	return r.getHeight(BKTAlliaLastHeight, KEYAlliaLastHeight)
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
	r.rwlock.Lock()
	defer r.rwlock.Unlock()

	if binary.LittleEndian.Uint16(r.retryTimes) > 0 {
		err = r.db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(BKTRetry)
			valArr := make([]uint16, 0)
			totalSize := uint64(0)
			err := bucket.ForEach(func(k, v []byte) error {
				mtxArr = append(mtxArr, hex.EncodeToString(k))
				valArr = append(valArr, binary.LittleEndian.Uint16(v)-1)
				if totalSize += uint64(len(k)); totalSize > r.maxReadSize {
					return OverReadSizeErr{
						Err: fmt.Errorf("read %d bytes from db, but oversize %d", totalSize, r.maxReadSize),
					}
				}
				return nil
			})
			if err != nil {
				log.Errorf("GetAll, %v", err)
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
		err = r.db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(BKTRetry)
			totalSize := uint64(0)
			err := bucket.ForEach(func(k, v []byte) error {
				mtxArr = append(mtxArr, hex.EncodeToString(k))
				if totalSize += uint64(len(k)); totalSize > r.maxReadSize {
					return OverReadSizeErr{
						Err: fmt.Errorf("read %d bytes from db, but oversize %d", totalSize, r.maxReadSize),
					}
				}
				return nil
			})
			if err != nil {
				log.Errorf("GetAll, %v", err)
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

type OverReadSizeErr struct {
	Err error
}

func (err OverReadSizeErr) Error() string {
	return err.Err.Error()
}
