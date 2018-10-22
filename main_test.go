package main

import "testing"

var m PasswordStore

func TestMemStore(t *testing.T) {
	dat := []struct {
		id, pw, vid, vpw string
		o                bool
	}{
		{id: "a", pw: "abc", vid: "a", vpw: "abc", o: true},
		{id: "a@b.c", pw: "abc", vid: "a@b.c", vpw: "abc", o: true},
		{id: "a", pw: "abc", vid: "b", vpw: "abc", o: false},
		{id: "a", pw: "abc", vid: "a", vpw: "acb", o: false},
	}
	//m = newMemStore()
	m = newBoltStore()
	for i, d := range dat {
		m.Store(d.id, d.pw)
		if res, _ := m.Verify(d.vid, d.vpw); res != d.o {
			t.Errorf("case %d: in %q, %q, %q, %q, %v: got: %v", i,
				d.id, d.pw, d.vid, d.vpw, d.o, res)
		}
	}
}

func TestBoltKVStore(t *testing.T) {
	if err := m.Put([]byte("key1"), []byte("value1")); err != nil {
		t.Errorf("put: %s", err)
	}
	v, err := m.Get([]byte("key1"))
	if err != nil {
		t.Errorf("get: %v", err)
	}
	if string(v) != "value1" {
		t.Errorf("did not retrieve value: %s", v)
	}
}

func TestUserAvailable(t *testing.T) {
	_, err := m.Verify("this_user_should_not_be_already_taken@example.com", "passwordDoesNotMatter")
	if err == nil {
		t.Error("User should be available")
	}
}

func TestGenSecretKey(t *testing.T) {
	//k := generateSecretKey(20) // 20 bytes long = 160 bits
	//t.Errorf("%x", k)
}
