package dockerci

import (
	"errors"
	"github.com/garyburd/redigo/redis"
	"path"
)

const (
	DEFAULT_POOL_SIZE = 10 // Number of redis connections to keep in the pool
	DEFAULT_TIMEOUT   = 0  // Defaults to 0 to block forever
)

var (
	ErrKeyIsAlreadySet = errors.New("key is already set")
)

type Store struct {
	pool *redis.Pool
}

// New returns a new Store with a redis pool for the
// given address
func New(addr, password string) *Store {
	return &Store{
		pool: newPool(addr, password),
	}
}

func (s *Store) Close() error {
	return s.pool.Close()
}

func (s *Store) AtomicSaveState(repository, commit, state string) error {
	isSet, err := redis.Int(s.do("SETNX", stateKey(repository, commit), state))
	if err != nil {
		return err
	}
	if isSet == 0 {
		return ErrKeyIsAlreadySet
	}
	return nil
}

func (s *Store) SaveState(repository, commit, state string) error {
	if _, err := s.do("SET", stateKey(repository, commit), state); err != nil {
		return err
	}
	return nil
}

func (s *Store) SaveOutput(repository, commit string, output []byte) error {
	if _, err := s.do("SET", outputKey(repository, commit), output); err != nil {
		return err
	}
	return nil
}

func (s *Store) do(cmd string, args ...interface{}) (interface{}, error) {
	conn := s.pool.Get()
	defer conn.Close()
	return conn.Do(cmd, args...)
}

func newPool(addr, password string) *redis.Pool {
	return redis.NewPool(func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", addr)
		if err != nil {
			return nil, err
		}
		if password != "" {
			if _, err := c.Do("AUTH", password); err != nil {
				return nil, err
			}
		}
		return c, nil
	}, DEFAULT_POOL_SIZE)
}

func stateKey(repository, commit string) string {
	return path.Join("/dockerci", repository, "commit", commit, "state")
}

func outputKey(repository, commit string) string {
	return path.Join("/dockerci", repository, "commit", commit, "output")
}
