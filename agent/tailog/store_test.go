package tailog

import (
	"container/ring"
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
		if err != test.expected.err || length != test.expected.length {
			t.Errorf("Test #%v\n Expected : %v,%v, Got : %v,%v ", test.id,
				test.expected.length, nil, length, err)
		}
	}
}

func FuzzStore_Write(f *testing.F) {
	b := []byte("hello")
	var capacity uint = 5
	f.Add(b, capacity)
	f.Fuzz(func(t *testing.T, b []byte, capacity uint) {
		store := NewStore(capacity)
		length, err := store.Write(b)
		if err != nil || length != len(b) {
			t.Errorf("Expected : %v,%v, Got : %v,%v ", len(b), nil, length, err)
		}
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
			if test.store.log.Value != test.expected.Value {
				t.Errorf("Test #%v\n Expected : %v, Got: %v ", test.id, test.expected.Value,
					test.store.log.Value)
			}
			continue
		}
		if test.store.log != test.expected {
			t.Errorf("Test #%v\n Expected : %v, Got: %v ", test.id, test.expected, test.store.log)
		}
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
		if (len(logs) > 0 && logs[0] != test.expected.logs[0]) ||
			capacity != test.expected.capacity {
			t.Errorf("Test #%v\n Expected : %v,%v, Got : %v,%v ", test.id,
				test.expected.logs, test.expected.capacity, logs, capacity)
		}
		if len(logs) != len(test.expected.logs) {
			t.Errorf("Test #%v\n Expected : %v,%v, Got : %v,%v ", test.id,
				test.expected.logs, test.expected.capacity, logs, capacity)
		}
	}
}
