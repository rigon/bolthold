// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package bolthold_test

import (
	"testing"

	bh "github.com/timshannon/bolthold"
	bolt "go.etcd.io/bbolt"
)

func TestIndexSlice(t *testing.T) {
	testWrap(t, func(store *bh.Store, t *testing.T) {
		var testData = []ItemTest{
			{
				Key:  0,
				Name: "John",
				Tags: []string{"red", "green", "blue"},
			},
			{
				Key:  1,
				Name: "Bill",
				Tags: []string{"red", "purple"},
			},
			{
				Key:  2,
				Name: "Jane",
				Tags: []string{"red", "orange"},
			},
			{
				Key:  3,
				Name: "Brian",
				Tags: []string{"red", "purple"},
			},
		}

		for _, data := range testData {
			ok(t, store.Insert(data.Key, data))
		}

		b := store.Bolt()

		ok(t, b.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte("_index:ItemTest:Tags"))
			assert(t, bucket != nil, "No index bucket found for Tags index")

			indexCount := 0
			bucket.ForEach(func(k, v []byte) error {
				indexCount++
				return nil
			})

			// each tag chould be indexed individually and there are 5 different tags
			equals(t, indexCount, 5)
			return nil
		}))

	})
}

func Test85SliceIndex(t *testing.T) {
	type Event struct {
		Id         uint64
		Type       string   `boltholdIndex:"Type"`
		Categories []string `boltholdSliceIndex:"Categories"`
	}

	testWrap(t, func(store *bh.Store, t *testing.T) {
		e1 := &Event{Id: 1, Type: "Type1", Categories: []string{"Cat 1", "Cat 2"}}
		e2 := &Event{Id: 2, Type: "Type1", Categories: []string{"Cat 3"}}

		ok(t, store.Insert(e1.Id, e1))
		ok(t, store.Insert(e2.Id, e2))

		var es []*Event
		ok(t, store.Find(&es, bh.Where("Categories").Contains("Cat 1").Index("Categories")))
		equals(t, len(es), 1)
	})
}

func Test87SliceIndex(t *testing.T) {
	type Event struct {
		Id         uint64
		Type       string   `boltholdIndex:"Type"`
		Categories []string `boltholdSliceIndex:"Categories"`
	}

	testWrap(t, func(store *bh.Store, t *testing.T) {
		e1 := &Event{Id: 1, Type: "Type1", Categories: []string{"Cat 1", "Cat 2"}}
		e2 := &Event{Id: 2, Type: "Type1", Categories: []string{"Cat 3"}}

		ok(t, store.Insert(e1.Id, e1))
		ok(t, store.Insert(e2.Id, e2))
		var es []*Event
		ok(t, store.Find(&es, bh.Where("Categories").ContainsAny("Cat 1").Index("Categories")))
		equals(t, len(es), 1)
	})
}

func TestSliceIndexWithPointers(t *testing.T) {
	type Event struct {
		Id         uint64
		Type       string    `boltholdIndex:"Type"`
		Categories []*string `boltholdSliceIndex:"Categories"`
	}

	testWrap(t, func(store *bh.Store, t *testing.T) {
		cat1 := "Cat 1"
		cat2 := "Cat 2"
		cat3 := "Cat 3"

		e1 := &Event{Id: 1, Type: "Type1", Categories: []*string{&cat1, &cat2}}
		e2 := &Event{Id: 2, Type: "Type1", Categories: []*string{&cat3}}

		ok(t, store.Insert(e1.Id, e1))
		ok(t, store.Insert(e2.Id, e2))

		var es []*Event
		ok(t, store.Find(&es, bh.Where("Categories").ContainsAll("Cat 1").Index("Categories")))
		equals(t, len(es), 1)
	})
}

type indexItem struct {
	ID       int
	Category string `boltholdIndex:"Category"`
}

const indexItemBucket = "_index:indexItem:Category"

// indexItems returns the record keys stored on disk under the index value for category
func indexItems(t *testing.T, store *bh.Store, category string) [][]byte {
	indexKey, err := bh.DefaultEncode(category)
	ok(t, err)

	var keys [][]byte
	ok(t, store.Bolt().View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(indexItemBucket)).Bucket(indexKey)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, _ []byte) error {
			keys = append(keys, append([]byte(nil), k...))
			return nil
		})
	}))
	return keys
}

func TestIndexStoresKeysAsBucketItems(t *testing.T) {
	testWrap(t, func(store *bh.Store, t *testing.T) {
		ok(t, store.Insert(1, indexItem{ID: 1, Category: "a"}))
		ok(t, store.Insert(2, indexItem{ID: 2, Category: "a"}))
		ok(t, store.Insert(3, indexItem{ID: 3, Category: "b"}))

		equals(t, len(indexItems(t, store, "a")), 2)
		equals(t, len(indexItems(t, store, "b")), 1)

		var found []indexItem
		ok(t, store.Find(&found, bh.Where("Category").Eq("a").Index("Category")))
		equals(t, len(found), 2)

		ok(t, store.Update(2, indexItem{ID: 2, Category: "b"}))
		equals(t, len(indexItems(t, store, "a")), 1)
		equals(t, len(indexItems(t, store, "b")), 2)

		// the last key removes the index value entirely
		ok(t, store.Delete(1, indexItem{}))
		equals(t, len(indexItems(t, store, "a")), 0)

		found = nil
		ok(t, store.Find(&found, bh.Where("Category").Eq("a").Index("Category")))
		equals(t, len(found), 0)
	})
}
