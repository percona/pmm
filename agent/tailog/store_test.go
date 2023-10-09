package tailog

import (
	"container/ring"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStore_Write(t *testing.T) {

	type expect struct {
		length int
		err    error
	}

	type test struct {
		id       int
		byteArr  []byte
		store    Store
		expected expect
	}

	tests := []test{
		{
			id:      1,
			byteArr: []byte("test1"),
			store:   *NewStore(1),
			expected: expect{
				length: len([]byte("test1")),
				err:    nil,
			},
		},
		{
			id:      2,
			byteArr: []byte("test2"),
			store:   *NewStore(0),
			expected: expect{
				length: len([]byte("test2")),
				err:    nil,
			},
		},
	}

	for _, test := range tests {
		length, err := test.store.Write(test.byteArr)
		assert.Equal(t, nil, err, fmt.Sprintf("Test #%v\n Expected : %v, Got : %v ", test.id,
			test.expected.err, err))
		assert.Equal(t, test.expected.length, length, fmt.Sprintf("Test #%v\n Expected : %v, Got : %v ",
			test.id, test.expected.length, length))
	}
}

func FuzzStore_Write(f *testing.F) {
	b := []byte("hello")
	var capacity uint = 5
	f.Add(b, capacity)
	f.Fuzz(func(t *testing.T, b []byte, capacity uint) {
		store := NewStore(capacity)
		length, err := store.Write(b)

		assert.Equal(t, nil, err, fmt.Sprintf("Expected : %v, Got : %v ", nil, err))
		assert.Equal(t, length, len(b), fmt.Sprintf("Expected : %v, Got : %v ", length, len(b)))
	})
}

func TestStore_Resize(t *testing.T) {

	type test struct {
		id       int
		capacity uint
		store    Store
		expected *ring.Ring
	}

	tests := []test{
		{
			id:       1,
			capacity: 2,
			store: Store{
				capacity: 2,
			},
			expected: nil,
		},
		{
			id:       2,
			capacity: 0,
			store: Store{
				capacity: 2,
			},
			expected: nil,
		},
		{
			id:       3,
			capacity: 3,
			store: Store{
				capacity: 2,
			},
			expected: &ring.Ring{
				Value: nil,
			},
		},
	}

	for _, test := range tests {
		test.store.Resize(test.capacity)
		if test.store.log != nil {
			assert.Equal(t, test.expected.Value, test.store.log.Value, fmt.Sprintf("Test #%v\n"+
				" Expected : %v, Got : %v ", test.id, test.expected.Value, test.store.log.Value))
			continue
		}
		assert.Equal(t, test.expected, test.store.log, fmt.Sprintf("Test #%v\n"+
			" Expected : %v, Got : %v ", test.id, test.expected, test.store.log))
	}
}

func TestStore_GetLogs(t *testing.T) {
	type expect struct {
		logs     []string
		capacity uint
	}

	type test struct {
		id       int
		store    Store
		expected expect
	}

	tests := []test{
		{
			id:    1,
			store: *NewStore(0),
			expected: expect{
				logs:     nil,
				capacity: 0,
			},
		},
		{
			id:    2,
			store: *NewStore(4),
			expected: expect{
				logs:     nil,
				capacity: 4,
			},
		},
	}

	for _, test := range tests {
		logs, capacity := test.store.GetLogs()
		if len(logs) > 0 {
			assert.Equal(t, test.expected.logs[0], logs[0], fmt.Sprintf("Test #%v\n"+
				" Expected : %v, Got : %v ", test.id, test.expected.logs[0], logs[0]))
		}
		assert.Equal(t, test.expected.capacity, capacity, fmt.Sprintf("Test #%v\n"+
			" Expected : %v, Got : %v ", test.id, test.expected.capacity, capacity))

		assert.Equal(t, len(test.expected.logs), len(logs), fmt.Sprintf("Test #%v\n"+
			" Expected : %v, Got : %v ", test.id, len(test.expected.logs), len(logs)))
	}
}
