package main

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/balasanjay/totp"
	"github.com/boltdb/bolt"
	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
)

const (
	stage = "Test"
	tmp   = "./static/tmp"
)

var (
	ps PasswordStore
)

// PasswordStore stores and verifies passwords
// It is also a generic KV store.
type PasswordStore interface {
	Store(id, passwd string) error
	Verify(id, passwd string) (bool, error)
	Put(key, value []byte) error
	Get(key []byte) ([]byte, error)
}

func main() {
	fmt.Println("otpweb")
	idx := template.Must(template.ParseFiles("tmpl/index.html"))
	reg := template.Must(template.ParseFiles("tmpl/register.html"))
	otp := template.Must(template.ParseFiles("tmpl/otp.html"))
	ps = newBoltStore()
	if err := os.MkdirAll(tmp, 0700); err != nil {
		log.Fatalf("could not make tmp folder: %s: %s", tmp, err)
	}

	// handlers
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := idx.Execute(w,
			struct {
				Stage string
			}{
				Stage: stage,
			}); err != nil {
			log.Printf("tpl execute: %v", err)
		}
	})
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("email") != "" {
			msg := register(r)
			if msg != "" {
				fmt.Fprintf(w, msg)
				return
			}
			fmt.Fprintf(w, "registered: %s", r.FormValue("email"))
			return
		}
		if err := reg.Execute(w,
			struct {
				Stage string
			}{
				Stage: stage,
			}); err != nil {
			log.Printf("reg execute: %v", err)
		}
	})
	http.HandleFunc("/otp", func(w http.ResponseWriter, r *http.Request) {
		em := r.FormValue("email")
		if em == "" {
			return
		}
		fn, err := writeQRCode(em)
		if err != nil {
			fmt.Fprintf(w, "email not found: %s", err)
		}
		if err := otp.Execute(w,
			struct {
				Stage  string
				QRCode string
			}{
				Stage:  stage,
				QRCode: fn,
			}); err != nil {
			log.Printf("otp execute: %v", err)
		}
	})
	http.HandleFunc("/otpvldt", func(w http.ResponseWriter, r *http.Request) {
		em := r.FormValue("email")
		otp := r.FormValue("otp")
		sk, err := ps.Get([]byte(em))
		if err != nil {
			fmt.Fprintf(w, "error: %s", err)
			return
		}
		ok := totp.Authenticate(sk, otp, nil)
		if !ok {
			fmt.Fprintf(w, "error: %s incorrect totp", em)
			return
		}
		fmt.Fprintf(w, "%s authenticated", em)
	})
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func writeQRCode(email string) (string, error) {
	tf, err := ioutil.TempFile(tmp, "")
	if err != nil {
		log.Fatalf("could not generate temp file: %s", err)
	}

	sk, err := ps.Get([]byte(email))
	if err != nil {
		return "", fmt.Errorf("error getting secret key: %q %s", email, err)
	}
	qrs := fmt.Sprintf("otpauth://totp/OTPWeb:%s?secret=%s&issuer=OTPWeb",
		email, base32.StdEncoding.EncodeToString(sk))
	if err := qrcode.WriteFile(qrs, qrcode.Medium, 256, tf.Name()); err != nil {
		log.Fatalf("could not write qr-code: %s", err)
	}
	tf.Close()
	// self-destruct in 3 seconds
	go func() {
		time.Sleep(3 * time.Second)
		os.Remove(tf.Name())
	}()
	return tf.Name(), nil
}

func register(r *http.Request) string {
	em := r.FormValue("email")
	chk := r.FormValue("chk-exists")
	if chk == "true" {
		if _, err := ps.Verify(em, ""); err == nil {
			return fmt.Sprintf("error: email: %q was previously registered", em)
		}
	}
	pw := r.FormValue("pw")
	ps.Store(em, pw)
	secKey := generateSecretKey(20)
	ps.Put([]byte(em), secKey)
	log.Printf("registered email: %s, pw len: %d \n", em, len(pw))
	return ""
}

type memStore struct {
	s map[string]string
	k map[string][]byte
}

func newMemStore() *memStore {
	ms := memStore{}
	ms.s = map[string]string{}
	ms.k = map[string][]byte{}
	return &ms
}

func (m *memStore) Store(id, pw string) error {
	m.s[id] = pw
	return nil
}
func (m *memStore) Put(key, value []byte) error {
	m.k[string(key)] = value
	return nil
}

func (m *memStore) Verify(id, pw string) (bool, error) {
	v, ok := m.s[id]
	if !ok {
		return false, fmt.Errorf("id: %s not found", id)
	}
	if v != pw {
		return false, nil
	}
	return true, nil
}

func (m *memStore) Get(key []byte) ([]byte, error) {
	v, ok := m.k[string(key)]
	if !ok {
		return []byte{}, fmt.Errorf("key not found: % x", key)
	}
	return v, nil
}

type boltStore struct {
	db *bolt.DB
}

func newBoltStore() *boltStore {
	var err error
	b := boltStore{}
	b.db, err = bolt.Open("user.db", 0600, nil)
	if err != nil {
		log.Fatalf("init bolt store: %v", err)
	}
	b.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("passwd"))
		if err != nil {
			log.Fatalf("create bucket: %s", err)
		}
		return nil
	})
	b.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("kvstore"))
		if err != nil {
			log.Fatalf("create bucket kvstore: %s", err)
		}
		return nil
	})
	return &b
}
func (b *boltStore) Store(id, pw string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("passwd"))
		h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("pw store: %v", err)
		}
		err = b.Put([]byte(id), h)
		return err
	})
}

func (b *boltStore) Put(key, value []byte) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("kvstore"))
		return b.Put(key, value)
	})
}
func (b *boltStore) Verify(id, pw string) (bool, error) {
	var verified bool
	err := b.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("passwd"))
		v := b.Get([]byte(id))
		if v == nil {
			return fmt.Errorf("bolt DB user not found")
		}
		err := bcrypt.CompareHashAndPassword(v, []byte(pw))
		if err != nil {
			verified = false
			return nil
		}
		verified = true
		return nil
	})
	return verified, err
}
func (b *boltStore) Get(key []byte) ([]byte, error) {
	var ret []byte
	err := b.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("kvstore"))
		v := b.Get(key)
		if v == nil {
			return fmt.Errorf("bolt DB key not found: % x", key)
		}
		ret = make([]byte, len(v), len(v))
		copy(ret, v)
		return nil
	})
	return ret, err
}

func generateSecretKey(l int) []byte {
	k := make([]byte, l, l)
	_, err := rand.Read(k)
	if err != nil {
		log.Fatalf("random key gen: %s", err)
	}
	return k
}
